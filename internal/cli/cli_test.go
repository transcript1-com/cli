package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/version"
)

func TestSingleLinkToStdout(t *testing.T) {
	t.Run("prints exactly the transcript and plain progress", func(t *testing.T) {
		h := newHarness(t)
		got := h.run(rickLink)
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stdout != transcriptText {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		expectContains(t, got.stderr, "[1/1] "+rickLink+": queued", "done", "1 done")
		if strings.Contains(got.stderr, "\x1b") {
			t.Fatalf("progress has escape codes: %q", got.stderr)
		}
	})

	t.Run("prints nothing but errors with quiet", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("-q", rickLink)
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stdout != transcriptText {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stderr != "" {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}
	})
}

func TestFormatAndOutFile(t *testing.T) {
	h := newHarness(t)
	got := h.run("-f", "srt", "-o", "out.srt", rickLink)
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	expectContains(t, h.read("out.srt"), "00:00:00,000 --> 00:00:03,200", "Welcome to the show.")
}

func TestBatchToFolder(t *testing.T) {
	h := newHarness(t)
	got := h.run("-o", "out", rickLink, otherURL)
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	expectContains(t, got.stderr, "2 done")
	expectContains(t, h.read("out/Video "+rickID+" ["+rickID+"].txt"), "Welcome to the show.")
	expectContains(t, h.read("out/Video "+otherID+" ["+otherID+"].txt"), "Welcome to the show.")
}

func TestFlagsAfterInputs(t *testing.T) {
	t.Run("takes a flag that follows the link", func(t *testing.T) {
		h := newHarness(t)
		got := h.run(rickLink, "-f", "vtt")
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if !strings.HasPrefix(got.stdout, "WEBVTT\n") {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}
	})

	t.Run("treats an input after -- as a file", func(t *testing.T) {
		h := newHarness(t)
		h.writeMedia("-intro.mp3")
		got := h.run("--", "-intro.mp3")
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stdout != transcriptText {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		expectContains(t, strings.Join(h.srv.Log(), "\n"), "POST /v1/transcriptions")
	})

	t.Run("reads ./get as an input and get as the command", func(t *testing.T) {
		h := newHarness(t)
		h.writeMedia("get/lecture.mp3")
		got := h.run("./get", "-o", "out/")
		if got.code != 0 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		if !h.exists("out/lecture.txt") {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		got = h.run("get")
		if got.code != 2 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		expectContains(t, got.stderr, "transcription id")
	})
}

func isCompletedWithFile(obj map[string]any) bool {
	if obj["status"] != "completed" {
		return false
	}

	return obj["file"] != nil
}

func TestJSONMode(t *testing.T) {
	h := newHarness(t)
	got := h.run("--json", "-o", "out", rickLink, otherURL)
	if got.code != 0 {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	if got.stderr != "" {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	lines := strings.Split(strings.TrimSpace(got.stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("stdout %q", got.stdout)
	}

	seen := map[string]bool{}
	for _, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}

		input, _ := obj["input"].(string)
		seen[input] = isCompletedWithFile(obj)
	}

	if !seen[rickLink] {
		t.Fatalf("lines %q", got.stdout)
	}

	if !seen[otherURL] {
		t.Fatalf("lines %q", got.stdout)
	}
}

func TestNoWaitThenGet(t *testing.T) {
	h := newHarness(t)
	got := h.run("--no-wait", rickLink)
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "tr_1 queued\n" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	got = h.run("get", "tr_1")
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != transcriptText {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}
}

func TestHelpAndVersion(t *testing.T) {
	t.Run("lists the commands and three examples", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("--help")
		if got.code != 0 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		if got.stderr != "" {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		expectContains(t, got.stdout, "transcript1 get", "transcript1 login", "transcript1 logout", "transcript1 update",
			"--version", "--format", "--jobs", "Examples:")
		_, examples, _ := strings.Cut(got.stdout, "Examples:")
		if n := strings.Count(examples, "\n  transcript1 "); n != 3 {
			t.Fatalf("examples %d in %q", n, got.stdout)
		}
	})

	t.Run("prints the version", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("--version")
		if got.code != 0 {
			t.Fatalf("code %d stdout %q", got.code, got.stdout)
		}

		if got.stdout != "transcript1 "+version.Version+"\n" {
			t.Fatalf("code %d stdout %q", got.code, got.stdout)
		}
	})
}

func TestUpdateNotConfigured(t *testing.T) {
	h := newHarness(t)
	got := h.run("update")
	if got.code != 1 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	expectContains(t, got.stderr, "updates are not configured in this build")
}

const noticeLine = "transcript1 1.2.0 is available. Run transcript1 update."

func fakeNotice(t *testing.T, release string) *int {
	t.Helper()
	calls := 0
	savedCheck := checkUpdate
	savedVersion := version.Version
	checkUpdate = func(_ context.Context, _ *http.Client, base, current, cacheDir string, _ time.Time) string {
		calls++
		if base != "https://downloads.transcript1.com" {
			t.Errorf("check %q %q %q", base, current, cacheDir)
		}

		if current != release {
			t.Errorf("check %q %q %q", base, current, cacheDir)
		}

		if cacheDir == "" {
			t.Errorf("check %q %q %q", base, current, cacheDir)
		}

		return noticeLine
	}

	version.Version = release
	t.Cleanup(func() {
		checkUpdate = savedCheck
		version.Version = savedVersion
	})

	return &calls
}

func TestNoticeSuppressed(t *testing.T) {
	t.Run("prints the notice after the run on a terminal", func(t *testing.T) {
		h := newHarness(t)
		calls := fakeNotice(t, "1.0.0")
		h.tty = true
		got := h.run(rickLink)
		if got.code != 0 {
			t.Fatalf("code %d calls %d stdout %q", got.code, *calls, got.stdout)
		}

		if got.stdout != transcriptText {
			t.Fatalf("code %d calls %d stdout %q", got.code, *calls, got.stdout)
		}

		if *calls != 1 {
			t.Fatalf("code %d calls %d stdout %q", got.code, *calls, got.stdout)
		}

		if !strings.HasSuffix(got.stderr, "1 minute used\n"+noticeLine+"\n") {
			t.Fatalf("stderr %q", got.stderr)
		}
	})

	cases := []struct {
		name, release string
		args          []string
		tty           bool
		env           string
	}{
		{name: "with --json", release: "1.0.0", args: []string{"--json", rickLink}, tty: true},
		{name: "with -q", release: "1.0.0", args: []string{"-q", rickLink}, tty: true},
		{name: "when stderr is not a terminal", release: "1.0.0", args: []string{rickLink}},
		{name: "with TRANSCRIPT1_NO_UPDATE_CHECK=1", release: "1.0.0", args: []string{rickLink}, tty: true, env: "1"},
		{name: "in a dev build", release: "dev", args: []string{rickLink}, tty: true},
	}

	for _, c := range cases {
		t.Run("stays silent "+c.name, func(t *testing.T) {
			h := newHarness(t)
			calls := fakeNotice(t, c.release)
			h.tty = c.tty
			h.env["TRANSCRIPT1_NO_UPDATE_CHECK"] = c.env
			got := h.run(c.args...)
			if got.code != 0 {
				t.Fatalf("code %d calls %d stderr %q", got.code, *calls, got.stderr)
			}

			if *calls != 0 {
				t.Fatalf("code %d calls %d stderr %q", got.code, *calls, got.stderr)
			}

			if strings.Contains(got.stderr+got.stdout, noticeLine) {
				t.Fatalf("code %d calls %d stderr %q", got.code, *calls, got.stderr)
			}
		})
	}
}
