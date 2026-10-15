package inputs

import (
	"reflect"
	"strings"
	"testing"
)

func TestExpandStdin(t *testing.T) {
	t.Run("reads one input per line and skips blanks and comments", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mp3", 1)
		t.Chdir(dir)
		list := "\ufeffhttps://youtu.be/dQw4w9WgXcQ\r\n\n# comment\n  a.mp3  \n"
		got, err := Expand([]string{"-"}, strings.NewReader(list))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 2 {
			t.Fatalf("got %+v", got)
		}

		if got[0].Kind != Link {
			t.Fatalf("got %+v", got)
		}

		if got[0].VideoID != wantID {
			t.Fatalf("got %+v", got)
		}

		if got[1].Kind != File {
			t.Fatalf("got %+v", got)
		}

		if got[1].Name != "a.mp3" {
			t.Fatalf("got %+v", got)
		}

		if got[1].Raw != "a.mp3" {
			t.Fatalf("Raw = %q, want trimmed", got[1].Raw)
		}
	})

	t.Run("names the stdin line that fails", func(t *testing.T) {
		_, err := Expand([]string{"-"}, strings.NewReader("nope.mp3\n"))
		if err == nil {
			t.Fatal("got nil, want an error")
		}

		if !strings.Contains(err.Error(), "nope.mp3: no such file or folder") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("keeps the order of arguments and list lines", func(t *testing.T) {
		got, err := Expand([]string{"https://youtu.be/aaaaaaaaaaa", "-", "https://youtu.be/ccccccccccc"},
			strings.NewReader("https://youtu.be/bbbbbbbbbbb\n"))
		if err != nil {
			t.Fatal(err)
		}

		ids := []string{got[0].VideoID, got[1].VideoID, got[2].VideoID}
		if want := []string{"aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"}; !reflect.DeepEqual(ids, want) {
			t.Fatalf("got %q, want %q", ids, want)
		}
	})
}

func TestExpandDedupe(t *testing.T) {
	t.Run("runs the same video once whatever the link form", func(t *testing.T) {
		got, err := Expand([]string{"https://youtu.be/dQw4w9WgXcQ", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=5"},
			strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("runs the same file once whatever the path form", func(t *testing.T) {
		dir := t.TempDir()
		p := touch(t, dir, "a.mp3", 1)
		t.Chdir(dir)
		got, err := Expand([]string{p, "a.mp3", "./a.mp3"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("runs a file once when a folder and the file are both given", func(t *testing.T) {
		dir := t.TempDir()
		p := touch(t, dir, "a.mp3", 1)
		got, err := Expand([]string{p, dir}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %+v", got)
		}
	})
}
