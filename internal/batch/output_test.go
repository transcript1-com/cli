package batch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/api"
)

func TestSkipExisting(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(existing, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	in := buildFileInput(t, "a.mp3")

	t.Run("skips an input whose output exists", func(t *testing.T) {
		srv, c := setup(t)
		res := run(t, c, in, Options{Dir: dir})
		if res.Status != Skipped {
			t.Fatalf("result %+v log %v", res, srv.Log())
		}

		if res.File != existing {
			t.Fatalf("result %+v log %v", res, srv.Log())
		}

		if len(srv.Log()) != 0 {
			t.Fatalf("result %+v log %v", res, srv.Log())
		}
	})

	t.Run("runs it again with overwrite", func(t *testing.T) {
		_, c := setup(t)
		res := run(t, c, in, Options{Dir: dir, Overwrite: true})
		got, _ := os.ReadFile(existing)
		if res.Status != Done {
			t.Fatalf("result %+v file %q", res, got)
		}

		if string(got) != wantText {
			t.Fatalf("result %+v file %q", res, got)
		}
	})
}

func TestTitleNaming(t *testing.T) {
	srv, c := setup(t)
	srv.Titles = map[string]string{rickID: "My talk"}
	dir := t.TempDir()
	res := run(t, c, buildLink(rickID), Options{Dir: dir})

	want := filepath.Join(dir, "My talk [dQw4w9WgXcQ].txt")
	if res.Status != Done {
		t.Fatalf("result %+v", res)
	}

	if res.File != want {
		t.Fatalf("result %+v", res)
	}

	if len(res.Body) != 0 {
		t.Fatalf("result %+v", res)
	}

	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("file %q err %v", got, err)
	}

	if string(got) != wantText {
		t.Fatalf("file %q err %v", got, err)
	}
}

func TestFileOutput(t *testing.T) {
	_, c := setup(t)
	out := filepath.Join(t.TempDir(), "sub", "talk.srt")
	res := run(t, c, buildFileInput(t, "talk.mp3"), Options{File: out, Format: "srt"})
	got, err := os.ReadFile(out)
	if res.Status != Done {
		t.Fatalf("result %+v file %q err %v", res, got, err)
	}

	if res.File != out {
		t.Fatalf("result %+v file %q err %v", res, got, err)
	}

	if err != nil {
		t.Fatalf("result %+v file %q err %v", res, got, err)
	}

	if !strings.HasPrefix(string(got), "1\n00:00:00,000") {
		t.Fatalf("result %+v file %q err %v", res, got, err)
	}
}

func TestUnwritableDir(t *testing.T) {
	_, c := setup(t)
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(blocker, "out")
	res := run(t, c, buildLink(rickID), Options{Dir: dir})

	t.Run("fails with the path in the error", func(t *testing.T) {
		if res.Status != Failed {
			t.Fatalf("result %+v", res)
		}

		if res.Err == nil {
			t.Fatalf("result %+v", res)
		}

		if !strings.Contains(res.Err.Error(), dir) {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("keeps the completed transcription", func(t *testing.T) {
		if res.T == nil {
			t.Fatalf("result %+v", res)
		}

		if res.T.Status != api.StatusCompleted {
			t.Fatalf("result %+v", res)
		}
	})
}
