package utils

import (
	"sync"
	"time"
)

// MessageRateLimiter allows at most limit calls per second. It guards one
// WebSocket connection against flooding the shared offline queues.
type MessageRateLimiter struct {
	limit int
	mu    sync.Mutex
	start time.Time
	count int
}

// NewMessageRateLimiter returns a limiter allowing [limit] messages per second.
// A limit of zero or less disables it.
//
//	@param limit
//	@return *MessageRateLimiter
func NewMessageRateLimiter(limit int) *MessageRateLimiter {
	return &MessageRateLimiter{limit: limit, start: time.Now()}
}

// Allow consumes one slot and reports whether the message may be processed.
//
//	@return bool
func (l *MessageRateLimiter) Allow() bool {
	if l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.start) >= time.Second {
		l.start = now
		l.count = 0
	}
	if l.count >= l.limit {
		return false
	}
	l.count++
	return true
}
