package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/batch"
	"github.com/transcript1/cli/internal/config"
	"github.com/transcript1/cli/internal/files"
)

const (
	tokensURL          = "https://transcript1.com/dashboard/account/tokens"
	errWriteToTemplate = "cannot write to %s: %w"
)

var errNoKey = errors.New("no API key: run transcript1 login or set TRANSCRIPT1_API_KEY")

func (s *settings) buildOptions(n int) batch.Options {
	dir, file := files.Target(s.out, n)

	return batch.Options{
		Format:     s.format,
		Timestamps: s.timestamps,
		Speakers:   s.speakers,
		Language:   s.language,
		Source:     s.source,
		Fresh:      s.fresh,
		Overwrite:  s.overwrite,
		NoWait:     s.noWait,
		JSON:       s.json,
		Jobs:       s.jobs,
		Dir:        dir,
		File:       file,
	}
}

// prove the folder writable before anything is charged
func prepareOutput(opt batch.Options) error {
	dir := opt.Dir
	if opt.File != "" {
		dir = filepath.Dir(opt.File)
	}

	if opt.NoWait {
		return nil
	}

	if dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return buildOutputError(dir, err)
	}

	probe, err := os.CreateTemp(dir, ".transcript1-*.tmp")
	if err != nil {
		return buildOutputError(dir, err)
	}

	probe.Close()

	return os.Remove(probe.Name())
}

func buildOutputError(dir string, err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}

	return fmt.Errorf(errWriteToTemplate, dir, err)
}

func (e Env) buildClient() (*api.Client, error) {
	key, err := config.ResolveKey(e.Getenv)
	if err != nil {
		return nil, err
	}

	if key == "" {
		return nil, errNoKey
	}

	return api.New(config.APIURL(e.Getenv), key), nil
}
