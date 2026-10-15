package files

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestExisting(t *testing.T) {
	t.Run("finds the output of a file input", func(t *testing.T) {
		dir := t.TempDir()
		want := touch(t, dir, "lecture.txt")
		got, ok := Existing(dir, buildFileInput("/m/lecture.mp3"), "text", false)
		if !ok {
			t.Fatalf("got %q, %v", got, ok)
		}

		if got != want {
			t.Fatalf("got %q, %v", got, ok)
		}
	})

	t.Run("finds the output of a link by its video id", func(t *testing.T) {
		dir := t.TempDir()
		want := touch(t, dir, "Anything [dQw4w9WgXcQ].txt")
		got, ok := Existing(dir, buildLinkInput(), "text", false)
		if !ok {
			t.Fatalf("got %q, %v", got, ok)
		}

		if got != want {
			t.Fatalf("got %q, %v", got, ok)
		}
	})

	t.Run("ignores another format", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "lecture.srt")
		touch(t, dir, "Anything [dQw4w9WgXcQ].srt")
		if got, ok := Existing(dir, buildFileInput("/m/lecture.mp3"), "text", false); ok {
			t.Fatalf("found %q", got)
		}

		if got, ok := Existing(dir, buildLinkInput(), "text", false); ok {
			t.Fatalf("found %q", got)
		}
	})

	t.Run("ignores another video id", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "Anything [aaaaaaaaaaa].txt")
		if got, ok := Existing(dir, buildLinkInput(), "text", false); ok {
			t.Fatalf("found %q", got)
		}
	})

	t.Run("uses the full name when asked", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "lecture.txt")
		if got, ok := Existing(dir, buildFileInput("/m/lecture.mp3"), "text", true); ok {
			t.Fatalf("found %q", got)
		}

		want := touch(t, dir, "lecture.mp3.txt")
		got, ok := Existing(dir, buildFileInput("/m/lecture.mp3"), "text", true)
		if !ok {
			t.Fatalf("got %q, %v", got, ok)
		}

		if got != want {
			t.Fatalf("got %q, %v", got, ok)
		}
	})

	t.Run("works in a folder with brackets in its name", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "[2026] talks")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		want := touch(t, dir, "Anything [dQw4w9WgXcQ].txt")
		got, ok := Existing(dir, buildLinkInput(), "text", false)
		if !ok {
			t.Fatalf("got %q, %v", got, ok)
		}

		if got != want {
			t.Fatalf("got %q, %v", got, ok)
		}
	})

	t.Run("reports nothing for a missing folder", func(t *testing.T) {
		if got, ok := Existing(filepath.Join(t.TempDir(), "none"), buildLinkInput(), "text", false); ok {
			t.Fatalf("found %q", got)
		}
	})
}
