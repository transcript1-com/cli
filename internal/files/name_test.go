package files

import (
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/inputs"
)

func buildFileInput(path string) inputs.Input {
	name := path[strings.LastIndex(path, "/")+1:]
	in := inputs.Input{
		Raw:  path,
		Kind: inputs.File,
		Path: path,
		Name: name,
	}

	return in
}

func buildLinkInput() inputs.Input {
	return inputs.Input{Kind: inputs.Link, VideoID: "dQw4w9WgXcQ"}
}

func TestName(t *testing.T) {
	t.Run("names a file after its stem", func(t *testing.T) {
		if got := Name(buildFileInput("/m/lecture.mp3"), "", "text", false); got != "lecture.txt" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("keeps the whole file name with full names", func(t *testing.T) {
		if got := Name(buildFileInput("/m/lecture.mp3"), "", "text", true); got != "lecture.mp3.txt" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("names a link after its title and video id", func(t *testing.T) {
		if got := Name(buildLinkInput(), "My talk", "srt", false); got != "My talk [dQw4w9WgXcQ].srt" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("sanitizes the title of a link", func(t *testing.T) {
		if got := Name(buildLinkInput(), "a/b", "vtt", false); got != "a-b [dQw4w9WgXcQ].vtt" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("uses untitled when a link has no title", func(t *testing.T) {
		if got := Name(buildLinkInput(), "", "json", false); got != "untitled [dQw4w9WgXcQ].json" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("keeps the whole video id suffix after a long title", func(t *testing.T) {
		got := Name(buildLinkInput(), strings.Repeat("x", 300), "text", false)
		want := strings.Repeat("x", 120) + " [dQw4w9WgXcQ].txt"
		if got != want {
			t.Fatalf("got %q", got)
		}
	})
}
