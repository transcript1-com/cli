package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/inputs"
)

const (
	tiktokID   = "6718335390845095173"
	tiktokLink = "https://www.tiktok.com/@scout2015/video/" + tiktokID
	shortLink  = "https://vm.tiktok.com/ZMabcdEF"
)

func buildSocialLink(platform, url string) inputs.Input {
	return inputs.Input{
		Raw:      url,
		Kind:     inputs.Link,
		Platform: platform,
		URL:      url,
	}
}

func assertFile(t *testing.T, res Result, want string) {
	t.Helper()
	if res.Status != Done {
		t.Fatalf("result %+v", res)
	}

	if res.File != want {
		t.Fatalf("file %q, want %q", res.File, want)
	}

	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
}

func TestNameWithoutLocalID(t *testing.T) {
	_, c := setup(t)
	dir := t.TempDir()

	t.Run("names the file after the media id of the job", func(t *testing.T) {
		res := run(t, c, buildSocialLink("tiktok", tiktokLink), Options{Dir: dir})
		assertFile(t, res, filepath.Join(dir, "Video "+tiktokID+" ["+tiktokID+"].txt"))
	})

	t.Run("names the file after the transcription id when the media id is null", func(t *testing.T) {
		res := run(t, c, buildSocialLink("tiktok", shortLink), Options{Dir: dir})
		assertFile(t, res, filepath.Join(dir, "Video ["+res.T.ID+"].txt"))
	})

	t.Run("keeps the video id of a YouTube link", func(t *testing.T) {
		res := run(t, c, buildLink(rickID), Options{Dir: dir})
		assertFile(t, res, filepath.Join(dir, "Video "+rickID+" ["+rickID+"].txt"))
	})
}

func TestSocialLinkRequest(t *testing.T) {
	srv, c := setup(t)
	res := run(t, c, buildSocialLink("x", "https://x.com/jack/status/20"), Options{})
	if res.Status != Done {
		t.Fatalf("result %+v", res)
	}

	body := string(srv.Requests()[0].Body)
	want := `{"url":"https://x.com/jack/status/20","source":"auto","language":"auto","diarize":false}`
	if body != want {
		t.Fatalf("body %s", body)
	}
}

func TestFetchSocialJob(t *testing.T) {
	_, c := setup(t)
	queued, err := c.CreateLink(context.Background(), "k", "https://x.com/jack/status/20", api.CreateOptions{}, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("builds a link input from the media id", func(t *testing.T) {
		tr := &api.Transcription{ID: "tr_9", Source: api.Source{Platform: "x", URL: "https://x.com/jack/status/20", MediaID: "20"}}
		want := inputs.Input{
			Raw:      "https://x.com/jack/status/20",
			Kind:     inputs.Link,
			Platform: "x",
			URL:      "https://x.com/jack/status/20",
			VideoID:  "20",
		}
		if got := buildInput(tr); got != want {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("uses the transcription id when the media id is empty", func(t *testing.T) {
		tr := &api.Transcription{ID: "tr_9", Source: api.Source{Platform: "tiktok", URL: shortLink}}
		if got := buildInput(tr); got.VideoID != "tr_9" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("still builds a file input for an upload", func(t *testing.T) {
		tr := &api.Transcription{ID: "tr_9", Source: api.Source{Platform: "upload", Filename: "a.mp3"}}
		if got := buildInput(tr); got.Kind != inputs.File {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("names the file from the media id", func(t *testing.T) {
		dir := t.TempDir()
		res := Fetch(context.Background(), c, queued.ID, Options{Dir: dir}, nil)
		assertFile(t, res, filepath.Join(dir, "Video 20 [20].txt"))
	})
}
