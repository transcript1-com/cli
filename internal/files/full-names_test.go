package files

import (
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/inputs"
)

func TestFullNames(t *testing.T) {
	t.Run("marks inputs that share a stem", func(t *testing.T) {
		got, err := FullNames([]inputs.Input{buildFileInput("a/x.mp3"), buildFileInput("a/x.wav"), buildFileInput("a/y.mp3")})
		if err != nil {
			t.Fatal(err)
		}

		if !got[0] {
			t.Fatalf("got %v", got)
		}

		if !got[1] {
			t.Fatalf("got %v", got)
		}

		if got[2] {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("fails when two folders hold the same file name", func(t *testing.T) {
		_, err := FullNames([]inputs.Input{buildFileInput("a/x.mp3"), buildFileInput("b/x.mp3")})
		if err == nil {
			t.Fatal("expected an error")
		}

		if !strings.Contains(err.Error(), "a/x.mp3") {
			t.Fatalf("error does not name both paths: %v", err)
		}

		if !strings.Contains(err.Error(), "b/x.mp3") {
			t.Fatalf("error does not name both paths: %v", err)
		}
	})

	t.Run("fails when two outputs differ only by case", func(t *testing.T) {
		_, err := FullNames([]inputs.Input{buildFileInput("a/X.mp3"), buildFileInput("a/x.wav")})
		if err == nil {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), "a/X.mp3") {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), "a/x.wav") {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("fails when a full name repeats another input's output", func(t *testing.T) {
		_, err := FullNames([]inputs.Input{buildFileInput("a/x.mp3"), buildFileInput("a/x.wav"), buildFileInput("a/x.mp3.wav")})
		if err == nil {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), "a/x.mp3") {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), "a/x.mp3.wav") {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("accepts files with different names", func(t *testing.T) {
		if _, err := FullNames([]inputs.Input{buildFileInput("a/a.mp3"), buildFileInput("a/b.mp3")}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("never collides links", func(t *testing.T) {
		got, err := FullNames([]inputs.Input{buildLinkInput(), buildLinkInput(), buildFileInput("a/x.mp3")})
		if err != nil {
			t.Fatal(err)
		}

		if got[0] {
			t.Fatalf("got %v", got)
		}

		if got[1] {
			t.Fatalf("got %v", got)
		}

		if got[2] {
			t.Fatalf("got %v", got)
		}
	})
}
