package handlers

import (
	"jsfraz/shept-server/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSearchUsersRejectsBlankUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	users, err := SearchUsers(c, &models.UsernameQuery{Username: " \t "})
	if err == nil {
		t.Fatal("blank username was accepted")
	}
	if users != nil {
		t.Fatalf("blank username returned %d users", len(*users))
	}
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", c.Writer.Status(), http.StatusBadRequest)
	}
}
