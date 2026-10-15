package output

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/transcript1/cli/internal/batch"
	"github.com/transcript1/cli/internal/inputs"
)

const longLink = "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLabcdefghijklmnopqrstuvwxyz0123456789"

var escapes = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\r`)

func listVisibleLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(escapes.ReplaceAllString(out, ""), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func buildLinkEvent(index int, stage batch.Stage) batch.Event {
	in := inputs.Input{
		Raw:  longLink,
		Kind: inputs.Link,
	}
	e := batch.Event{
		Index: index,
		Input: in,
		Stage: stage,
		Sent:  45,
		Total: 100,
	}

	return e
}

func TestTTYLiveLinesFitTheTerminal(t *testing.T) {
	t.Run("a long link is cut with an ellipsis before the stage text", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 12, false)
		r.width = 40
		r.Event(buildLinkEvent(2, batch.Uploading))
		lines := listVisibleLines(buf.String())
		if len(lines) != 1 {
			t.Fatalf("lines = %q", lines)
		}

		if got := utf8.RuneCountInString(lines[0]); got > 39 {
			t.Fatalf("line %q is %d runes, want at most 39", lines[0], got)
		}

		if !strings.HasPrefix(lines[0], "[3/12] https://") {
			t.Fatalf("line = %q", lines[0])
		}

		if !strings.HasSuffix(lines[0], "…  uploading 45%") {
			t.Fatalf("line = %q", lines[0])
		}
	})

	t.Run("every redraw fits and the cursor moves up by the live line count", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 12, false)
		r.width = 40
		r.Event(buildLinkEvent(0, batch.Uploading))
		r.Event(buildLinkEvent(1, batch.Transcribing))
		before := buf.Len()
		r.Event(buildLinkEvent(1, batch.Uploading))
		for _, line := range listVisibleLines(buf.String()) {
			if got := utf8.RuneCountInString(line); got > 39 {
				t.Fatalf("line %q is %d runes, want at most 39", line, got)
			}
		}

		redraw := buf.String()[before:]
		if !strings.HasPrefix(redraw, "\x1b[2A") {
			t.Fatalf("redraw = %q", redraw)
		}

		if strings.Count(redraw, "\n") != 2 {
			t.Fatalf("redraw = %q", redraw)
		}
	})

	t.Run("a short name is untouched", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 1, false)
		r.width = 40
		r.Event(batch.Event{Input: buildFileInput("a.mp3"), Stage: batch.Uploading, Sent: 1, Total: 2})
		got := listVisibleLines(buf.String())
		if len(got) != 1 {
			t.Fatalf("lines = %q", got)
		}

		if got[0] != "[1/1] a.mp3  uploading 50%" {
			t.Fatalf("lines = %q", got)
		}
	})

	t.Run("a cut never splits a character", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 1, false)
		r.width = 30
		in := inputs.Input{
			Raw:  "ünal-ünal-ünal-ünal-ünal.mp3",
			Kind: inputs.File,
			Name: "ünal-ünal-ünal-ünal-ünal.mp3",
		}
		r.Event(batch.Event{Input: in, Stage: batch.Transcribing})
		line := listVisibleLines(buf.String())[0]
		if !utf8.ValidString(line) {
			t.Fatalf("line = %q", line)
		}

		if utf8.RuneCountInString(line) > 29 {
			t.Fatalf("line = %q", line)
		}
	})

	t.Run("a plain run keeps the full link", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 12, false)
		r.width = 40
		r.Event(buildLinkEvent(2, batch.Uploading))
		if want := "[3/12] " + longLink + ": uploading\n"; buf.String() != want {
			t.Fatalf("output = %q, want %q", buf.String(), want)
		}
	})

	t.Run("a final line keeps the full link", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 12, false)
		r.width = 40
		r.Event(buildFinishEvent(2, buildLinkEvent(2, batch.Finished).Input, batch.Result{Status: batch.Done, T: buildBilled(1)}))
		if !strings.Contains(buf.String(), "[3/12] "+longLink+"  done · 1 min\n") {
			t.Fatalf("output = %q", buf.String())
		}
	})
}

func TestDefaultWidthWhenNotATerminal(t *testing.T) {
	if got := NewReporter(&bytes.Buffer{}, TTY, 1, false).width; got != 80 {
		t.Fatalf("width = %d, want 80", got)
	}
}
