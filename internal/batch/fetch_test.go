package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
)

func TestFetch(t *testing.T) {
	srv, c := setup(t)
	queued, err := c.CreateLink(context.Background(), "k", buildLink(rickID).URL, api.CreateOptions{}, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("waits for the transcription and returns its body", func(t *testing.T) {
		var ev events
		res := Fetch(context.Background(), c, queued.ID, Options{}, ev.emit)
		if res.Status != Done {
			t.Fatalf("result %+v", res)
		}

		if string(res.Body) != wantText {
			t.Fatalf("result %+v", res)
		}

		if res.T.ID != queued.ID {
			t.Fatalf("result %+v", res)
		}

		if stages := ev.listStages(); stages[len(stages)-1] != Finished {
			t.Fatalf("stages %v", stages)
		}
	})

	t.Run("names the file from the transcription", func(t *testing.T) {
		dir := t.TempDir()
		res := Fetch(context.Background(), c, queued.ID, Options{Dir: dir}, nil)
		want := filepath.Join(dir, "Video "+rickID+" ["+rickID+"].txt")
		_, err := os.Stat(want)
		if res.Status != Done {
			t.Fatalf("result %+v err %v", res, err)
		}

		if res.File != want {
			t.Fatalf("result %+v err %v", res, err)
		}

		if err != nil {
			t.Fatalf("result %+v err %v", res, err)
		}
	})

	t.Run("prints the id and status with no-wait", func(t *testing.T) {
		before := len(srv.Log())
		res := Fetch(context.Background(), c, queued.ID, Options{NoWait: true}, nil)
		if res.Status != Started {
			t.Fatalf("result %+v", res)
		}

		if res.T.ID != queued.ID {
			t.Fatalf("result %+v", res)
		}

		if len(srv.Log()) != before+1 {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("keeps a server file name with folders inside the output folder", func(t *testing.T) {
		in := buildFileInput(t, "talk.mp3")
		in.Name = "../../x.mp3"
		started := run(t, c, in, Options{NoWait: true})
		dir := filepath.Join(t.TempDir(), "a", "b")
		res := Fetch(context.Background(), c, started.T.ID, Options{Dir: dir}, nil)
		want := filepath.Join(dir, "x.txt")
		_, err := os.Stat(want)
		if res.Status != Done {
			t.Fatalf("result %+v err %v", res, err)
		}

		if res.File != want {
			t.Fatalf("result %+v err %v", res, err)
		}

		if err != nil {
			t.Fatalf("result %+v err %v", res, err)
		}
	})

	t.Run("names a file called .. untitled", func(t *testing.T) {
		got := buildInput(&api.Transcription{Source: api.Source{Filename: "..", Size: 20}})
		if got.Name != "untitled" {
			t.Fatalf("input %+v", got)
		}
	})

	t.Run("fails with the API error for an unknown id", func(t *testing.T) {
		res := Fetch(context.Background(), c, "tr_404", Options{}, nil)
		if res.Status != Failed {
			t.Fatalf("result %+v", res)
		}

		if !api.IsCode(res.Err, api.CodeNotFound) {
			t.Fatalf("result %+v", res)
		}
	})
}
