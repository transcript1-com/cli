package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func SaveKey(key string) error {
	path, err := KeyPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	body, err := json.Marshal(keyFile{APIKey: key})
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, fileName+".*.tmp")
	if err != nil {
		return err
	}

	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}
