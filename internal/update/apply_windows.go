//go:build windows

package update

import "os"

// a running exe can be renamed but not overwritten
func replace(tmp, exe string) error {
	old := exe + oldSuffix
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}

	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		return err
	}

	return nil
}
