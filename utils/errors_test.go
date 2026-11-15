package utils

import (
	"errors"
	"testing"
)

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"code passes through", errors.New(ErrorMediaNotFound), ErrorMediaNotFound},
		{"sentence is replaced", errors.New("media file too large"), ErrorInternal},
		{"nil is replaced", nil, ErrorInternal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ErrorCode(test.err); got != test.want {
				t.Fatalf("ErrorCode() = %q, want %q", got, test.want)
			}
		})
	}
}
