package models

import "testing"

// TestConversationKeyIsOrderIndependent locks the archive's conversation key:
// the same two accounts must land in one index range regardless of who sends.
func TestConversationKeyIsOrderIndependent(t *testing.T) {
	key := ConversationKey("aaaaaaaa-0000-0000-0000-000000000001", "bbbbbbbb-0000-0000-0000-000000000002")
	if key != ConversationKey("bbbbbbbb-0000-0000-0000-000000000002", "aaaaaaaa-0000-0000-0000-000000000001") {
		t.Fatal("conversation key depends on the argument order")
	}
	if key != "aaaaaaaa-0000-0000-0000-000000000001|bbbbbbbb-0000-0000-0000-000000000002" {
		t.Fatalf("unexpected key: %q", key)
	}
}

// TestToWireSetsThePeerOfTheViewer keeps a message readable from both sides:
// the recipient must not see its own id as the peer.
func TestToWireSetsThePeerOfTheViewer(t *testing.T) {
	message := PrivateMessage{SenderId: "sender", ReceiverId: "receiver"}
	if peer := message.ToWire("receiver", true).PeerId; peer != "sender" {
		t.Fatalf("receiver peer = %q, want sender", peer)
	}
	if peer := message.ToWire("sender", true).PeerId; peer != "receiver" {
		t.Fatalf("sender peer = %q, want receiver", peer)
	}
}
