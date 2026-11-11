package middlewares

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLogger(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set(RequestIDHeader, "test-request-id")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(RequestIDHeader); got != "test-request-id" {
		t.Fatalf("request id header: got %q", got)
	}
	for _, want := range []string{"request_id=test-request-id", "path=/ping", "status=204"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("log does not contain %q: %s", want, logs.String())
		}
	}

	// A client that sends no id gets a generated one.
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if got := recorder.Header().Get(RequestIDHeader); got == "" {
		t.Fatal("generated request id is empty")
	}

	// Line breaks in the header are stripped so they cannot forge log lines.
	request = httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set(RequestIDHeader, "evil\nrequest")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if got := recorder.Header().Get(RequestIDHeader); got != "evilrequest" {
		t.Fatalf("sanitized request id: got %q", got)
	}
	if strings.Contains(logs.String(), "evil\nrequest") {
		t.Fatal("log contains an unsanitized request id")
	}
}
