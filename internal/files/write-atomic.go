package files

import (
	"fmt"
	"os"
	"path/filepath"
)

func buildPathError(path string, err error) error {
	return fmt.Errorf("%s: %w", path, err)
}

func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return buildPathError(path, err)
	}

	tmp, err := os.CreateTemp(dir, ".transcript1-*")
	if err != nil {
		return buildPathError(path, err)
	}

	if err := fill(tmp, data); err != nil {
		os.Remove(tmp.Name())
		return buildPathError(path, err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return buildPathError(path, err)
	}

	return nil
}

func fill(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}

	if err := f.Chmod(0o644); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}
