package cli

import (
	"strings"
	"testing"

	"github.com/transcript1-com/cli/internal/config"
)

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(h *harness)
		args  []string
		want  string
	}{
		{name: "an unknown flag", args: []string{"--bogus", rickLink}, want: "bogus"},
		{name: "timestamps with srt", args: []string{"--timestamps", "-f", "srt", rickLink}, want: "--timestamps"},
		{name: "an unknown format", args: []string{"-f", "pdf", rickLink}, want: "pdf"},
		{
			name:  "captions for a file",
			setup: func(h *harness) { h.writeMedia("talk.mp3") },
			args:  []string{"--source", "captions", "talk.mp3"},
			want:  "--source",
		},
		{
			name: "speakers with captions",
			args: []string{"--speakers", "--source", "captions", rickLink},
			want: "--speakers",
		},
		{name: "an unknown source", args: []string{"--source", "magic", rickLink}, want: "magic"},
		{name: "an unsupported link", args: []string{"https://example.com/watch?v=1"}, want: "Only YouTube, X, TikTok, Instagram, Facebook and Bluesky links"},
		{
			name:  "an unsupported extension",
			setup: func(h *harness) { h.write("notes.txt", "hi") },
			args:  []string{"notes.txt"},
			want:  "unsupported file type",
		},
		{
			name: "two inputs writing the same file",
			setup: func(h *harness) {
				h.writeMedia("a/talk.mp3")
				h.writeMedia("b/talk.mp3")
			},
			args: []string{"a/talk.mp3", "b/talk.mp3"},
			want: "same file",
		},
		{
			name: "two inputs whose outputs differ only by case",
			setup: func(h *harness) {
				h.writeMedia("Talk.mp3")
				h.writeMedia("talk.wav")
			},
			args: []string{"Talk.mp3", "talk.wav"},
			want: "Talk.mp3 and talk.wav would write the same file",
		},
		{name: "a file pattern with no match", args: []string{"nope/*.mp3"}, want: "no files match nope/*.mp3"},
		{name: "zero jobs", args: []string{"--jobs", "0", rickLink}, want: "--jobs"},
		{name: "too many jobs", args: []string{"--jobs", "21", rickLink}, want: "--jobs"},
		{name: "no inputs", args: []string{"-"}, want: "no inputs: pass a YouTube, X, TikTok, Instagram, Facebook or Bluesky link, a file or a folder"},
		{
			name:  "an output folder under a file",
			setup: func(h *harness) { h.write("blocker", "x") },
			args:  []string{"-o", "blocker/out/", rickLink},
			want:  "blocker",
		},
		{
			name:  "no key",
			setup: func(h *harness) { delete(h.env, config.EnvAPIKey) },
			args:  []string{rickLink},
			want:  "transcript1 login",
		},
		{name: "get without an id", args: []string{"get"}, want: "transcription id"},
		{name: "get with a transcribe flag", args: []string{"get", "--jobs", "3", "tr_1"}, want: "jobs"},
	}

	for _, tc := range cases {
		t.Run("rejects "+tc.name+" before any request", func(t *testing.T) {
			h := newHarness(t)
			if tc.setup != nil {
				tc.setup(h)
			}

			got := h.run(tc.args...)
			if got.code != 2 {
				t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
			}

			if got.stdout != "" {
				t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
			}

			expectContains(t, got.stderr, "transcript1: ", tc.want)
			if log := h.srv.Log(); len(log) != 0 {
				t.Fatalf("requests sent: %v", log)
			}
		})
	}

	t.Run("names the flag in an unknown flag error", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("--bogus")
		if !strings.Contains(got.stderr, "transcript1 --help") {
			t.Fatalf("stderr %q", got.stderr)
		}
	})
}

func TestValidateCaptionsNonYouTube(t *testing.T) {
	h := newHarness(t)
	link := "https://vm.tiktok.com/ZMabcdEF"
	got := h.run("--source", "captions", link)
	if got.code != 2 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	expectContains(t, got.stderr, "--source captions works with YouTube links only, and "+link+" is not one")
	if log := h.srv.Log(); len(log) != 0 {
		t.Fatalf("requests sent: %v", log)
	}
}
