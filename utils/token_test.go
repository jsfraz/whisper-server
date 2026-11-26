package utils

import (
	"strings"
	"testing"
	"time"
)

func TestTokenTypeIsEnforced(t *testing.T) {
	secret := "test-secret-not-used-in-production"

	access, err := GenerateToken("user", "device", 60, secret, nil, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := TokenValid(access, secret, TokenTypeAccess)
	if err != nil {
		t.Fatalf("access token rejected for its own type: %v", err)
	}
	if claims.UserId != "user" || claims.DeviceId != "device" {
		t.Fatalf("claims = %+v, want user/device", claims)
	}
	// A WS token signed with the access secret must not pass as an access token.
	if _, err := TokenValid(access, secret, TokenTypeWs); err == nil {
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
	if _, err := TokenValid(token, secret, TokenTypeAccess); err == nil {
		t.Fatal("a token without a type was accepted")
	}
}

// TestTokenWithoutDeviceIsRefused locks the lockout: a token without a device
// claim cannot be checked for revocation, so it must not authenticate at all.
func TestTokenWithoutDeviceIsRefused(t *testing.T) {
	secret := "test-secret-not-used-in-production"
	token, err := GenerateToken("user", "", 60, secret, nil, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TokenValid(token, secret, TokenTypeAccess); err == nil {
		t.Fatal("a token without a device claim was accepted")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"0.1.0", "0.1.0", 0},
		{"0.1.0", "0.2.0", -1},
		{"0.10.0", "0.9.9", 1},
		{"1.0", "0.9.9", 1},
		{"0.1.0-dev", "0.1.0", 0},
		{"", "0.1.0", -1},
	}
	for _, test := range tests {
		if actual := CompareVersions(test.a, test.b); actual != test.expected {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", test.a, test.b, actual, test.expected)
		}
	}
}

func TestClientVersionGate(t *testing.T) {
	previous := GetSingleton().Config.MinClientVersion
	GetSingleton().Config.MinClientVersion = "0.2.0"
	defer func() { GetSingleton().Config.MinClientVersion = previous }()

	if ClientVersionSupported("") {
		t.Fatal("a client without a version must be refused")
	}
	if ClientVersionSupported("0.1.9") {
		t.Fatal("a client below the minimum must be refused")
	}
	if !ClientVersionSupported("0.2.0") {
		t.Fatal("the minimum version must be accepted")
	}
	if !ClientVersionSupported("0.3.0") {
		t.Fatal("a newer client must be accepted")
	}
}

func TestCursorRoundTrip(t *testing.T) {
	cursor, err := DecodeCursor(EncodeCursor(time.Unix(0, 123456789), "abc"))
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Id != "abc" || cursor.SentAt.UnixNano() != 123456789 {
		t.Fatalf("cursor = %+v", cursor)
	}
	if _, err := DecodeCursor("not-a-cursor"); err == nil {
		t.Fatal("a malformed cursor was accepted")
	}
	if _, err := DecodeCursor(EncodeCursor(time.Unix(0, 1), "")); err == nil {
		t.Fatal("a cursor without an id was accepted")
	}
}
