package output

import (
	"io"
	"os"

	"golang.org/x/sys/windows"
)

// windows consoles print escape codes as text until enabled
func EnableEscapes(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return true
	}

	h := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return false
	}

	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}

	return windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
