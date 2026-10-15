package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/fakeapi"
	"github.com/transcript1/cli/internal/inputs"
)

const testSize = 50

type recorder struct {
	mu       sync.Mutex
	created  []string
	progress [][2]int64
	log      func() []string
	putsSeen []int
}

func (r *recorder) buildOptions() Options {
	return Options{
		IdempotencyKey: "idem-1",
		Created: func(id string) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.created = append(r.created, id)
			r.putsSeen = append(r.putsSeen, countPrefix(r.log(), "PUT chunk"))
		},
		Progress: func(sent, total int64) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.progress = append(r.progress, [2]int64{sent, total})
		},
	}
}

func setup(t *testing.T, name string) (*fakeapi.Server, *api.Client, inputs.Input, []byte) {
	t.Helper()
	srv := fakeapi.New(t)
	c := api.New(srv.URL, srv.Key)
	c.Sleep = func(context.Context, time.Duration) error { return nil }
	data := make([]byte, testSize)
	for i := range data {
		data[i] = byte('a' + i%26)
	}

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	in := inputs.Input{
		Raw:  path,
		Kind: inputs.File,
		Path: path,
		Name: name,
		Size: testSize,
	}

	return srv, c, in, data
}

func waitForPeak(srv *fakeapi.Server, peak int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if srv.PeakChunks() >= peak {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}
}

func countPrefix(log []string, prefix string) int {
	n := 0
	for _, line := range log {
		if strings.HasPrefix(line, prefix) {
			n++
		}
	}

	return n
}

func TestUploadAllChunksThenComplete(t *testing.T) {
	srv, c, in, data := setup(t, "talk.mp3")
	r := &recorder{log: srv.Log}
	tr, err := Run(context.Background(), c, in, r.buildOptions())
	if err != nil {
		t.Fatal(err)
	}

	log := srv.Log()

	t.Run("puts every planned chunk once", func(t *testing.T) {
		puts := listChunkPuts(srv)
		if len(puts) != 7 {
			t.Fatalf("puts %v", puts)
		}

		for i := range 7 {
			if !slices.Equal(puts[i], []int{200}) {
				t.Fatalf("chunk %d answered %v", i, puts[i])
			}
		}

		if !bytes.Equal(srv.Uploaded(tr.ID), data) {
			t.Fatalf("uploaded %q", srv.Uploaded(tr.ID))
		}
	})

	t.Run("completes once, after the last chunk", func(t *testing.T) {
		last := log[len(log)-1]
		if countPrefix(log, "POST /v1/transcriptions/") != 1 {
			t.Fatalf("log %v", log)
		}

		if last != "POST /v1/transcriptions/"+tr.ID+"/upload/complete" {
			t.Fatalf("log %v", log)
		}

		if countPrefix(log, "POST /v1/transcriptions") != 2 {
			t.Fatalf("log %v", log)
		}

		if log[0] != "POST /v1/transcriptions" {
			t.Fatalf("log %v", log)
		}
	})

	t.Run("returns the queued transcription", func(t *testing.T) {
		if tr.Status != api.StatusQueued {
			t.Fatalf("status %q", tr.Status)
		}
	})

	t.Run("reports the id before the first chunk", func(t *testing.T) {
		if !slices.Equal(r.created, []string{tr.ID}) {
			t.Fatalf("created %v, puts before %v", r.created, r.putsSeen)
		}

		if r.putsSeen[0] != 0 {
			t.Fatalf("created %v, puts before %v", r.created, r.putsSeen)
		}
	})

	t.Run("reports growing progress that ends at the full size", func(t *testing.T) {
		if len(r.progress) != 7 {
			t.Fatalf("progress %v", r.progress)
		}

		if r.progress[6] != [2]int64{testSize, testSize} {
			t.Fatalf("progress %v", r.progress)
		}

		for i := 1; i < len(r.progress); i++ {
			if r.progress[i][0] <= r.progress[i-1][0] {
				t.Fatalf("progress %v", r.progress)
			}
		}
	})
}

func TestUploadParallelLimit(t *testing.T) {
	srv, c, in, _ := setup(t, "talk.mp3")
	srv.BlockChunks = make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"})
		done <- err
	}()

	waitForPeak(srv, 4, 5*time.Second)
	time.Sleep(50 * time.Millisecond)
	close(srv.BlockChunks)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	t.Run("keeps exactly 4 chunk PUTs in flight", func(t *testing.T) {
		if srv.PeakChunks() != 4 {
			t.Fatalf("peak %d", srv.PeakChunks())
		}
	})
}

func TestUploadTokenRenewal(t *testing.T) {
	srv, c, in, data := setup(t, "talk.mp3")
	srv.ChunkStatus = map[int][]int{3: {401}}
	tr, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"})
	if err != nil {
		t.Fatal(err)
	}

	log := srv.Log()

	t.Run("creates again with the same key and reads the progress once", func(t *testing.T) {
		var keys []string
		for _, r := range srv.Requests() {
			if r.Method != "POST" {
				continue
			}

			if r.Path != "/v1/transcriptions" {
				continue
			}

			keys = append(keys, r.Header.Get("Idempotency-Key"))
		}

		if !slices.Equal(keys, []string{"idem-1", "idem-1"}) {
			t.Fatalf("keys %v", keys)
		}

		if countPrefix(log, "GET /v1/transcriptions/"+tr.ID+"/upload") != 1 {
			t.Fatalf("log %v", log)
		}
	})

	t.Run("never sends an acknowledged chunk again", func(t *testing.T) {
		for index, statuses := range listChunkPuts(srv) {
			if countStatus(statuses, 200) != 1 {
				t.Fatalf("chunk %d answered %v", index, statuses)
			}
		}

		if !slices.Equal(listChunkPuts(srv)[3], []int{401, 200}) {
			t.Fatalf("chunk 3 answered %v", listChunkPuts(srv)[3])
		}
	})

	t.Run("finishes the upload", func(t *testing.T) {
		if tr.Status != api.StatusQueued {
			t.Fatalf("status %q, uploaded %q", tr.Status, srv.Uploaded(tr.ID))
		}

		if !bytes.Equal(srv.Uploaded(tr.ID), data) {
			t.Fatalf("status %q, uploaded %q", tr.Status, srv.Uploaded(tr.ID))
		}
	})
}

func TestUploadTokenRenewalLimit(t *testing.T) {
	srv, c, in, _ := setup(t, "talk.mp3")
	srv.ChunkStatus = map[int][]int{3: {401, 401, 401}}
	_, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"})

	t.Run("fails with unauthorized after 2 renewals", func(t *testing.T) {
		if !api.IsCode(err, api.CodeUnauthorized) {
			t.Fatalf("got %v", err)
		}

		if n := countPrefix(srv.Log(), "POST /v1/transcriptions"); n != 3 {
			t.Fatalf("%d creates, log %v", n, srv.Log())
		}

		if !slices.Equal(listChunkPuts(srv)[3], []int{401, 401, 401}) {
			t.Fatalf("chunk 3 answered %v", listChunkPuts(srv)[3])
		}
	})

	t.Run("does not complete", func(t *testing.T) {
		if countPrefix(srv.Log(), "POST /v1/transcriptions/") != 0 {
			t.Fatalf("log %v", srv.Log())
		}
	})
}

func TestUploadIncompleteResent(t *testing.T) {
	srv, c, in, data := setup(t, "talk.mp3")
	srv.CompleteMissing = []int{2}
	tr, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sends the missing chunk again and completes twice", func(t *testing.T) {
		if !slices.Equal(listChunkPuts(srv)[2], []int{200, 200}) {
			t.Fatalf("chunk 2 answered %v", listChunkPuts(srv)[2])
		}

		if n := countPrefix(srv.Log(), "POST /v1/transcriptions/"+tr.ID+"/upload/complete"); n != 2 {
			t.Fatalf("log %v", srv.Log())
		}
	})

	t.Run("returns the queued transcription", func(t *testing.T) {
		if tr.Status != api.StatusQueued {
			t.Fatalf("status %q", tr.Status)
		}

		if !bytes.Equal(srv.Uploaded(tr.ID), data) {
			t.Fatalf("status %q", tr.Status)
		}
	})
}

func TestUploadCanceled(t *testing.T) {
	srv, c, in, _ := setup(t, "talk.mp3")
	srv.BlockChunks = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	r := &recorder{log: srv.Log}
	opt := r.buildOptions()
	created := opt.Created
	opt.Created = func(id string) {
		created(id)
		go func() {
			for srv.PeakChunks() == 0 {
				time.Sleep(time.Millisecond)
			}

			cancel()
		}()
	}

	_, err := Run(ctx, c, in, opt)

	t.Run("returns context.Canceled", func(t *testing.T) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("reported the id first", func(t *testing.T) {
		if len(r.created) != 1 {
			t.Fatalf("created %v", r.created)
		}

		if !strings.HasPrefix(r.created[0], "tr_") {
			t.Fatalf("created %v", r.created)
		}
	})

	t.Run("does not complete", func(t *testing.T) {
		if countPrefix(srv.Log(), "POST /v1/transcriptions/") != 0 {
			t.Fatalf("log %v", srv.Log())
		}
	})
}

func TestUploadFilenameAndMime(t *testing.T) {
	srv, c, in, _ := setup(t, "Interview Ünal.m4a")
	if _, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"}); err != nil {
		t.Fatal(err)
	}

	var body struct {
		Upload struct {
			Filename string `json:"filename"`
			Size     int64  `json:"size"`
			MimeType string `json:"mime_type"`
		} `json:"upload"`
	}
	if err := json.Unmarshal(srv.Requests()[0].Body, &body); err != nil {
		t.Fatal(err)
	}

	t.Run("sends the name as it is, with the mime type of its extension", func(t *testing.T) {
		u := body.Upload
		if u.Filename != "Interview Ünal.m4a" {
			t.Fatalf("got %+v", body.Upload)
		}

		if u.MimeType != "audio/mp4" {
			t.Fatalf("got %+v", body.Upload)
		}

		if u.Size != testSize {
			t.Fatalf("got %+v", body.Upload)
		}
	})
}

func countStatus(statuses []int, status int) int {
	n := 0
	for _, s := range statuses {
		if s == status {
			n++
		}
	}

	return n
}

func listChunkPuts(srv *fakeapi.Server) map[int][]int {
	puts := map[int][]int{}
	for _, r := range srv.Requests() {
		if r.Chunk >= 0 {
			puts[r.Chunk] = append(puts[r.Chunk], r.Status)
		}
	}

	return puts
}
