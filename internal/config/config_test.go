package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("AppData", t.TempDir())
}

func getEmptyEnv(string) string {
	return ""
}

func TestSaveLoadDelete(t *testing.T) {
	isolate(t)

	t.Run("loads nothing before a key is saved", func(t *testing.T) {
		got, err := LoadKey()
		if err != nil {
			t.Fatalf("LoadKey() error = %v, want nil", err)
		}

		if got != "" {
			t.Fatalf("LoadKey() = %q, want empty", got)
		}
	})

	t.Run("loads the key it saved", func(t *testing.T) {
		if err := SaveKey("t1_abc"); err != nil {
			t.Fatal(err)
		}

		got, err := LoadKey()
		if err != nil {
			t.Fatalf("LoadKey() error = %v, want nil", err)
		}

		if got != "t1_abc" {
			t.Fatalf("LoadKey() = %q, want t1_abc", got)
		}
	})

	t.Run("replaces a saved key", func(t *testing.T) {
		if err := SaveKey("t1_new"); err != nil {
			t.Fatal(err)
		}

		got, _ := LoadKey()
		if got != "t1_new" {
			t.Fatalf("LoadKey() = %q, want t1_new", got)
		}
	})

	t.Run("deletes the key and tolerates a second delete", func(t *testing.T) {
		if err := DeleteKey(); err != nil {
			t.Fatal(err)
		}

		if err := DeleteKey(); err != nil {
			t.Fatalf("second DeleteKey() = %v, want nil", err)
		}

		got, err := LoadKey()
		if err != nil {
			t.Fatalf("LoadKey() error = %v, want nil", err)
		}

		if got != "" {
			t.Fatalf("LoadKey() = %q, want empty", got)
		}
	})

	t.Run("leaves no temp files behind", func(t *testing.T) {
		if err := SaveKey("t1_abc"); err != nil {
			t.Fatal(err)
		}

		path, _ := KeyPath()
		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}

		if len(entries) != 1 {
			t.Fatalf("config folder has %d entries, want 1", len(entries))
		}
	})
}

func TestKeyFileModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix modes only")
	}

	isolate(t)
	if err := SaveKey("t1_abc"); err != nil {
		t.Fatal(err)
	}

	path, _ := KeyPath()

	t.Run("saves the file as 0600", func(t *testing.T) {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("file mode = %o, want 600", got)
		}
	})

	t.Run("creates the folder as 0700", func(t *testing.T) {
		info, err := os.Stat(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}

		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("folder mode = %o, want 700", got)
		}
	})
}

func TestResolveKeyPrefersEnv(t *testing.T) {
	isolate(t)
	if err := SaveKey("t1_file"); err != nil {
		t.Fatal(err)
	}

	getenv := func(name string) string {
		if name == EnvAPIKey {
			return "t1_env"
		}

		return ""
	}

	got, err := ResolveKey(getenv)
	if err != nil {
		t.Fatalf("ResolveKey() error = %v, want nil", err)
	}

	if got != "t1_env" {
		t.Fatalf("ResolveKey() = %q, want t1_env", got)
	}
}

func TestResolveKeyFile(t *testing.T) {
	isolate(t)
	if err := SaveKey("t1_file"); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveKey(getEmptyEnv)
	if err != nil {
		t.Fatalf("ResolveKey() error = %v, want nil", err)
	}

	if got != "t1_file" {
		t.Fatalf("ResolveKey() = %q, want t1_file", got)
	}
}

func TestResolveKeyMissing(t *testing.T) {
	isolate(t)
	got, err := ResolveKey(getEmptyEnv)
	if err != nil {
		t.Fatalf("ResolveKey() error = %v, want nil", err)
	}

	if got != "" {
		t.Fatalf("ResolveKey() = %q, want empty", got)
	}
}

func TestURLsDefaultAndOverride(t *testing.T) {
	t.Run("uses the production hosts by default", func(t *testing.T) {
		if got := APIURL(getEmptyEnv); got != "https://api.transcript1.com" {
			t.Fatalf("APIURL() = %q", got)
		}

		if got := DownloadsURL(getEmptyEnv); got != "https://downloads.transcript1.com" {
			t.Fatalf("DownloadsURL() = %q", got)
		}
	})

	t.Run("trims the trailing slash of an override", func(t *testing.T) {
		getenv := func(name string) string {
			switch name {
			case EnvAPIURL:
				return "http://127.0.0.1:3000/mock-api/"
			case EnvDownloadsURL:
				return "http://127.0.0.1:4000/"
			}

			return ""
		}

		if got := APIURL(getenv); got != "http://127.0.0.1:3000/mock-api" {
			t.Fatalf("APIURL() = %q", got)
		}

		if got := DownloadsURL(getenv); got != "http://127.0.0.1:4000" {
			t.Fatalf("DownloadsURL() = %q", got)
		}
	})
}
