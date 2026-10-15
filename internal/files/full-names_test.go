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
