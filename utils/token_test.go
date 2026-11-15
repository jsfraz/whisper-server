package utils

import (
	"strings"
	"testing"
)

func TestTokenTypeIsEnforced(t *testing.T) {
	secret := "test-secret-not-used-in-production"

	access, err := GenerateToken("user", 60, secret, nil, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := TokenValid(access, secret, TokenTypeAccess); err != nil {
		t.Fatalf("access token rejected for its own type: %v", err)
	}
	// A WS token signed with the access secret must not pass as an access token.
	if _, _, err := TokenValid(access, secret, TokenTypeWs); err == nil {
		t.Fatal("access token was accepted as a ws token")
	}
}

func TestTokenWithoutTypeIsRefused(t *testing.T) {
	// A token minted before types existed carries no `typ` claim.
	secret := "test-secret-not-used-in-production"
	token := strings.Join([]string{
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"eyJzdWIiOiJ1c2VyIiwiZXhwIjo5OTk5OTk5OTk5fQ",
		"signature",
	}, ".")
	if _, _, err := TokenValid(token, secret, TokenTypeAccess); err == nil {
		t.Fatal("a token without a type was accepted")
	}
}
