package output

import (
	"strings"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/api"
)

var allCodes = []string{
	api.CodeInvalidRequest, api.CodeInvalidURL, api.CodeUnsupportedPlatform, api.CodeUnsupportedMedia,
	api.CodeUnauthorized, api.CodeInsufficientScope, api.CodeEmailNotVerified, api.CodeInsufficientMinutes,
	api.CodeNoActivePlan, api.CodeNotFound, api.CodeUploadIncomplete, api.CodeInvalidState,
	api.CodeIdempotencyConflict, api.CodeMediaTooLarge, api.CodeRateLimited, api.CodeTooManyActiveJobs,
	api.CodeInternalError, api.CodeServiceUnavailable,
	api.CodeMediaUnavailable, api.CodeMediaPrivate, api.CodeAgeRestricted, api.CodeLiveStream,
	api.CodeNoCaptions, api.CodeMediaTooLong, api.CodeNoSpeech, api.CodeFetchFailed,
	api.CodeTranscriptionFailed, api.CodeUploadTokenRefused, "http_502",
}

var mcpOnly = []string{
	"create_upload", "get_transcription", "list_transcriptions", "get_upload_status", "complete_upload",
	"request_id", "Reconnect",
}

func isNamedFallback(code, step string) bool {
	if step != fallbackStep {
		return false
	}

	return !strings.HasPrefix(code, "http_")
}

func TestEveryCodeHasANextStep(t *testing.T) {
	for _, code := range allCodes {
		t.Run(code+" has a terminal next step", func(t *testing.T) {
			step := NextStep(&api.Error{Code: code})
			if step == "" {
				t.Fatal("next step is empty")
			}

			if isNamedFallback(code, step) {
				t.Fatal("named code falls back to the generic step")
			}

			for _, word := range mcpOnly {
				if strings.Contains(step, word) {
					t.Fatalf("next step %q mentions %q", step, word)
				}
			}
		})
	}
}

func TestNextStepWording(t *testing.T) {
	cases := map[string]string{
		api.CodeInvalidURL: "Pass a single YouTube video link " +
			"(youtube.com/watch?v=…, youtu.be/…, /shorts/, /embed/ or /live/)",
		api.CodeMediaPrivate:        "if you have the file, run transcript1 <file>",
		api.CodeAgeRestricted:       "if you have the file, run transcript1 <file>",
		api.CodeNoCaptions:          "Run again with --source auto or --source stt",
		api.CodeMediaTooLong:        "Speech-to-text accepts up to 5 hours; split the media and transcribe the parts",
		api.CodeNotFound:            "Check the id (it starts with tr_)",
		api.CodeUploadIncomplete:    "Run the command again",
		api.CodeInvalidState:        "Run the command again",
		api.CodeIdempotencyConflict: "Run the command again",
		api.CodeTooManyActiveJobs:   "Wait for a running transcription to finish, or lower --jobs",
		api.CodeUploadTokenRefused:  "Run the command again; the upload starts over",
	}
	for code, want := range cases {
		t.Run(code+" says what to do", func(t *testing.T) {
			if got := NextStep(&api.Error{Code: code}); !strings.Contains(got, want) {
				t.Fatalf("NextStep = %q, want it to contain %q", got, want)
			}
		})
	}
}

func TestNextStepUsesErrorDetails(t *testing.T) {
	t.Run("rate limit names the wait", func(t *testing.T) {
		got := NextStep(&api.Error{Code: api.CodeRateLimited, RetryAfter: 12 * time.Second})
		if got != "Wait 12 seconds, then run the command again" {
			t.Fatalf("NextStep = %q", got)
		}
	})

	t.Run("too large names the limit", func(t *testing.T) {
		e := &api.Error{Code: api.CodeMediaTooLarge, Details: map[string]any{"max_bytes": 2000000000.0}}
		if got := NextStep(e); !strings.Contains(got, "2000000000 bytes") {
			t.Fatalf("NextStep = %q", got)
		}
	})

	t.Run("unsupported platform lists the platforms", func(t *testing.T) {
		e := &api.Error{Code: api.CodeUnsupportedPlatform, Details: map[string]any{
			"supported_platforms": []any{"youtube", "vimeo"},
		}}
		if got := NextStep(e); !strings.Contains(got, "youtube, vimeo") {
			t.Fatalf("NextStep = %q", got)
		}
	})

	t.Run("a plan error uses the action url", func(t *testing.T) {
		e := &api.Error{Code: api.CodeNoActivePlan, Details: map[string]any{"action_url": "https://x.test/plans"}}
		if got := NextStep(e); got != "Choose a plan at https://x.test/plans" {
			t.Fatalf("NextStep = %q", got)
		}
	})

	t.Run("a nil error has no step", func(t *testing.T) {
		if got := NextStep(nil); got != "" {
			t.Fatalf("NextStep = %q", got)
		}
	})
}
