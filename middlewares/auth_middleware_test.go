package middlewares

import (
	"jsfraz/whisper-server/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	testAccessSecret = "test-access-secret"
	testDeviceID     = "device-id"
)

// accessToken signs an access token for userID, which does not have to exist.
func accessToken(t *testing.T, userID string) string {
	t.Helper()
	utils.GetSingleton().Config.AccessTokenSecret = testAccessSecret
	token, err := utils.GenerateToken(userID, testDeviceID, 60, testAccessSecret, nil, utils.TokenTypeAccess)
	if err != nil {
		t.Fatalf("signing the token failed: %v", err)
	}
	return token
}

// stubDatabase replaces the database lookups the middlewares make.
func stubDatabase(t *testing.T, exists bool, deletePending bool, err error) {
	t.Helper()
	previousExists, previousDeleted, previousDevice := userExistsById, willUserBeDeleted, isDeviceActive
	userExistsById = func(string) (bool, error) { return exists, err }
	willUserBeDeleted = func(string) (bool, error) { return deletePending, err }
	isDeviceActive = func(string, string) (bool, error) { return true, nil }
	t.Cleanup(func() {
		userExistsById, willUserBeDeleted, isDeviceActive = previousExists, previousDeleted, previousDevice
	})
}

// stubRevokedDevice makes every device look revoked.
func stubRevokedDevice(t *testing.T) {
	t.Helper()
	previousDevice := isDeviceActive
	isDeviceActive = func(string, string) (bool, error) { return false, nil }
	t.Cleanup(func() { isDeviceActive = previousDevice })
}

func requestWithToken(t *testing.T, token string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

// TestAuthMiddlewareMissingUserDoesNotPanic covers the abort that used to pass a
// nil error to AbortWithError: a token for a deleted user panicked on every call.
func TestAuthMiddlewareMissingUserDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stubDatabase(t, false, false, nil)

	handlerCalled := false
	router := gin.New()
	router.Use(AuthMiddleware)
	router.GET("/ping", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requestWithToken(t, accessToken(t, "missing-user")))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(recorder.Body.String(), utils.ErrorCurrentUserDoesNotExist) {
		t.Fatalf("body = %q, want the %q error", recorder.Body.String(), utils.ErrorCurrentUserDoesNotExist)
	}
	if handlerCalled {
		t.Fatal("the handler ran for a user that does not exist")
	}
}

// TestAuthMiddlewareDeletePendingDoesNotPanic covers the second nil error abort.
func TestAuthMiddlewareDeletePendingDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stubDatabase(t, true, true, nil)

	handlerCalled := false
	router := gin.New()
	router.Use(AuthMiddleware)
	router.GET("/ping", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requestWithToken(t, accessToken(t, "pending-user")))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("the handler ran for a user waiting for deletion")
	}
}

// TestAuthMiddlewareAcceptsExistingUser keeps the happy path honest.
func TestAuthMiddlewareAcceptsExistingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stubDatabase(t, true, false, nil)

	handlerCalled := false
	router := gin.New()
	router.Use(AuthMiddleware)
	router.GET("/ping", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requestWithToken(t, accessToken(t, "existing-user")))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Fatal("the handler did not run for an existing user")
	}
}

// TestUserDeletionMiddlewarePendingUserDoesNotPanic covers the same abort shape
// in the deletion middleware.
func TestUserDeletionMiddlewarePendingUserDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stubDatabase(t, true, true, nil)

	handlerCalled := false
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", "pending-user") })
	router.Use(UserDeletionMiddleware)
	router.GET("/ping", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(recorder.Body.String(), utils.ErrorAccountWillBeDeleted) {
		t.Fatalf("body = %q, want the %q error", recorder.Body.String(), utils.ErrorAccountWillBeDeleted)
	}
	if handlerCalled {
		t.Fatal("the handler ran for a user waiting for deletion")
	}
}

// TestAuthMiddlewareRejectsRevokedDevice locks the hard lockout: a stolen device
// keeps its key and its unexpired access token, and still gets nothing.
func TestAuthMiddlewareRejectsRevokedDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stubDatabase(t, true, false, nil)
	stubRevokedDevice(t)

	handlerCalled := false
	router := gin.New()
	router.Use(AuthMiddleware)
	router.GET("/ping", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requestWithToken(t, accessToken(t, "existing-user")))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(recorder.Body.String(), utils.ErrorDeviceRevoked) {
		t.Fatalf("body = %q, want the %q error", recorder.Body.String(), utils.ErrorDeviceRevoked)
	}
	if handlerCalled {
		t.Fatal("the handler ran for a revoked device")
	}
}
