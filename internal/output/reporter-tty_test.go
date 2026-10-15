package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/batch"
)

func TestTTYRedraws(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, TTY, 5, false)
	in := buildFileInput("lecture.mp3")
	r.Event(batch.Event{Index: 1, Input: in, Stage: batch.Uploading, Sent: 45, Total: 100})
	r.Event(batch.Event{Index: 1, Input: in, Stage: batch.Transcribing})
	r.Event(buildFinishEvent(1, in, batch.Result{Status: batch.Done, T: buildBilled(12), File: "transcripts/lecture.txt"}))
	out := buf.String()
	for _, want := range []string{
		"\r\x1b[K[2/5] lecture.mp3  uploading 45%",
		"\r\x1b[K[2/5] lecture.mp3  transcribing",
		"\r\x1b[K[2/5] lecture.mp3  done · 12 min · transcripts/lecture.txt\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	}

	if strings.Contains(out, "\x1b[3") {
		t.Fatalf("output %q has color codes", out)
	}
}

func TestTTYRedrawsOnlyWhenTheLineChanges(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, TTY, 1, false)
	in := buildFileInput("a.mp3")
	r.Event(batch.Event{Input: in, Stage: batch.Uploading, Sent: 450, Total: 1000})
	before := buf.Len()
	r.Event(batch.Event{Input: in, Stage: batch.Uploading, Sent: 451, Total: 1000})
	if buf.Len() != before {
		t.Fatalf("a same-percent event redrew: %q", buf.String())
	}
}

func TestTTYKeepsOneLinePerRunningInput(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, TTY, 2, false)
	a, b := buildFileInput("a.mp3"), buildFileInput("b.mp3")
	r.Event(batch.Event{Index: 0, Input: a, Stage: batch.Transcribing})
	r.Event(batch.Event{Index: 1, Input: b, Stage: batch.Transcribing})
	buf.Reset()
	r.Event(buildFinishEvent(0, a, batch.Result{Status: batch.Done, T: buildBilled(2)}))
	out := buf.String()
	if !strings.HasPrefix(out, "\x1b[2A") {
		t.Fatalf("output %q does not start by moving up over both live lines", out)
	}

	if !strings.Contains(out, "[1/2] a.mp3  done · 2 min\n") {
		t.Fatalf("output = %q", out)
	}

	if !strings.Contains(out, "[2/2] b.mp3  transcribing\n") {
		t.Fatalf("output = %q", out)
	}
}

func TestTTYColors(t *testing.T) {
	t.Run("a done line is green when color is on", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 1, true)
		r.Event(buildFinishEvent(0, buildFileInput("a.mp3"), batch.Result{Status: batch.Done, T: buildBilled(1)}))
		if !strings.Contains(buf.String(), "\x1b[32mdone") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("an error line is red when color is on", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 1, true)
		r.Event(buildFinishEvent(0, buildFileInput("a.mp3"), batch.Result{Status: batch.Failed, Err: errors.New("boom")}))
		if !strings.Contains(buf.String(), "\x1b[31mtranscript1: a.mp3: boom\x1b[0m") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("a plain run never colors even when asked", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 1, true)
		r.Event(buildFinishEvent(0, buildFileInput("a.mp3"), batch.Result{Status: batch.Done, T: buildBilled(1)}))
		if strings.Contains(buf.String(), "\x1b") {
			t.Fatalf("output = %q", buf.String())
		}
	})
}
