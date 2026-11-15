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
	"github.com/gorilla/websocket"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
)

type env struct {
	server    *httptest.Server
	mediaDir  string
	private   *rsa.PrivateKey
	publicPem string
}

// TestE2E covers the phase 2 scenarios against the real router, SQLite and Valkey.
func TestE2E(t *testing.T) {
	e := startEnv(t)

	t.Run("register via invite, RSA auth and refresh", func(t *testing.T) {
		user := e.register(t, "authuser")
		tokens := e.authenticate(t, user)
		if tokens.AccessToken == "" || tokens.RefreshToken == "" {
			t.Fatal("auth returned empty tokens")
		}

		var refreshed models.RefreshResponse
		status := e.doJSON(t, http.MethodPost, "/api/auth/refresh", models.Refresh{RefreshToken: tokens.RefreshToken}, "", &refreshed)
		if status != http.StatusOK || refreshed.AccessToken == "" {
			t.Fatalf("refresh: status %d, token %q", status, refreshed.AccessToken)
		}
	})

	t.Run("the login token is single use", func(t *testing.T) {
		user := e.register(t, "singletoken")
		signed := e.loginToken(t, user, e.authChallenge(t))

		if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{Token: signed}, "", nil); status != http.StatusOK {
			t.Fatalf("first login status: %d", status)
		}
		// Replaying the same token (captured on the wire) must not work.
		if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{Token: signed}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("replayed login status: %d, want %d", status, http.StatusUnauthorized)
		}
		// A token without any challenge is refused as well.
		withoutChallenge := e.loginToken(t, user, "")
		if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{Token: withoutChallenge}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("token without challenge status: %d, want %d", status, http.StatusUnauthorized)
		}
		// A challenge the server never issued is refused too.
		if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{Token: e.loginToken(t, user, "made-up-challenge")}, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("unknown challenge status: %d, want %d", status, http.StatusUnauthorized)
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
	})

	t.Run("offline queue is delivered after reconnect", func(t *testing.T) {
		sender := e.register(t, "senderoffline")
		receiver := e.register(t, "receiveroffline")
		senderConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"offline-ciphertext"`)
		waitFor(t, func() bool {
			keys, err := database.GetAllUserPrivateMessageKeys(receiver.Id)
			return err == nil && len(keys) == 1
		})

		receiverConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		defer func() { _ = receiverConn.Close() }()
		message := e.readMessage(t, receiverConn)
		if message.SenderId != sender.Id || message.RecipientOnline {
			t.Fatalf("unexpected queued message: sender %s, online %t", message.SenderId, message.RecipientOnline)
		}
		if string(message.Content) != `"offline-ciphertext"` {
			t.Fatalf("unexpected content: %s", message.Content)
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
		victimTokens := e.authenticate(t, victim)

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

	t.Run("an acknowledged message leaves the offline queue", func(t *testing.T) {
		sender := e.register(t, "acksender")
		receiver := e.register(t, "ackreceiver")
		senderConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"acked-ciphertext"`)
		waitFor(t, func() bool {
			keys, err := database.GetAllUserPrivateMessageKeys(receiver.Id)
			return err == nil && len(keys) == 1
		})

		receiverConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		defer func() { _ = receiverConn.Close() }()
		message := e.readMessage(t, receiverConn)

		e.ackMessages(t, receiverConn, []models.AckedMessage{
			{SenderId: message.SenderId, SentAt: message.SentAt},
		})

		waitFor(t, func() bool {
			keys, err := database.GetAllUserPrivateMessageKeys(receiver.Id)
			return err == nil && len(keys) == 0
		})
	})

	t.Run("an unacknowledged message is delivered again on reconnect", func(t *testing.T) {
		sender := e.register(t, "resendsender")
		receiver := e.register(t, "resendreceiver")
		senderConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, sender).AccessToken))
		defer func() { _ = senderConn.Close() }()

		e.sendMessage(t, senderConn, receiver.Id, `"unacked-ciphertext"`)
		waitFor(t, func() bool {
			keys, err := database.GetAllUserPrivateMessageKeys(receiver.Id)
			return err == nil && len(keys) == 1
		})

		// First delivery is read but never acknowledged, like a client that dies
		// right after the read.
		firstConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		message := e.readMessage(t, firstConn)
		if string(message.Content) != `"unacked-ciphertext"` {
			t.Fatalf("unexpected content: %s", message.Content)
		}
		_ = firstConn.Close()

		// The client stored nothing, so the message must still be queued.
		waitFor(t, func() bool {
			keys, err := database.GetAllUserPrivateMessageKeys(receiver.Id)
			return err == nil && len(keys) == 1
		})

		secondConn := e.dialWS(t, e.wsToken(t, e.authenticate(t, receiver).AccessToken))
		defer func() { _ = secondConn.Close() }()
		again := e.readMessage(t, secondConn)
		if string(again.Content) != `"unacked-ciphertext"` {
			t.Fatalf("unexpected content after reconnect: %s", again.Content)
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
		SqlitePassword:        randomSecret(t),
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
		MessageTtl:            60,
		MediaTtl:              60,
		MediaMaxUploadSize:    1 << 20,
		MediaDir:              mediaDir,
		// The whole suite talks from 127.0.0.1; the limiter would reject the
		// later subtests.
		RateLimitBurst:     100000,
		RateLimitPerMinute: 100000,
	}

	database.InitSqliteAt(filepath.Join(t.TempDir(), "e2e.sqlite"))
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

// loginToken signs the RSA login JWT for [user] with a fresh challenge.
func (e *env) loginToken(t *testing.T, user models.User, challenge string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": user.Id,
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
	signed := e.loginToken(t, user, e.authChallenge(t))
	var response models.AuthResponse
	if status := e.doJSON(t, http.MethodPost, "/api/auth", models.Auth{Token: signed}, "", &response); status != http.StatusOK {
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

// ackMessages tells the server the received messages are stored.
func (e *env) ackMessages(t *testing.T, conn *websocket.Conn, messages []models.AckedMessage) {
	t.Helper()
	payload, err := json.Marshal(models.WsMessageAck{Messages: messages})
	if err != nil {
		t.Fatal(err)
	}
	message, err := json.Marshal(models.WsMessage{Type: models.WsMessageTopicAck, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		t.Fatalf("send ws ack: %v", err)
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
	if out != nil {
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, out); err != nil {
				t.Fatalf("decode %s: %v (%s)", path, err, data)
			}
		}
	}
	return response.StatusCode
}

func encodePublicKey(public *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

func randomSecret(t *testing.T) string {
	t.Helper()
	value, err := utils.RandomASCIIString(32)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// mediaFiles returns the file names currently in the media directory.
func mediaFiles(t *testing.T, dir string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool, len(entries))
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	return names
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
