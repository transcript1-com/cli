package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/transcript1/cli/internal/files"
)

func (r *runner) isToFile() bool {
	if r.opt.Dir != "" {
		return true
	}

	return r.opt.File != ""
}

func (r *runner) isJSONToStdout() bool {
	if !r.opt.JSON {
		return false
	}

	return !r.isToFile()
}

func (r *runner) writeOutput(ctx context.Context, res Result) Result {
	if r.isJSONToStdout() {
		return r.writeObject(ctx, res)
	}

	data, err := r.buildContent(ctx, &res)
	if err != nil {
		return markFailed(res, err)
	}

	res.Status = Done
	if !r.isToFile() {
		res.Body = data
		return res
	}

	path := r.buildPath(res)
	if err := files.WriteAtomic(path, data); err != nil {
		return markFailed(res, err)
	}

	res.File = path

	return res
}

func (r *runner) buildPath(res Result) string {
	if r.opt.File != "" {
		return r.opt.File
	}

	title := ""
	if res.T.Media != nil {
		title = res.T.Media.Title
	}

	return filepath.Join(r.opt.Dir, files.Name(r.in, title, r.getFormat(), r.fullName))
}

func (r *runner) buildContent(ctx context.Context, res *Result) ([]byte, error) {
	if r.getFormat() != files.FormatJSON {
		data, err := r.c.Render(ctx, res.T.ID, r.getFormat(), r.opt.Timestamps)
		return addNewline(data), err
	}

	if err := r.loadTranscript(ctx, res); err != nil {
		return nil, err
	}

	return addNewline(res.T.Raw), nil
}

func (r *runner) writeObject(ctx context.Context, res Result) Result {
	if err := r.loadTranscript(ctx, &res); err != nil {
		return markFailed(res, err)
	}

	res.Status = Done

	return res
}

func (r *runner) loadTranscript(ctx context.Context, res *Result) error {
	if hasTranscript(res.T.Raw) {
		return nil
	}

	t, err := r.c.Get(ctx, res.T.ID, 0)
	if err != nil {
		return err
	}

	t.Reused = res.T.Reused
	res.T = t

	return nil
}

// a reused job can come back without its transcript
func hasTranscript(raw []byte) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}

	_, ok := obj["transcript"]

	return ok
}

func addNewline(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	if bytes.HasSuffix(data, []byte("\n")) {
		return data
	}

	return append(append([]byte(nil), data...), '\n')
}
