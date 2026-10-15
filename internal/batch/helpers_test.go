package batch

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/fakeapi"
	"github.com/transcript1/cli/internal/inputs"
)

const rickID = "dQw4w9WgXcQ"

type events struct {
	mu   sync.Mutex
	list []Event
}

func (e *events) emit(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, ev)
}

func (e *events) listStages() []Stage {
	e.mu.Lock()
	defer e.mu.Unlock()
	stages := make([]Stage, len(e.list))
	for i, ev := range e.list {
		stages[i] = ev.Stage
	}

	return stages
}

func (e *events) has(stage Stage) bool {
	for _, s := range e.listStages() {
		if s == stage {
			return true
		}
	}

	return false
}

func setup(t *testing.T) (*fakeapi.Server, *api.Client) {
	t.Helper()
	srv := fakeapi.New(t)
	c := api.New(srv.URL, srv.Key)
	c.Sleep = func(context.Context, time.Duration) error { return nil }

	return srv, c
}

func buildLink(id string) inputs.Input {
	url := "https://www.youtube.com/watch?v=" + id
	in := inputs.Input{
		Raw:     url,
		Kind:    inputs.Link,
		URL:     url,
		VideoID: id,
	}

	return in
}

func buildFileInput(t *testing.T, name string) inputs.Input {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	data := []byte("0123456789abcdefghij")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	in := inputs.Input{
		Raw:  path,
		Kind: inputs.File,
		Path: path,
		Name: name,
		Size: int64(len(data)),
	}

	return in
}

func run(t *testing.T, c *api.Client, in inputs.Input, opt Options) Result {
	t.Helper()
	return runOne(context.Background(), c, 0, in, opt, false, nil)
}
