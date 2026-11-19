package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
)

// testClientVersion is what the app sends in X-Client-Version.
const testClientVersion = "0.1.0"

// testDeviceID is the installation the test suite signs in from.
const testDeviceID = "11111111-1111-4111-8111-111111111111"

type env struct {
	server    *httptest.Server
	mediaDir  string
	private   *rsa.PrivateKey
	publicPem string
}

// TestE2E covers the phase 2 scenarios against the real router, PostgreSQL and
// Valkey.
func TestE2E(t *testing.T) {
	e := startEnv(t)

	t.Run("register via invite, RSA auth and refresh", func(t *testing.T) {
		user := e.register(t, "authuser")
		tokens := e.authenticate(t, user)
		if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.ServerVersion == "" {
			t.Fatal("auth returned empty tokens or version")
		}

		var refreshed models.RefreshResponse
		status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: tokens.RefreshToken}, "", &refreshed)
		if status != http.StatusOK || refreshed.AccessToken == "" {
			t.Fatalf("refresh: status %d, token %q", status, refreshed.AccessToken)
		}
	})

	t.Run("the server reports its version and the oldest client", func(t *testing.T) {
		var version models.VersionResponse
		if status := e.doJSON(t, http.MethodGet, "/api/version", nil, "", &version); status != http.StatusOK {
			t.Fatalf("version status: %d", status)
		}
		if version.Name != "whisper-server" || version.Version == "" || version.MinClientVersion == "" {
			t.Fatalf("unexpected version: %+v", version)
		}
	})

	t.Run("a client older than the minimum is refused", func(t *testing.T) {
		previous := utils.GetSingleton().Config.MinClientVersion
		utils.GetSingleton().Config.MinClientVersion = "0.2.0"
		defer func() { utils.GetSingleton().Config.MinClientVersion = previous }()

		user := e.register(t, "oldclient")
		signed := e.loginToken(t, user, e.authChallenge(t), testDeviceID)
		status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{
			Token: signed, DeviceId: testDeviceID, Label: "test", Platform: "linux",
		}, "", nil)
		if status != http.StatusUpgradeRequired {
			t.Fatalf("old client login status: %d, want %d", status, http.StatusUpgradeRequired)
		}
	})

	t.Run("a refresh token is single use and rotates", func(t *testing.T) {
		user := e.register(t, "rotateuser")
		tokens := e.authenticate(t, user)

		var first models.RefreshResponse
		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: tokens.RefreshToken}, "", &first); status != http.StatusOK {
			t.Fatalf("first refresh status: %d", status)
		}
		if first.RefreshToken == "" || first.RefreshToken == tokens.RefreshToken {
			t.Fatal("refresh token was not rotated")
		}
		// The spent token must not work again.
		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: tokens.RefreshToken}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("replayed refresh token status: %d, want %d", status, http.StatusUnauthorized)
		}
		// The rotated one does.
		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: first.RefreshToken}, "", nil); status != http.StatusOK {
			t.Fatalf("rotated refresh token status: %d", status)
		}
	})

	t.Run("a message with an implausible timestamp is refused", func(t *testing.T) {
		sender := e.register(t, "skewsender")
		receiver := e.register(t, "skewreceiver")
		conn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = conn.Close() }()

		payload := json.RawMessage(fmt.Sprintf(`{"receiverId":%q,"content":"x","key":"k","nonce":"n","mac":"m","sentAt":%q}`,
			receiver.Id, time.Now().Add(48*time.Hour).UTC().Format(time.RFC3339Nano)))
		message, err := json.Marshal(models.WsMessage{Type: models.WsMessageTopicMessage, Payload: payload})
		if err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
			t.Fatal(err)
		}
		if reply := e.readError(t, conn); reply == "" {
			t.Fatal("no error reported for an implausible timestamp")
		}
	})

	t.Run("a user has a media quota", func(t *testing.T) {
		previous := utils.GetSingleton().Config.MediaMaxPerUser
		utils.GetSingleton().Config.MediaMaxPerUser = 2
		defer func() { utils.GetSingleton().Config.MediaMaxPerUser = previous }()

		sender := e.register(t, "quotasender")
		receiver := e.register(t, "quotareceiver")
		senderTokens := e.authenticate(t, sender)

		for i := 0; i < 2; i++ {
			e.uploadMedia(t, senderTokens.AccessToken, receiver.Id, []byte("blob"))
		}
		if status := e.uploadMediaStatus(t, senderTokens.AccessToken, receiver.Id, []byte("blob")); status != http.StatusTooManyRequests {
			t.Fatalf("upload above the quota status: %d, want %d", status, http.StatusTooManyRequests)
		}
	})

	t.Run("the login token is single use", func(t *testing.T) {
		user := e.register(t, "singletoken")
		signed := e.loginToken(t, user, e.authChallenge(t), testDeviceID)
		request := models.Auth{Token: signed, DeviceId: testDeviceID, Label: "test", Platform: "linux"}

		if status := e.doJSON(t, http.MethodPost, "/api/auth", request, "", nil); status != http.StatusOK {
			t.Fatalf("first login status: %d", status)
		}
		// Replaying the same token (captured on the wire) must not work.
		if status := e.doJSON(t, http.MethodPost, "/api/auth", request, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("replayed login status: %d, want %d", status, http.StatusUnauthorized)
		}
		// A token without any challenge is refused as well.
		withoutChallenge := models.Auth{
			Token:    e.loginToken(t, user, "", testDeviceID),
			DeviceId: testDeviceID, Label: "test", Platform: "linux",
		}
		if status := e.doJSON(t, http.MethodPost, "/api/auth", withoutChallenge, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("token without challenge status: %d, want %d", status, http.StatusUnauthorized)
		}
		// A challenge the server never issued is refused too.
		unknownChallenge := models.Auth{
			Token:    e.loginToken(t, user, "made-up-challenge", testDeviceID),
			DeviceId: testDeviceID, Label: "test", Platform: "linux",
		}
		if status := e.doJSON(t, http.MethodPost, "/api/auth", unknownChallenge, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("unknown challenge status: %d, want %d", status, http.StatusUnauthorized)
		}
	})

	t.Run("the signed token must name the device from the request", func(t *testing.T) {
		user := e.register(t, "didswap")
		signed := e.loginToken(t, user, e.authChallenge(t), "22222222-2222-4222-8222-222222222222")
		status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{
			Token: signed, DeviceId: testDeviceID, Label: "test", Platform: "linux",
		}, "", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("login with a swapped device id: %d, want %d", status, http.StatusUnauthorized)
		}
	})

	t.Run("a device id of another account is refused", func(t *testing.T) {
		first := e.register(t, "deviceowner")
		second := e.register(t, "deviceintruder")
		ownerDeviceId := uuid.NewString()
		e.authenticateDevice(t, first, ownerDeviceId, "owner")

		signed := e.loginToken(t, second, e.authChallenge(t), ownerDeviceId)
		status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{
			Token: signed, DeviceId: ownerDeviceId, Label: "test", Platform: "linux",
		}, "", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("login with a foreign device id: %d, want %d", status, http.StatusUnauthorized)
		}
	})

	t.Run("a revoked device cannot log in, call the API or keep its socket", func(t *testing.T) {
		user := e.register(t, "revokedevice")
		phoneID := uuid.NewString()
		phone := e.authenticateDevice(t, user, phoneID, "Pixel 8")
		laptopID := uuid.NewString()
		laptop := e.authenticateDevice(t, user, laptopID, "Linux")

		// Both devices are listed, the caller flagged.
		var devices models.DevicesResponse
		if status := e.doJSON(t, http.MethodGet, "/api/user/devices", nil, laptop.AccessToken, &devices); status != http.StatusOK {
			t.Fatalf("devices status: %d", status)
		}
		if len(devices.Devices) != 2 {
			t.Fatalf("device count = %d, want 2", len(devices.Devices))
		}
		current := 0
		for _, device := range devices.Devices {
			if device.Current {
				current++
				if device.Id != laptopID {
					t.Fatalf("current device = %s, want %s", device.Id, laptopID)
				}
			}
		}
		if current != 1 {
			t.Fatalf("current devices = %d, want 1", current)
		}

		// A live socket of the revoked device, to prove it is dropped.
		phoneConn := e.dialWS(t, e.wsToken(t, phone.AccessToken))
		defer func() { _ = phoneConn.Close() }()

		if status := e.doJSON(t, http.MethodPost, "/api/user/devices/"+phoneID+"/revoke", nil, laptop.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("revoke status: %d", status)
		}

		// The unexpired access token stops working immediately.
		if status := e.doJSON(t, http.MethodGet, "/api/user/devices", nil, phone.AccessToken, nil); status != http.StatusUnauthorized {
			t.Fatalf("revoked device API status: %d, want %d", status, http.StatusUnauthorized)
		}
		// Its refresh session is gone.
		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: phone.RefreshToken}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("revoked device refresh status: %d, want %d", status, http.StatusUnauthorized)
		}
		// And the account key is not enough to sign in again.
		signed := e.loginToken(t, user, e.authChallenge(t), phoneID)
		if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{
			Token: signed, DeviceId: phoneID, Label: "Pixel 8", Platform: "android",
		}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("revoked device login status: %d, want %d", status, http.StatusUnauthorized)
		}

		// The other device keeps working and can let it back in.
		if status := e.doJSON(t, http.MethodGet, "/api/user/devices", nil, laptop.AccessToken, nil); status != http.StatusOK {
			t.Fatalf("paired device status after revoke: %d", status)
		}
		if status := e.doJSON(t, http.MethodPost, "/api/user/devices/"+phoneID+"/unrevoke", nil, laptop.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("unrevoke status: %d", status)
		}
		signed = e.loginToken(t, user, e.authChallenge(t), phoneID)
		if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{
			Token: signed, DeviceId: phoneID, Label: "Pixel 8", Platform: "android",
		}, "", nil); status != http.StatusOK {
			t.Fatalf("unrevoked device login status: %d, want %d", status, http.StatusOK)
		}
	})

	t.Run("a refresh token without a session id is refused", func(t *testing.T) {
		// Tokens issued before sessions were recorded cannot be revoked, so the
		// server refuses them and the client logs in again with its RSA key.
		user := e.register(t, "legacytoken")
		legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": user.Id,
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		signed, err := legacy.SignedString([]byte(utils.GetSingleton().Config.RefreshTokenSecret))
		if err != nil {
			t.Fatal(err)
		}

		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: signed}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("legacy refresh token status: %d, want %d", status, http.StatusUnauthorized)
		}
	})

	t.Run("a revoked refresh session is refused", func(t *testing.T) {
		user := e.register(t, "revokeduser")
		tokens := e.authenticate(t, user)

		if err := database.DeleteRefreshSessions(user.Id); err != nil {
			t.Fatal(err)
		}
		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: tokens.RefreshToken}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("revoked refresh token status: %d, want %d", status, http.StatusUnauthorized)
		}
	})

	t.Run("WS handshake consumes the one-time token", func(t *testing.T) {
		user := e.register(t, "wsuser")
		tokens := e.authenticate(t, user)
		wsToken := e.wsToken(t, tokens.AccessToken)

		conn := e.dialWS(t, wsToken)
		_ = conn.Close()

		// The token was removed from Valkey by the first handshake.
		second, err := e.dial(wsToken)
		if err == nil {
			_ = second.Close()
			t.Fatal("second handshake with the same token succeeded")
		}
	})

	t.Run("online message delivery", func(t *testing.T) {
		sender := e.register(t, "senderonline")
		receiver := e.register(t, "receiveronline")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		receiverConn := e.dialWS(t, e.wsToken(t, receiverTokens.AccessToken))
		defer func() { _ = receiverConn.Close() }()
		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"online-ciphertext"`)
		message := e.readMessage(t, receiverConn)
		if message.SenderId != sender.Id || message.RecipientOnline != true {
			t.Fatalf("unexpected message: sender %s, online %t", message.SenderId, message.RecipientOnline)
		}
		if message.Seq == 0 || message.PeerId != sender.Id {
			t.Fatalf("message without seq or peer: %+v", message)
		}
	})

	t.Run("an offline message arrives with the delta sync", func(t *testing.T) {
		sender := e.register(t, "senderoffline")
		receiver := e.register(t, "receiveroffline")
		senderConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"offline-ciphertext"`)
		// The sender's confirmation means the row is in the archive.
		e.readAck(t, senderConn)

		receiverConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		defer func() { _ = receiverConn.Close() }()
		synced := e.sync(t, receiverConn, 0)
		if len(synced.Messages) != 1 {
			t.Fatalf("synced %d messages, want 1", len(synced.Messages))
		}
		message := synced.Messages[0]
		if message.SenderId != sender.Id || message.RecipientOnline {
			t.Fatalf("unexpected synced message: sender %s, online %t", message.SenderId, message.RecipientOnline)
		}
		if string(message.Content) != `"offline-ciphertext"` {
			t.Fatalf("unexpected content: %s", message.Content)
		}
		if message.PeerId != sender.Id {
			t.Fatalf("peer = %s, want %s", message.PeerId, sender.Id)
		}
		if len(synced.Peers) != 1 || synced.Peers[0].Id != sender.Id {
			t.Fatalf("peers = %+v, want the sender", synced.Peers)
		}
	})

	t.Run("the delta sync returns only what the cursor missed", func(t *testing.T) {
		sender := e.register(t, "deltasender")
		receiver := e.register(t, "deltareceiver")
		senderConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"first"`)
		e.readAck(t, senderConn)
		e.sendMessage(t, senderConn, receiver.Id, `"second"`)
		e.readAck(t, senderConn)

		receiverConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		defer func() { _ = receiverConn.Close() }()

		first := e.sync(t, receiverConn, 0)
		if len(first.Messages) != 2 || first.HasMore {
			t.Fatalf("first sync = %d messages, more %t", len(first.Messages), first.HasMore)
		}
		// The same cursor returns nothing: the client stored both.
		second := e.sync(t, receiverConn, first.SyncedThrough)
		if len(second.Messages) != 0 {
			t.Fatalf("second sync returned %d messages, want 0", len(second.Messages))
		}
		if second.SyncedThrough != first.SyncedThrough {
			t.Fatalf("cursor moved without new messages: %d -> %d", first.SyncedThrough, second.SyncedThrough)
		}

		// A new message after the cursor is the only one delivered.
		e.sendMessage(t, senderConn, receiver.Id, `"third"`)
		e.readAck(t, senderConn)
		third := e.sync(t, receiverConn, first.SyncedThrough)
		if len(third.Messages) != 1 || string(third.Messages[0].Content) != `"third"` {
			t.Fatalf("delta sync returned %+v", third.Messages)
		}
	})

	t.Run("a resent message is archived once", func(t *testing.T) {
		sender := e.register(t, "resendsender")
		receiver := e.register(t, "resendreceiver")
		senderConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn.Close() }()

		sentAt := time.Now().UTC()
		payload := json.RawMessage(fmt.Sprintf(`{"receiverId":%q,"content":"dup","key":"k","nonce":"n","mac":"m","sentAt":%q}`,
			receiver.Id, sentAt.Format(time.RFC3339Nano)))
		message, err := json.Marshal(models.WsMessage{Type: models.WsMessageTopicMessage, Payload: payload})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			if err := senderConn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				t.Fatal(err)
			}
			e.readAck(t, senderConn)
		}

		messages, err := database.SyncSince(receiver.Id, 0, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 1 {
			t.Fatalf("archive holds %d copies, want 1", len(messages))
		}
	})

	t.Run("history is paginated newest first", func(t *testing.T) {
		sender := e.register(t, "historysender")
		receiver := e.register(t, "historyreceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()
		for i := 0; i < 3; i++ {
			e.sendMessage(t, senderConn, receiver.Id, fmt.Sprintf("%q", fmt.Sprintf("m%d", i)))
			e.readAck(t, senderConn)
		}

		var page models.HistoryPage
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+sender.Id+"&limit=2", nil, receiverTokens.AccessToken, &page); status != http.StatusOK {
			t.Fatalf("history status: %d", status)
		}
		if len(page.Messages) != 2 || page.NextCursor == "" {
			t.Fatalf("first page = %d messages, cursor %q", len(page.Messages), page.NextCursor)
		}
		if string(page.Messages[0].Content) != `"m2"` || string(page.Messages[1].Content) != `"m1"` {
			t.Fatalf("unexpected page order: %s, %s", page.Messages[0].Content, page.Messages[1].Content)
		}
		if len(page.Peers) != 1 || page.Peers[0].Id != sender.Id {
			t.Fatalf("peers = %+v", page.Peers)
		}

		var older models.HistoryPage
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+sender.Id+"&limit=2&before="+url.QueryEscape(page.NextCursor), nil, receiverTokens.AccessToken, &older); status != http.StatusOK {
			t.Fatalf("second page status: %d", status)
		}
		if len(older.Messages) != 1 || string(older.Messages[0].Content) != `"m0"` {
			t.Fatalf("second page = %+v", older.Messages)
		}
		if older.NextCursor != "" {
			t.Fatalf("unexpected cursor on the last page: %q", older.NextCursor)
		}

		// A malformed cursor is a client bug, not a server error.
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+sender.Id+"&before=nonsense", nil, receiverTokens.AccessToken, nil); status != http.StatusBadRequest {
			t.Fatalf("malformed cursor status: %d", status)
		}
	})

	t.Run("the archive retention is configurable and validated", func(t *testing.T) {
		user := e.register(t, "retentionuser")
		tokens := e.authenticate(t, user)

		var retention models.RetentionResponse
		if status := e.doJSON(t, http.MethodGet, "/api/user/retention", nil, tokens.AccessToken, &retention); status != http.StatusOK {
			t.Fatalf("retention status: %d", status)
		}
		if retention.Days != models.DefaultRetentionDays {
			t.Fatalf("default retention = %d, want %d", retention.Days, models.DefaultRetentionDays)
		}

		if status := e.doJSON(t, http.MethodPut, "/api/user/retention", models.RetentionRequest{Days: 7}, tokens.AccessToken, nil); status != http.StatusOK {
			t.Fatalf("set retention status: %d", status)
		}
		if status := e.doJSON(t, http.MethodPut, "/api/user/retention", models.RetentionRequest{Days: 2}, tokens.AccessToken, nil); status != http.StatusBadRequest {
			t.Fatalf("invalid retention status: %d, want %d", status, http.StatusBadRequest)
		}
	})

	t.Run("an expired message leaves the archive", func(t *testing.T) {
		sender := e.register(t, "expiresender")
		receiver := e.register(t, "expirereceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()
		e.sendMessage(t, senderConn, receiver.Id, `"expiring"`)
		e.readAck(t, senderConn)

		// Age the row instead of waiting 7 days.
		if err := utils.GetSingleton().Db.Exec(
			`UPDATE messages SET expires_at = ? WHERE receiver_id = ?`, time.Now().Add(-time.Hour), receiver.Id).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := database.PruneExpired(100); err != nil {
			t.Fatal(err)
		}

		var page models.HistoryPage
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+sender.Id, nil, receiverTokens.AccessToken, &page); status != http.StatusOK {
			t.Fatalf("history status: %d", status)
		}
		if len(page.Messages) != 0 {
			t.Fatalf("expired message is still in the archive: %+v", page.Messages)
		}
	})

	t.Run("media upload and download keep only ciphertext", func(t *testing.T) {
		sender := e.register(t, "mediasender")
		receiver := e.register(t, "mediareceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		ciphertext := make([]byte, 4096)
		if _, err := rand.Read(ciphertext); err != nil {
			t.Fatal(err)
		}
		mediaID := e.uploadMedia(t, senderTokens.AccessToken, receiver.Id, ciphertext)

		stored, err := os.ReadFile(filepath.Join(e.mediaDir, mediaID)) //nolint:gosec // server-generated id inside the test media dir
		if err != nil {
			t.Fatalf("read stored media: %v", err)
		}
		if !bytes.Equal(stored, ciphertext) {
			t.Fatal("stored media differs from uploaded ciphertext")
		}

		downloaded := e.downloadMedia(t, receiverTokens.AccessToken, mediaID)
		if !bytes.Equal(downloaded, ciphertext) {
			t.Fatal("downloaded media differs from uploaded ciphertext")
		}
		if status := e.doJSON(t, http.MethodGet, "/api/media/"+mediaID, nil, senderTokens.AccessToken, nil); status != http.StatusForbidden {
			t.Fatalf("sender download status: %d", status)
		}

		if status := e.doJSON(t, http.MethodDelete, "/api/media/"+mediaID, nil, receiverTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("confirm status: %d", status)
		}
		if status := e.doJSON(t, http.MethodGet, "/api/media/"+mediaID, nil, receiverTokens.AccessToken, nil); status != http.StatusNotFound {
			t.Fatalf("download after confirm status: %d", status)
		}
	})

	t.Run("sweeper removes media whose expiry notification was missed", func(t *testing.T) {
		sender := e.register(t, "sweepsender")
		receiver := e.register(t, "sweepreceiver")
		senderTokens := e.authenticate(t, sender)

		liveID := e.uploadMedia(t, senderTokens.AccessToken, receiver.Id, []byte("live"))
		orphanID := e.uploadMedia(t, senderTokens.AccessToken, receiver.Id, []byte("orphan"))

		// Simulate an expiry notification that never reached the subscriber
		client := utils.GetSingleton().Valkey
		if err := client.Do(context.Background(), client.B().Del().Key("media:"+orphanID).Build()).Error(); err != nil {
			t.Fatal(err)
		}

		removed, err := database.SweepOrphanedMedia()
		if err != nil {
			t.Fatalf("sweep: %v", err)
		}
		if removed != 1 {
			t.Fatalf("sweep removed %d files, want 1", removed)
		}
		if _, err := os.Stat(filepath.Join(e.mediaDir, orphanID)); !os.IsNotExist(err) {
			t.Fatalf("orphaned file still on disk: %v", err)
		}
		if _, err := os.Stat(filepath.Join(e.mediaDir, liveID)); err != nil {
			t.Fatalf("live media file was removed: %v", err)
		}
	})

	t.Run("an invite code can only be used once even under a race", func(t *testing.T) {
		code, err := utils.RandomASCIIString(64)
		if err != nil {
			t.Fatal(err)
		}
		if err := database.PushInvite(code, *models.NewInvite("race@example.com", false, time.Now().Add(time.Hour)), 3600); err != nil {
			t.Fatal(err)
		}

		// Two registrations with the same code at the same time: the second must
		// not create an account.
		statuses := make(chan int, 2)
		for _, username := range []string{"raceone", "racetwo"} {
			username := username
			go func() {
				statuses <- e.doJSON(t, http.MethodPost, "/api/auth/register", models.Register{
					InviteCode: code,
					Username:   username,
					PublicKey:  e.publicPem,
				}, "", nil)
			}()
		}
		first, second := <-statuses, <-statuses
		okCount := 0
		for _, status := range []int{first, second} {
			switch status {
			case http.StatusOK:
				okCount++
			case http.StatusUnauthorized:
			default:
				t.Fatalf("unexpected register status: %d", status)
			}
		}
		if okCount != 1 {
			t.Fatalf("invite created %d accounts, want 1 (statuses %d, %d)", okCount, first, second)
		}
	})

	t.Run("a deleted user loses API access before the next connect", func(t *testing.T) {
		admin := e.registerAdmin(t, "deleteadmin")
		victim := e.register(t, "deletevictim")
		adminTokens := e.authenticate(t, admin)
		victimTokens := e.authenticateDevice(t, victim, "44444444-4444-4444-8444-444444444444", "victim")

		// The victim is offline: deletion is only applied on WS connect, so the
		// API must already refuse the token.
		if status := e.doJSON(t, http.MethodPatch, "/api/user", models.IdsRequest{Ids: []string{victim.Id}}, adminTokens.AccessToken, nil); status != http.StatusOK {
			t.Fatalf("delete status: %d", status)
		}
		if status := e.doJSON(t, http.MethodGet, "/api/user?id="+victim.Id, nil, victimTokens.AccessToken, nil); status != http.StatusUnauthorized {
			t.Fatalf("deleted user still reached the API: %d", status)
		}
		// And the refresh session is revoked with the account.
		if status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: victimTokens.RefreshToken}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("deleted user refreshed its token: %d", status)
		}
	})

	t.Run("an oversized upload is rejected before it is stored", func(t *testing.T) {
		sender := e.register(t, "bigsender")
		receiver := e.register(t, "bigreceiver")
		senderTokens := e.authenticate(t, sender)

		limit := utils.GetSingleton().Config.MediaMaxUploadSize
		before := mediaFiles(t, e.mediaDir)
		body := bytes.Repeat([]byte("a"), int(limit)+1024)
		status := e.uploadMediaStatus(t, senderTokens.AccessToken, receiver.Id, body)
		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversized upload status: %d", status)
		}
		// Nothing new may be stored; only the abandoned upload temp file may
		// linger until the sweeper removes it.
		for name := range mediaFiles(t, e.mediaDir) {
			if before[name] {
				continue
			}
			if !strings.HasSuffix(name, database.MediaUploadTempSuffix) {
				t.Fatalf("oversized upload was stored as %s", name)
			}
		}
	})

	t.Run("an edited message reaches the peer and the log", func(t *testing.T) {
		sender := e.register(t, "editsender")
		receiver := e.register(t, "editreceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		receiverConn := e.dialWS(t, e.wsToken(t, receiverTokens.AccessToken))
		defer func() { _ = receiverConn.Close() }()
		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"before"`)
		ack := e.readAck(t, senderConn)
		e.readMessage(t, receiverConn)

		edit := map[string]json.RawMessage{
			"content": json.RawMessage(`"after"`),
			"key":     json.RawMessage(`"k2"`),
			"nonce":   json.RawMessage(`"n2"`),
			"mac":     json.RawMessage(`"m2"`),
		}
		var edited models.EditMessageResponse
		if status := e.doJSON(t, http.MethodPatch, "/api/messages/"+ack.Id, edit, senderTokens.AccessToken, &edited); status != http.StatusOK {
			t.Fatalf("edit status: %d", status)
		}
		if edited.EditedAt.IsZero() {
			t.Fatal("edit did not answer with an instant")
		}
		event := e.readEvent(t, receiverConn, models.WsResponseTypeEdit)
		if event.Kind != models.MessageEventEdit || event.MessageId != ack.Id {
			t.Fatalf("unexpected edit event: %+v", event)
		}
		if string(event.Content) != `"after"` || event.PeerId != sender.Id || event.V != 1 {
			t.Fatalf("unexpected edit payload: %+v", event)
		}

		// Only the sender may rewrite a message.
		if status := e.doJSON(t, http.MethodPatch, "/api/messages/"+ack.Id, edit, receiverTokens.AccessToken, nil); status != http.StatusForbidden {
			t.Fatalf("peer edit status: %d", status)
		}

		var page models.HistoryPage
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+sender.Id, nil, receiverTokens.AccessToken, &page); status != http.StatusOK {
			t.Fatalf("history status: %d", status)
		}
		if len(page.Messages) != 1 || string(page.Messages[0].Content) != `"after"` || page.Messages[0].EditedAt == nil {
			t.Fatalf("history kept the old version: %+v", page.Messages)
		}
	})

	t.Run("delete for everyone is idempotent and reaches the peer", func(t *testing.T) {
		sender := e.register(t, "deletesender")
		receiver := e.register(t, "deletereceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		receiverConn := e.dialWS(t, e.wsToken(t, receiverTokens.AccessToken))
		defer func() { _ = receiverConn.Close() }()
		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"gone"`)
		ack := e.readAck(t, senderConn)
		e.readMessage(t, receiverConn)

		for i := 0; i < 2; i++ {
			if status := e.doJSON(t, http.MethodDelete, "/api/messages/"+ack.Id, nil, senderTokens.AccessToken, nil); status != http.StatusNoContent {
				t.Fatalf("delete %d status: %d", i, status)
			}
		}
		event := e.readEvent(t, receiverConn, models.WsResponseTypeDelete)
		if event.Kind != models.MessageEventDelete || event.MessageId != ack.Id {
			t.Fatalf("unexpected delete event: %+v", event)
		}

		// The peer cannot delete somebody else's message, and a message retention
		// already swept away is a no-op rather than a failure.
		if status := e.doJSON(t, http.MethodDelete, "/api/messages/"+ack.Id, nil, receiverTokens.AccessToken, nil); status != http.StatusForbidden {
			t.Fatalf("peer delete status: %d", status)
		}
		if status := e.doJSON(t, http.MethodDelete, "/api/messages/"+uuid.NewString(), nil, senderTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("unknown message delete status: %d", status)
		}

		// The duplicate delete must not append a second log entry.
		receiverConn2 := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		defer func() { _ = receiverConn2.Close() }()
		synced := e.syncEvents(t, receiverConn2, 0, 0)
		if len(synced.Events) != 1 || synced.Events[0].Kind != models.MessageEventDelete {
			t.Fatalf("event log holds %+v", synced.Events)
		}
		if len(synced.Messages) != 1 || synced.Messages[0].DeletedAt == nil || string(synced.Messages[0].Content) != "null" {
			t.Fatalf("tombstone missing from the delta: %+v", synced.Messages)
		}
	})

	t.Run("a reaction can be replaced and removed", func(t *testing.T) {
		sender := e.register(t, "reactsender")
		receiver := e.register(t, "reactreceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()
		receiverConn := e.dialWS(t, e.wsToken(t, receiverTokens.AccessToken))
		defer func() { _ = receiverConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"react to me"`)
		ack := e.readAck(t, senderConn)
		e.readMessage(t, receiverConn)

		// The receiver reacts, so the sender's socket carries the frame.
		reaction := map[string]json.RawMessage{
			"content": json.RawMessage(`"👍"`),
			"key":     json.RawMessage(`"k"`),
			"nonce":   json.RawMessage(`"n"`),
			"mac":     json.RawMessage(`"m"`),
		}
		if status := e.doJSON(t, http.MethodPut, "/api/messages/"+ack.Id+"/reaction", reaction, receiverTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("reaction status: %d", status)
		}
		event := e.readEvent(t, senderConn, models.WsResponseTypeReaction)
		if event.ActorId != receiver.Id || string(event.Content) != `"👍"` {
			t.Fatalf("unexpected reaction event: %+v", event)
		}

		// Removing it is a reaction event with no ciphertext.
		if status := e.doJSON(t, http.MethodDelete, "/api/messages/"+ack.Id+"/reaction", nil, receiverTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("reaction removal status: %d", status)
		}
		removed := e.readEvent(t, senderConn, models.WsResponseTypeReaction)
		if string(removed.Content) != "null" || removed.ActorId != receiver.Id {
			t.Fatalf("unexpected reaction removal: %+v", removed)
		}

		// An outsider has no message to react to.
		outsider := e.register(t, "reactoutsider")
		outsiderTokens := e.authenticate(t, outsider)
		if status := e.doJSON(t, http.MethodPut, "/api/messages/"+ack.Id+"/reaction", reaction, outsiderTokens.AccessToken, nil); status != http.StatusForbidden {
			t.Fatalf("outsider reaction status: %d", status)
		}
	})

	t.Run("a pin is shared and can be removed", func(t *testing.T) {
		sender := e.register(t, "pinsender")
		receiver := e.register(t, "pinreceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()
		receiverConn := e.dialWS(t, e.wsToken(t, receiverTokens.AccessToken))
		defer func() { _ = receiverConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"pin me"`)
		ack := e.readAck(t, senderConn)
		e.readMessage(t, receiverConn)

		if status := e.doJSON(t, http.MethodPut, "/api/messages/"+ack.Id+"/pin", nil, receiverTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("pin status: %d", status)
		}
		event := e.readEvent(t, receiverConn, models.WsResponseTypePin)
		if event.Pinned == nil || !*event.Pinned || event.MessageId != ack.Id {
			t.Fatalf("unexpected pin event: %+v", event)
		}

		// The pin is shared: the other participant sees it in the page too.
		var page models.HistoryPage
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+receiver.Id, nil, senderTokens.AccessToken, &page); status != http.StatusOK {
			t.Fatalf("history status: %d", status)
		}
		if len(page.Pins) != 1 || page.Pins[0].MessageId != ack.Id || page.Pins[0].PinnedBy != receiver.Id {
			t.Fatalf("pins = %+v", page.Pins)
		}

		if status := e.doJSON(t, http.MethodDelete, "/api/messages/"+ack.Id+"/pin", nil, senderTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("unpin status: %d", status)
		}
		unpinned := e.readEvent(t, receiverConn, models.WsResponseTypePin)
		if unpinned.Pinned == nil || *unpinned.Pinned {
			t.Fatalf("unexpected unpin event: %+v", unpinned)
		}
		var after models.HistoryPage
		if status := e.doJSON(t, http.MethodGet, "/api/messages?peerId="+receiver.Id, nil, senderTokens.AccessToken, &after); status != http.StatusOK {
			t.Fatalf("history status: %d", status)
		}
		if len(after.Pins) != 0 {
			t.Fatalf("pins survived the unpin: %+v", after.Pins)
		}
	})

	t.Run("a read cursor reaches the peer and the sync", func(t *testing.T) {
		reader := e.register(t, "readreader")
		peer := e.register(t, "readpeer")
		readerTokens := e.authenticate(t, reader)
		peerTokens := e.authenticate(t, peer)

		peerConn := e.dialWS(t, e.wsToken(t, peerTokens.AccessToken))
		defer func() { _ = peerConn.Close() }()

		readAt := time.Now().UTC()
		if status := e.doJSON(t, http.MethodPut, "/api/chats/"+peer.Id+"/read",
			map[string]string{"readAt": readAt.Format(time.RFC3339Nano)},
			readerTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("read cursor status: %d", status)
		}

		var receipt models.ReadReceipt
		deadline := time.Now().Add(10 * time.Second)
		for {
			_ = peerConn.SetReadDeadline(deadline)
			var response struct {
				Type    string             `json:"type"`
				Payload models.ReadReceipt `json:"payload"`
			}
			messageType, data, err := peerConn.ReadMessage()
			if err != nil {
				t.Fatalf("read ws read frame: %v", err)
			}
			if messageType != websocket.BinaryMessage {
				t.Fatalf("unexpected ws message type: %d", messageType)
			}
			if err := json.Unmarshal(data, &response); err != nil {
				t.Fatalf("decode read frame: %v", err)
			}
			if response.Type == string(models.WsResponseTypeRead) {
				receipt = response.Payload
				break
			}
		}
		if receipt.PeerId != reader.Id || receipt.V != 1 {
			t.Fatalf("unexpected read frame: %+v", receipt)
		}

		// The cursor never moves backwards, so an older instant is ignored.
		if status := e.doJSON(t, http.MethodPut, "/api/chats/"+peer.Id+"/read",
			map[string]string{"readAt": readAt.Add(-time.Hour).Format(time.RFC3339Nano)},
			readerTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("stale read cursor status: %d", status)
		}

		peerConn2 := e.dialWS(t, e.wsToken(t, e.authenticate(t, peer).AccessToken))
		defer func() { _ = peerConn2.Close() }()
		synced := e.sync(t, peerConn2, 0)
		if len(synced.Reads) != 1 || synced.Reads[0].PeerId != reader.Id {
			t.Fatalf("reads = %+v", synced.Reads)
		}
		if !synced.Reads[0].ReadAt.Equal(receipt.ReadAt) {
			t.Fatalf("cursor moved backwards: %s -> %s", receipt.ReadAt, synced.Reads[0].ReadAt)
		}
	})

	t.Run("sync replays the lifecycle log and cleared chats", func(t *testing.T) {
		sender := e.register(t, "syncsender")
		receiver := e.register(t, "syncreceiver")
		senderTokens := e.authenticate(t, sender)
		receiverTokens := e.authenticate(t, receiver)

		senderConn := e.dialWS(t, e.wsToken(t, senderTokens.AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"lifecycle"`)
		ack := e.readAck(t, senderConn)

		if status := e.doJSON(t, http.MethodPut, "/api/messages/"+ack.Id+"/pin", nil, senderTokens.AccessToken, nil); status != http.StatusNoContent {
			t.Fatalf("pin status: %d", status)
		}
		if status := e.doJSON(t, http.MethodPatch, "/api/messages/"+ack.Id,
			map[string]string{"content": `"edited"`, "key": "k", "nonce": "n", "mac": "m"},
			senderTokens.AccessToken, nil); status != http.StatusOK {
			t.Fatalf("edit status: %d", status)
		}

		receiverConn := e.dialWS(t, e.wsToken(t, receiverTokens.AccessToken))
		defer func() { _ = receiverConn.Close() }()
		synced := e.syncEvents(t, receiverConn, 0, 0)
		if len(synced.Events) != 2 {
			t.Fatalf("synced %d events, want 2: %+v", len(synced.Events), synced.Events)
		}
		if synced.Events[0].Kind != models.MessageEventPin || synced.Events[1].Kind != models.MessageEventEdit {
			t.Fatalf("unexpected event order: %+v", synced.Events)
		}
		if synced.EventsThrough == 0 || synced.EventsThrough != synced.Events[1].EventSeq {
			t.Fatalf("events cursor = %d", synced.EventsThrough)
		}

		// A second sync at the cursor replays nothing.
		again := e.syncEvents(t, receiverConn, synced.SyncedThrough, synced.EventsThrough)
		if len(again.Events) != 0 {
			t.Fatalf("cursor replayed %d events", len(again.Events))
		}

		// Clearing the chat is replayed to the devices that missed it. Only the
		// account that cleared loses the chat, so a second sender device sees it.
		if status := e.doJSON(t, http.MethodDelete, "/api/messages?peerId="+receiver.Id, nil, senderTokens.AccessToken, nil); status != http.StatusOK {
			t.Fatalf("clear status: %d", status)
		}
		senderConn2 := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn2.Close() }()
		cleared := e.syncEvents(t, senderConn2, 0, 0)
		if len(cleared.Cleared) != 1 || cleared.Cleared[0].PeerId != receiver.Id {
			t.Fatalf("cleared = %+v", cleared.Cleared)
		}
		if len(cleared.Messages) != 0 {
			t.Fatalf("a cleared chat still synced %d messages", len(cleared.Messages))
		}
	})

}

func startEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	valkeyPass := randomSecret(t)
	accessSecret := randomSecret(t)
	refreshSecret := randomSecret(t)
	wsSecret := randomSecret(t)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "valkey/valkey:8-alpine",
			ExposedPorts: []string{"6379/tcp"},
			Cmd:          []string{"valkey-server", "--requirepass", valkeyPass, "--notify-keyspace-events", "Ex"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("Docker is not available: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}

	// The archive lives in PostgreSQL; the suite runs against a throwaway one.
	pg, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("whisper"),
		postgres.WithUsername("whisper"),
		postgres.WithPassword(randomSecret(t)),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)))
	if err != nil {
		t.Skipf("Docker is not available: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(pg)
	})
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	private, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatal(err)
	}
	publicPem, err := encodePublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	mediaDir := t.TempDir()
	singleton := utils.GetSingleton()
	singleton.Config = utils.Config{
		GinMode:               "debug",
		ServerUrl:             "http://localhost",
		DatabaseUrl:           dsn,
		MinClientVersion:      testClientVersion,
		ValkeyHost:            host,
		ValkeyPort:            int(port.Num()),
		ValkeyPassword:        valkeyPass,
		AdminMail:             "admin@example.com",
		SmtpHost:              "localhost",
		SmtpUser:              "admin@example.com",
		SmtpPassword:          randomSecret(t),
		AccessTokenSecret:     accessSecret,
		AccessTokenLifespan:   900,
		RefreshTokenSecret:    refreshSecret,
		RefreshTokenLifespan:  3600,
		WsAccessTokenSecret:   wsSecret,
		WsAccessTokenLifespan: 60,
		MediaTtl:              60,
		MediaMaxUploadSize:    1 << 20,
		MediaDir:              mediaDir,
		// The whole suite talks from 127.0.0.1; the limiter would reject the
		// later subtests.
		RateLimitBurst:     100000,
		RateLimitPerMinute: 100000,
	}

	database.InitDbAt(dsn)
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", host, port.Num())},
		Password:    valkeyPass,
	})
	if err != nil {
		t.Fatal(err)
	}
	singleton.Valkey = client
	t.Cleanup(client.Close)
	singleton.Hub = utils.NewHub()
	go singleton.Hub.Run()

	router, err := routes.NewRouter()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &env{server: server, mediaDir: mediaDir, private: private, publicPem: publicPem}
}

// register creates a user through the real invite + register endpoints.
func (e *env) register(t *testing.T, username string) models.User {
	t.Helper()
	return e.registerWithInvite(t, username, false)
}

// registerAdmin creates a user from an admin invite.
func (e *env) registerAdmin(t *testing.T, username string) models.User {
	t.Helper()
	return e.registerWithInvite(t, username, true)
}

func (e *env) registerWithInvite(t *testing.T, username string, admin bool) models.User {
	t.Helper()
	code, err := utils.RandomASCIIString(64)
	if err != nil {
		t.Fatal(err)
	}
	invite := models.NewInvite(username+"@example.com", admin, time.Now().Add(time.Hour))
	if err := database.PushInvite(code, *invite, 3600); err != nil {
		t.Fatal(err)
	}
	var user models.User
	status := e.doJSON(t, http.MethodPost, "/api/auth/register", models.Register{
		InviteCode: code,
		Username:   username,
		PublicKey:  e.publicPem,
	}, "", &user)
	if status != http.StatusOK {
		t.Fatalf("register %s: status %d", username, status)
	}
	return user
}

// authChallenge fetches a one-time login challenge.
func (e *env) authChallenge(t *testing.T) string {
	t.Helper()
	var response models.AuthChallengeResponse
	if status := e.doJSON(t, http.MethodGet, "/api/auth/challenge", nil, "", &response); status != http.StatusOK {
		t.Fatalf("challenge: status %d", status)
	}
	if response.Challenge == "" || response.ExpiresIn <= 0 {
		t.Fatalf("unexpected challenge: %+v", response)
	}
	return response.Challenge
}

// loginToken signs the RSA login JWT for [user] with a fresh challenge. The
// device id is inside the signature, like the app signs it.
func (e *env) loginToken(t *testing.T, user models.User, challenge string, deviceId string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": user.Id,
		"did": deviceId,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
	}
	if challenge != "" {
		claims["nonce"] = challenge
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(e.private)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

// authenticate performs the RSA challenge and returns the token pair.
func (e *env) authenticate(t *testing.T, user models.User) models.AuthResponse {
	t.Helper()
	// Every login is a fresh installation: a device id belongs to one account.
	return e.authenticateDevice(t, user, uuid.NewString(), "test device")
}

// authenticateDevice signs in from one specific device.
func (e *env) authenticateDevice(t *testing.T, user models.User, deviceId string, label string) models.AuthResponse {
	t.Helper()
	signed := e.loginToken(t, user, e.authChallenge(t), deviceId)
	var response models.AuthResponse
	if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{
		Token: signed, DeviceId: deviceId, Label: label, Platform: "linux",
	}, "", &response); status != http.StatusOK {
		t.Fatalf("auth: status %d", status)
	}
	return response
}

func (e *env) wsToken(t *testing.T, accessToken string) string {
	t.Helper()
	var response models.WsAuthResponse
	if status := e.doJSON(t, http.MethodPost, "/api/wsauth", nil, accessToken, &response); status != http.StatusOK {
		t.Fatalf("wsauth: status %d", status)
	}
	return response.AccessToken
}

func (e *env) dialWS(t *testing.T, token string) *websocket.Conn {
	t.Helper()
	conn, err := e.dial(token)
	if err != nil {
		t.Fatalf("ws handshake: %v", err)
	}
	return conn
}

func (e *env) dial(token string) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws?wsAccessToken=" + url.QueryEscape(token)
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		return nil, err
	}
	return conn, nil
}

// sendMessage writes one encrypted message frame.
func (e *env) sendMessage(t *testing.T, conn *websocket.Conn, receiverID string, content string) {
	t.Helper()
	payload := json.RawMessage(fmt.Sprintf(`{"receiverId":%q,"content":%s,"key":"key","nonce":"nonce","mac":"mac","sentAt":%q}`,
		receiverID, content, time.Now().UTC().Format(time.RFC3339Nano)))
	message, err := json.Marshal(models.WsMessage{Type: models.WsMessageTopicMessage, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		t.Fatalf("send ws message: %v", err)
	}
}

// sync asks the server for the delta above [sinceSeq].
func (e *env) sync(t *testing.T, conn *websocket.Conn, sinceSeq int64) models.WsSynced {
	t.Helper()
	return e.syncRequest(t, conn, models.WsSyncRequest{SinceSeq: sinceSeq})
}

// syncEvents asks for the delta of messages and of lifecycle events in one go.
func (e *env) syncEvents(t *testing.T, conn *websocket.Conn, sinceSeq int64, sinceEventSeq int64) models.WsSynced {
	t.Helper()
	return e.syncRequest(t, conn, models.WsSyncRequest{SinceSeq: sinceSeq, SinceEventSeq: &sinceEventSeq})
}

func (e *env) syncRequest(t *testing.T, conn *websocket.Conn, request models.WsSyncRequest) models.WsSynced {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	message, err := json.Marshal(models.WsMessage{Type: models.WsMessageTopicSync, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		t.Fatalf("send ws sync: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read ws sync response: %v", err)
		}
		if messageType != websocket.BinaryMessage {
			t.Fatalf("unexpected ws message type: %d", messageType)
		}
		var response struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatalf("decode ws sync response: %v", err)
		}
		if response.Type != string(models.WsResponseTypeSynced) {
			continue
		}
		var synced models.WsSynced
		if err := json.Unmarshal(response.Payload, &synced); err != nil {
			t.Fatalf("decode synced payload: %v", err)
		}
		return synced
	}
}

// readAck waits for the confirmation of one sent message.
func (e *env) readAck(t *testing.T, conn *websocket.Conn) models.DeliveryAck {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read ws ack: %v", err)
		}
		if messageType != websocket.BinaryMessage {
			t.Fatalf("unexpected ws message type: %d", messageType)
		}
		var response struct {
			Type    string             `json:"type"`
			Payload models.DeliveryAck `json:"payload"`
		}
		if err := json.Unmarshal(data, &response); err != nil {
			continue
		}
		if response.Type == string(models.WsResponseTypeAck) {
			return response.Payload
		}
	}
}

// readError returns the payload of the next error frame, skipping delivery
// acknowledgements of earlier messages.
func (e *env) readError(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		var response struct {
			Type    string `json:"type"`
			Payload string `json:"payload"`
		}
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read ws error frame: %v", err)
		}
		if messageType != websocket.BinaryMessage {
			t.Fatalf("unexpected ws message type: %d", messageType)
		}
		if err := json.Unmarshal(data, &response); err != nil {
			// A non-string payload (ack) does not decode into this struct.
			continue
		}
		switch response.Type {
		case string(models.WsResponseTypeAck):
			continue
		case string(models.WsResponseTypeError):
			return response.Payload
		default:
			t.Fatalf("expected an error frame, got %s", response.Type)
		}
	}
}

func (e *env) readMessage(t *testing.T, conn *websocket.Conn) models.PrivateMessage {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var response struct {
		Type    string                  `json:"type"`
		Payload []models.PrivateMessage `json:"payload"`
	}
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws message: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("unexpected ws message type: %d", messageType)
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("decode ws response: %v", err)
	}
	if response.Type != string(models.WsResponseTypeMessages) || len(response.Payload) != 1 {
		t.Fatalf("unexpected ws response: %s (%d messages)", response.Type, len(response.Payload))
	}
	return response.Payload[0]
}

// readEvent waits for one mutation frame of the given type, skipping everything
// else the socket may carry.
func (e *env) readEvent(t *testing.T, conn *websocket.Conn, want models.WsResponseType) models.MessageEvent {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		var response struct {
			Type    string              `json:"type"`
			Payload models.MessageEvent `json:"payload"`
		}
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read ws %s frame: %v", want, err)
		}
		if messageType != websocket.BinaryMessage {
			t.Fatalf("unexpected ws message type: %d", messageType)
		}
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatalf("decode ws %s frame: %v", want, err)
		}
		if response.Type != string(want) {
			continue
		}
		return response.Payload
	}
}

func (e *env) uploadMedia(t *testing.T, accessToken string, receiverID string, ciphertext []byte) string {
	t.Helper()
	response := e.uploadRaw(t, accessToken, receiverID, ciphertext)
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("upload status: %d", response.StatusCode)
	}
	var upload models.MediaUploadResponse
	if err := json.NewDecoder(response.Body).Decode(&upload); err != nil {
		t.Fatal(err)
	}
	return upload.Id
}

// uploadMediaStatus uploads and only reports the status code.
func (e *env) uploadMediaStatus(t *testing.T, accessToken string, receiverID string, ciphertext []byte) int {
	t.Helper()
	response := e.uploadRaw(t, accessToken, receiverID, ciphertext)
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode
}

func (e *env) uploadRaw(t *testing.T, accessToken string, receiverID string, ciphertext []byte) *http.Response {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "blob.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(ciphertext); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("receiverId", receiverID); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/media", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func (e *env) downloadMedia(t *testing.T, accessToken string, mediaID string) []byte {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, e.server.URL+"/api/media/"+mediaID, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("download status: %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func (e *env) doJSON(t *testing.T, method, path string, body any, accessToken string, out any) int {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	// The app sends its version on every request; the login refuses an old one.
	request.Header.Set("X-Client-Version", testClientVersion)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode >= http.StatusBadRequest {
		t.Logf("%s %s -> %d %s", method, path, response.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("decode %s: %v (%s)", path, err, data)
		}
	}
	return response.StatusCode
}

func randomSecret(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", buffer)
}

func encodePublicKey(key *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", err
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

// mediaFiles lists the stored media files in a directory.
func mediaFiles(t *testing.T, dir string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{}
	for _, entry := range entries {
		files[entry.Name()] = true
	}
	return files
}
