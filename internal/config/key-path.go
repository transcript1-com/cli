package config

import (
	"os"
	"path/filepath"
)

func KeyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, dirName, fileName), nil
}
