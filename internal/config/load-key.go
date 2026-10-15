package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func LoadKey() (string, error) {
	path, err := KeyPath()
	if err != nil {
		return "", err
	}

	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	var file keyFile
	if err := json.Unmarshal(body, &file); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	return file.APIKey, nil
}
