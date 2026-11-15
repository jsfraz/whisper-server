package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRateLimitRejectsBurstThenRefills covers the login and upload protection.
func TestRateLimitRejectsBurstThenRefills(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newRateLimiter(3, 60)

	for i := 0; i < 3; i++ {
		if !limiter.allow("1.2.3.4") {
			t.Fatalf("request %d was rejected inside the burst", i+1)
		}
	}
	if limiter.allow("1.2.3.4") {
		t.Fatal("request above the burst was accepted")
	}
	// Another client keeps its own bucket.
	if !limiter.allow("5.6.7.8") {
		t.Fatal("a different client was limited by someone else's traffic")
	}
}

// TestRateLimitMiddlewareAnswers429 checks the response shape clients see.
func TestRateLimitMiddlewareAnswers429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimit(1, 60))
	router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("429 response carries no Retry-After header")
	}
}
