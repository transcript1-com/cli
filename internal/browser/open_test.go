package browser

import (
	"reflect"
	"testing"
)

const loginURL = "https://transcript1.com/cli/login?code=WXYZ-2345&x=1"

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		goos string
		name string
		args []string
	}{
		{"darwin", "open", []string{loginURL}},
		{"linux", "xdg-open", []string{loginURL}},
		{"freebsd", "xdg-open", []string{loginURL}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", loginURL}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, args := BuildCommand(tt.goos, loginURL)
			if name != tt.name {
				t.Fatalf("name = %q, want %q", name, tt.name)
			}

			if !reflect.DeepEqual(args, tt.args) {
				t.Fatalf("args = %q, want %q", args, tt.args)
			}
		})
	}
}

func TestShouldOpen(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		env       map[string]string
		noBrowser bool
		want      bool
	}{
		{"no-browser flag on darwin", "darwin", nil, true, false},
		{"ssh connection on darwin", "darwin", map[string]string{"SSH_CONNECTION": "1.2.3.4 22 5.6.7.8 22"}, false, false},
		{"ssh tty on darwin", "darwin", map[string]string{"SSH_TTY": "/dev/pts/0"}, false, false},
		{"linux without display", "linux", nil, false, false},
		{"linux with DISPLAY", "linux", map[string]string{"DISPLAY": ":0"}, false, true},
		{"linux with WAYLAND_DISPLAY", "linux", map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, false, true},
		{"darwin bare", "darwin", nil, false, true},
		{"windows bare", "windows", nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(name string) string { return tt.env[name] }
			if got := ShouldOpen(tt.goos, getenv, tt.noBrowser); got != tt.want {
				t.Fatalf("ShouldOpen = %v, want %v", got, tt.want)
			}
		})
	}
}
