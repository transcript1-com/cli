package output

import (
	"errors"
	"fmt"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
)

func TestErrorLineMinutes(t *testing.T) {
	err := &api.Error{
		Status:  402,
		Code:    api.CodeInsufficientMinutes,
		Message: "Not enough minutes.",
		Details: map[string]any{
			"minutes_needed":    20.0,
			"minutes_remaining": 3.0,
			"action_url":        "https://transcript1.com/dashboard/billing",
		},
	}
	want := "transcript1: talk.mp3: insufficient_minutes: 20 minutes needed, 3 remaining. " +
		"Top up at https://transcript1.com/dashboard/billing"
	if got := ErrorLine("talk.mp3", err); got != want {
		t.Fatalf("ErrorLine = %q, want %q", got, want)
	}
}

func TestErrorLineUnauthorized(t *testing.T) {
	err := &api.Error{
		Status:  401,
		Code:    api.CodeUnauthorized,
		Message: "Invalid API key",
	}
	want := "transcript1: talk.mp3: unauthorized: Invalid API key. " +
		"Run transcript1 login or set TRANSCRIPT1_API_KEY. Keys: https://transcript1.com/dashboard/account/tokens"
	if got := ErrorLine("talk.mp3", err); got != want {
		t.Fatalf("ErrorLine = %q, want %q", got, want)
	}
}

func TestErrorLinePlainError(t *testing.T) {
	got := ErrorLine("talk.mp3", errors.New("dial tcp: connection refused"))
	if want := "transcript1: talk.mp3: dial tcp: connection refused"; got != want {
		t.Fatalf("ErrorLine = %q, want %q", got, want)
	}
}

func TestErrorLineShapes(t *testing.T) {
	t.Run("a wrapped api error still gets its next step", func(t *testing.T) {
		err := fmt.Errorf("get: %w", &api.Error{Code: api.CodeNotFound, Message: "No such transcription."})
		want := "transcript1: tr_1: not_found: No such transcription. Check the id (it starts with tr_)"
		if got := ErrorLine("tr_1", err); got != want {
			t.Fatalf("ErrorLine = %q, want %q", got, want)
		}
	})

	t.Run("an error without a message keeps the code and the step", func(t *testing.T) {
		got := ErrorLine("a.mp3", &api.Error{Code: api.CodeNoSpeech})
		want := "transcript1: a.mp3: no_speech: No speech was detected (nothing was charged); " +
			"check that the media has audible speech"
		if got != want {
			t.Fatalf("ErrorLine = %q, want %q", got, want)
		}
	})

	t.Run("a non-json http error gets the generic step", func(t *testing.T) {
		err := &api.Error{
			Status:  502,
			Code:    "http_502",
			Message: "the server answered HTTP 502",
		}
		got := ErrorLine("a.mp3", err)
		want := "transcript1: a.mp3: http_502: the server answered HTTP 502. " +
			"Retry later; if it keeps failing, contact support@transcript1.com"
		if got != want {
			t.Fatalf("ErrorLine = %q, want %q", got, want)
		}
	})

	t.Run("minutes without details fall back to the message", func(t *testing.T) {
		err := &api.Error{Code: api.CodeInsufficientMinutes, Message: "Out of minutes."}
		want := "transcript1: a.mp3: insufficient_minutes: Out of minutes. " +
			"Top up at https://transcript1.com/dashboard/billing"
		if got := ErrorLine("a.mp3", err); got != want {
			t.Fatalf("ErrorLine = %q, want %q", got, want)
		}
	})
}
