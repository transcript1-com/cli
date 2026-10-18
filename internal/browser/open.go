package browser

import (
	"os/exec"
	"runtime"
)

// BuildCommand returns the command that opens url; the url is always a single argument.
func BuildCommand(goos, url string) (string, []string) {
	if goos == "darwin" {
		return "open", []string{url}
	}

	if goos == "windows" {
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	}

	return "xdg-open", []string{url}
}

// ShouldOpen reports whether a browser can be opened from this session.
func ShouldOpen(goos string, getenv func(string) string, noBrowser bool) bool {
	if noBrowser {
		return false
	}

	if getenv("SSH_CONNECTION") != "" {
		return false
	}

	if getenv("SSH_TTY") != "" {
		return false
	}

	if goos == "darwin" {
		return true
	}

	if goos == "windows" {
		return true
	}

	if getenv("DISPLAY") != "" {
		return true
	}

	return getenv("WAYLAND_DISPLAY") != ""
}

// Open starts the system browser on url without waiting for it.
func Open(url string) error {
	name, args := BuildCommand(runtime.GOOS, url)
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}

	go cmd.Wait()

	return nil
}
