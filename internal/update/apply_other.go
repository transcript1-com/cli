//go:build !windows

package update

import "os"

func replace(tmp, exe string) error {
	return os.Rename(tmp, exe)
}
