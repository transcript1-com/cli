package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func LoadKey() (string, error) {
	file, err := LoadKeyFile()
	if err != nil {
		return "", err
	}

	return file.APIKey, nil
}

func LoadKeyFile() (KeyFile, error) {
	path, err := KeyPath()
	if err != nil {
		return KeyFile{}, err
	}

	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return KeyFile{}, nil
	}

	if err != nil {
		return KeyFile{}, err
	}

	var file KeyFile
	if err := json.Unmarshal(body, &file); err != nil {
		return KeyFile{}, fmt.Errorf("read %s: %w", path, err)
	}

	return file, nil
}
