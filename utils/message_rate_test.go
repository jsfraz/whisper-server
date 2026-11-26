package utils

import "testing"

func TestMessageRateLimiterRejectsFlood(t *testing.T) {
	limiter := NewMessageRateLimiter(3)
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Fatalf("message %d was rejected inside the limit", i+1)
		}
	}
	if limiter.Allow() {
		t.Fatal("a message above the limit was accepted")
	}
}

func TestMessageRateLimiterCanBeDisabled(t *testing.T) {
	limiter := NewMessageRateLimiter(0)
	for i := 0; i < 1000; i++ {
		if !limiter.Allow() {
			t.Fatal("a disabled limiter rejected a message")
		}
	}
}
