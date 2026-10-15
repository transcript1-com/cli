package output

import (
	"io"
	"os"

	"golang.org/x/term"
)

const defaultWidth = 80

func getTerminalWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return defaultWidth
	}

	if !term.IsTerminal(int(f.Fd())) {
		return defaultWidth
	}

	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return defaultWidth
	}

	if width <= 0 {
		return defaultWidth
	}

	return width
}
