package models

import (
	"testing"
	"time"
)

func TestEffectiveExpiry(t *testing.T) {
	sentAt := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		retention  int
		ttlSeconds int
		want       time.Time
	}{
		{"retention only", 30, 0, sentAt.AddDate(0, 0, 30)},
		{"ttl shorter than retention", 30, 3600, sentAt.Add(time.Hour)},
		{"ttl longer than retention keeps retention", 7, 7 * 24 * 60 * 60, sentAt.AddDate(0, 0, 7)},
		{"a week ttl against a month retention", 30, MaxMessageTTLSeconds, sentAt.AddDate(0, 0, 7)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveExpiry(sentAt, tt.retention, tt.ttlSeconds); !got.Equal(tt.want) {
				t.Fatalf("EffectiveExpiry = %s, want %s", got, tt.want)
			}
		})
	}
}
