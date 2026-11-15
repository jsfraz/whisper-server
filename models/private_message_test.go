package models

import (
	"strconv"
	"testing"
	"time"
)

// TestOfflineQueueKeyIsStable locks the ack contract: the receiving client
// sends back the sender id and sentAt it received, and the server must derive
// the same key to remove the entry.
func TestOfflineQueueKeyIsStable(t *testing.T) {
	sentAt := time.Date(2026, 11, 15, 20, 0, 0, 123456789, time.UTC)
	key := OfflineQueueKey("receiver", "sender", sentAt)
	want := "msg:receiver_sender_" + strconv.FormatInt(sentAt.UnixNano(), 10)
	if key != want {
		t.Fatalf("key = %q, want %q", key, want)
	}
	// A resent message (same participants, same client timestamp) must land on
	// the same key instead of creating a second copy.
	again := OfflineQueueKey("receiver", "sender", sentAt.In(time.FixedZone("CET", 3600)))
	if again != key {
		t.Fatalf("key depends on the timezone: %q != %q", again, key)
	}
}

// TestOfflineQueueKeyIsScopedToConversation keeps two senders apart.
func TestOfflineQueueKeyIsScopedToConversation(t *testing.T) {
	sentAt := time.Now().UTC()
	if OfflineQueueKey("receiver", "senderA", sentAt) == OfflineQueueKey("receiver", "senderB", sentAt) {
		t.Fatal("keys of two senders collide")
	}
	if OfflineQueueKey("receiverA", "sender", sentAt) == OfflineQueueKey("receiverB", "sender", sentAt) {
		t.Fatal("keys of two receivers collide")
	}
}
