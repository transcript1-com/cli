package config

import (
	"errors"
	"io/fs"
	"os"
)

func DeleteKey() error {
	path, err := KeyPath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}
