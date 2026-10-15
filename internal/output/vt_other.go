//go:build !windows

package output

import "io"

func EnableEscapes(io.Writer) bool {
	return true
}
