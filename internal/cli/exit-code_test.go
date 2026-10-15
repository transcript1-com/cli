package cli

import (
	"context"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/api"
)

func TestFailedInputExit1(t *testing.T) {
	h := newHarness(t)
	h.srv.FailCode = api.CodeNoSpeech
	got := h.run(rickLink)
	if got.code != 1 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	expectContains(t, got.stderr, "transcript1: "+rickLink+": no_speech", "1 failed")
}

func TestSkippedForMinutesExit1(t *testing.T) {
	h := newHarness(t)
	h.srv.MinutesLeft = 1
	got := h.run("--jobs", "1", "-o", "out", rickLink, otherURL, thirdURL)
	if got.code != 1 {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	expectContains(t, got.stderr, "insufficient_minutes", "1 done, 1 failed, 1 skipped")
}

func TestSkippedExistingExit0(t *testing.T) {
	h := newHarness(t)
	h.write("out/Video "+rickID+" ["+rickID+"].txt", "old")
	got := h.run("-o", "out", rickLink, otherURL)
	if got.code != 0 {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	expectContains(t, got.stderr, "1 done, 1 skipped")
}

func TestUnauthorizedExit2(t *testing.T) {
	h := newHarness(t)
	h.env["TRANSCRIPT1_API_KEY"] = "t1_wrong"
	got := h.run(rickLink)
	if got.code != 2 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	expectContains(t, got.stderr, "unauthorized", "transcript1 login")
}

func TestUnauthorizedJSONExit2(t *testing.T) {
	h := newHarness(t)
	h.env["TRANSCRIPT1_API_KEY"] = "t1_wrong"
	got := h.run("--json", rickLink)
	if got.code != 2 {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	expectContains(t, got.stdout, `"unauthorized"`)
	expectContains(t, got.stderr, "transcript1 login")
}

func TestInterruptedExit130(t *testing.T) {
	h := newHarness(t)
	h.srv.Polls = 1_000_000
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for len(h.srv.Log()) < 2 {
			time.Sleep(time.Millisecond)
		}

		cancel()
	}()

	got := h.runWith(ctx, rickLink)
	if got.code != 130 {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	expectContains(t, got.stderr, "still running: tr_1 ("+rickLink+"), fetch with transcript1 get tr_1")
}
