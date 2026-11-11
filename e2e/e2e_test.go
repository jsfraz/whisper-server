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
			t.Fatalf("unexpected message: sender %d, online %t", message.SenderId, message.RecipientOnline)
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
			t.Fatalf("unexpected queued message: sender %d, online %t", message.SenderId, message.RecipientOnline)
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
	code, err := utils.RandomASCIIString(64)
	if err != nil {
		t.Fatal(err)
	}
	invite := models.NewInvite(username+"@example.com", false, time.Now().Add(time.Hour))
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

// authenticate performs the RSA challenge and returns the token pair.
func (e *env) authenticate(t *testing.T, user models.User) models.AuthResponse {
	t.Helper()
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": user.Id,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
	})
	signed, err := token.SignedString(e.private)
	if err != nil {
		t.Fatal(err)
	}
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

func (e *env) sendMessage(t *testing.T, conn *websocket.Conn, receiverID uint64, content string) {
	t.Helper()
	payload := json.RawMessage(fmt.Sprintf(`{"receiverId":%d,"content":%s,"key":"key","nonce":"nonce","mac":"mac","sentAt":%q}`,
		receiverID, content, time.Now().UTC().Format(time.RFC3339Nano)))
	message, err := json.Marshal(models.WsMessage{Type: models.WsMessageTopicMessage, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		t.Fatalf("send ws message: %v", err)
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

func (e *env) uploadMedia(t *testing.T, accessToken string, receiverID uint64, ciphertext []byte) string {
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
	if err := writer.WriteField("receiverId", fmt.Sprintf("%d", receiverID)); err != nil {
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
