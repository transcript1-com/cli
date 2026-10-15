package output

import (
	"testing"

	"github.com/transcript1/cli/internal/inputs"
)

func TestDisplay(t *testing.T) {
	t.Run("a link shows as typed", func(t *testing.T) {
		in := inputs.Input{
			Raw:  "https://youtu.be/abc",
			Kind: inputs.Link,
			URL:  "https://www.youtube.com/watch?v=abc",
		}
		if got := Display(in); got != "https://youtu.be/abc" {
			t.Fatalf("Display = %q", got)
		}
	})

	t.Run("a file shows its base name", func(t *testing.T) {
		in := inputs.Input{
			Raw:  "talks/lecture.mp3",
			Kind: inputs.File,
			Name: "lecture.mp3",
		}
		if got := Display(in); got != "lecture.mp3" {
			t.Fatalf("Display = %q", got)
		}
	})

	t.Run("a file without a name falls back to the raw path base", func(t *testing.T) {
		in := inputs.Input{Raw: "talks/lecture.mp3", Kind: inputs.File}
		if got := Display(in); got != "lecture.mp3" {
			t.Fatalf("Display = %q", got)
		}
	})
}
