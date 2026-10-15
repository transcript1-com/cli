package inputs

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExpandFolder(t *testing.T) {
	t.Run("lists top-level media files in name order", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "b.wav", 1)
		touch(t, dir, "a.mp3", 1)
		touch(t, dir, "notes.txt", 1)
		touch(t, dir, "sub/c.mp3", 1)
		got, err := Expand([]string{dir}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{"a.mp3", "b.wav"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got %q, want %q", listNames(got), want)
		}

		if got[0].Path != filepath.Join(dir, "a.mp3") {
			t.Fatalf("got %+v", got[0])
		}

		if got[0].Kind != File {
			t.Fatalf("got %+v", got[0])
		}
	})

	t.Run("refuses a folder without media files", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "notes.txt", 1)
		touch(t, dir, "sub/c.mp3", 1)
		expectError(t, []string{dir}, dir+": no audio or video files in this folder")
	})

	t.Run("refuses a media file in a folder that is over 2 GB", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "big.mkv", MaxUploadBytes+1)
		expectError(t, []string{dir}, "big.mkv: larger than 2 GB")
	})
}
