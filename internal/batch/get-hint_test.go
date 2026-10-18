package batch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
)

func TestGetHint(t *testing.T) {
	buildFailed := func(status string) Result {
		return Result{
			Status: Failed,
			Err:    errors.New("boom"),
			T: &api.Transcription{
				ID:     "tr_7",
				Status: status,
			},
		}
	}

	t.Run("points a job that is still running to get", func(t *testing.T) {
		if got := buildFailed(api.StatusTranscribing).GetHint(); got != "fetch with transcript1 get tr_7" {
			t.Fatalf("hint %q", got)
		}
	})

	t.Run("points a completed job whose output failed to get", func(t *testing.T) {
		if got := buildFailed(api.StatusCompleted).GetHint(); got != "fetch with transcript1 get tr_7" {
			t.Fatalf("hint %q", got)
		}
	})

	t.Run("gives no hint for a job that failed or was canceled", func(t *testing.T) {
		if buildFailed(api.StatusFailed).GetHint() != "" {
			t.Fatal("hint for a failed job")
		}

		if buildFailed(api.StatusCanceled).GetHint() != "" {
			t.Fatal("hint for a failed job")
		}
	})

	t.Run("gives no hint without a job id or for a result that did not fail", func(t *testing.T) {
		done := buildFailed(api.StatusCompleted)
		done.Status = Done
		if (Result{Status: Failed}).GetHint() != "" {
			t.Fatal("unexpected hint")
		}

		if done.GetHint() != "" {
			t.Fatal("unexpected hint")
		}
	})
}

func TestWriteFailureAfterCompletedHasTheHint(t *testing.T) {
	_, c := setup(t)
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	res := run(t, c, buildLink(rickID), Options{File: filepath.Join(blocker, "x.txt")})
	if res.Status != Failed {
		t.Fatalf("result %+v", res)
	}

	if res.GetHint() != "fetch with transcript1 get tr_1" {
		t.Fatalf("result %+v", res)
	}
}
