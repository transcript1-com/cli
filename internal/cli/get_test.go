package cli

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	t.Run("writes the format asked for to a file", func(t *testing.T) {
		h := newHarness(t)
		h.run("--no-wait", rickLink)
		got := h.run("get", "tr_1", "-f", "srt", "-o", "talk.srt")
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stdout != "" {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		expectContains(t, h.read("talk.srt"), "-->")
	})

	t.Run("prints one json line", func(t *testing.T) {
		h := newHarness(t)
		h.run("--no-wait", rickLink)
		got := h.run("get", "--json", "tr_1")
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stderr != "" {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if strings.Count(got.stdout, "\n") != 1 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		expectContains(t, got.stdout, `"id":"tr_1"`, `"transcript"`)
	})

	t.Run("fails with the API error for an unknown id", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("get", "tr_99")
		if got.code != 1 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		expectContains(t, got.stderr, "transcript1: tr_99: not_found")
	})
}

func TestNoWaitWritesNothing(t *testing.T) {
	h := newHarness(t)
	got := h.run("--no-wait", "--jobs", "1", rickLink, otherURL)
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "tr_1 queued\ntr_2 queued\n" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if h.exists("transcripts") {
		t.Fatal("created the default folder")
	}
}
