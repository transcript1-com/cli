package inputs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func touch(t *testing.T, dir, name string, size int64) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}

	return p
}

func listNames(in []Input) []string {
	out := make([]string, 0, len(in))
	for _, i := range in {
		out = append(out, i.Name)
	}

	return out
}

func expectError(t *testing.T, args []string, parts ...string) {
	t.Helper()
	got, err := Expand(args, strings.NewReader(""))
	if err == nil {
		t.Fatalf("Expand(%q) = %v, want an error", args, got)
	}

	for _, p := range parts {
		if !strings.Contains(err.Error(), p) {
			t.Fatalf("error %q does not contain %q", err, p)
		}
	}
}

func TestExpandLink(t *testing.T) {
	t.Run("reads a youtu.be link", func(t *testing.T) {
		got, err := Expand([]string{"https://youtu.be/dQw4w9WgXcQ?si=x"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %v", got)
		}

		want := Input{
			Raw:     "https://youtu.be/dQw4w9WgXcQ?si=x",
			Kind:    Link,
			URL:     wantCanonical,
			VideoID: wantID,
		}
		if got[0] != want {
			t.Fatalf("got %+v", got[0])
		}
	})

	t.Run("refuses a link that is not YouTube", func(t *testing.T) {
		expectError(t, []string{"https://vimeo.com/1"}, "https://vimeo.com/1: Only YouTube links and files are supported for now")
	})
}

func TestExpandFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("reads a file with its absolute path, name and size", func(t *testing.T) {
		p := touch(t, dir, "a.mp3", 12)
		got, err := Expand([]string{p}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %v", got)
		}

		want := Input{
			Raw:  p,
			Kind: File,
			Path: p,
			Name: "a.mp3",
			Size: 12,
		}
		if got[0] != want {
			t.Fatalf("got %+v", got[0])
		}
	})

	t.Run("makes a relative path absolute", func(t *testing.T) {
		t.Chdir(dir)
		got, err := Expand([]string{"a.mp3"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %v", got)
		}

		if got[0].Path != filepath.Join(dir, "a.mp3") {
			t.Fatalf("got %+v", got[0])
		}

		if got[0].Raw != "a.mp3" {
			t.Fatalf("got %+v", got[0])
		}
	})

	t.Run("takes names with a leading dash, spaces and non-ASCII as files", func(t *testing.T) {
		touch(t, dir, "-intro.mp3", 1)
		touch(t, dir, "Interview Ünal.m4a", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"-intro.mp3", "Interview Ünal.m4a"}, strings.NewReader("never read"))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 2 {
			t.Fatalf("got %v", got)
		}

		for _, in := range got {
			if in.Kind != File {
				t.Fatalf("got %+v", in)
			}
		}

		if want := []string{"-intro.mp3", "Interview Ünal.m4a"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got names %q", listNames(got))
		}
	})

	t.Run("refuses an unsupported extension", func(t *testing.T) {
		p := touch(t, dir, "x.txt", 1)
		expectError(t, []string{p}, p+": unsupported file type", "mp3, wav, m4a, flac, ogg, opus, mp4, mov, webm or mkv")
	})

	t.Run("refuses a file larger than 2 GB", func(t *testing.T) {
		p := touch(t, dir, "big.mp4", MaxUploadBytes+1)
		expectError(t, []string{p}, p+": larger than 2 GB (2,000,000,000 bytes)")
	})

	t.Run("accepts a file of exactly 2 GB", func(t *testing.T) {
		p := touch(t, dir, "edge.mp4", MaxUploadBytes)
		got, err := Expand([]string{p}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("refuses a path that does not exist", func(t *testing.T) {
		expectError(t, []string{"nope.mp3"}, "nope.mp3: no such file or folder")
	})
}

func TestExpandPattern(t *testing.T) {
	t.Run("expands a pattern to the matching files in order", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "rec/b.mp3", 1)
		touch(t, dir, "rec/a.mp3", 1)
		touch(t, dir, "rec/c.wav", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"rec/*.mp3"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{"a.mp3", "b.mp3"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got names %q", listNames(got))
		}
	})

	t.Run("lists a file once when a pattern and its name both match", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mp3", 1)
		touch(t, dir, "b.mp3", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"a.mp3", "*.mp3"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{"a.mp3", "b.mp3"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got names %q", listNames(got))
		}
	})

	t.Run("expands patterns read from stdin", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mp3", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"-"}, strings.NewReader("*.mp3\n"))
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{"a.mp3"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got names %q", listNames(got))
		}
	})

	t.Run("skips hidden files unless the pattern names a dot", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mp3", 1)
		touch(t, dir, ".hidden.mp3", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"*.mp3"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{"a.mp3"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got names %q", listNames(got))
		}
	})

	t.Run("errors when nothing matches", func(t *testing.T) {
		t.Chdir(t.TempDir())
		expectError(t, []string{"rec/*.mp3"}, "no files match rec/*.mp3")
	})

	t.Run("errors on a malformed pattern", func(t *testing.T) {
		t.Chdir(t.TempDir())
		expectError(t, []string{"rec/[.mp3"}, "rec/[.mp3", "pattern")
	})

	t.Run("uses an existing file whose name has pattern characters as is", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "what?.mp3", 1)
		touch(t, dir, "whats.mp3", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"what?.mp3"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{"what?.mp3"}; !reflect.DeepEqual(listNames(got), want) {
			t.Fatalf("got names %q", listNames(got))
		}
	})

	t.Run("never expands a link", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "dQw4w9WgXcQ?si=x", 1)
		t.Chdir(dir)
		got, err := Expand([]string{"https://youtu.be/dQw4w9WgXcQ?si=x"}, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 || got[0].Kind != Link {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("keeps the old error for a missing name without pattern characters", func(t *testing.T) {
		t.Chdir(t.TempDir())
		expectError(t, []string{"nope.mp3"}, "nope.mp3: no such file or folder")
	})
}
