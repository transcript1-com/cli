package api

import (
	"errors"
	"fmt"
	"testing"
)

func TestSlotBusy(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"a 429 too_many_active_jobs", &Error{Status: 429, Code: CodeTooManyActiveJobs}, true},
		{"a 429 rate_limited", &Error{Status: 429, Code: CodeRateLimited}, true},
		{"a wrapped slot error", fmt.Errorf("upload: %w", &Error{Status: 429, Code: CodeRateLimited}), true},
		{"a 402 insufficient_minutes", &Error{Status: 402, Code: CodeInsufficientMinutes}, false},
		{"a 429 with another code", &Error{Status: 429, Code: CodeInternalError}, false},
		{"a plain error", errors.New("boom"), false},
		{"no error", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := SlotBusy(c.err); ok != c.want {
				t.Fatalf("got %v", ok)
			}
		})
	}
}
