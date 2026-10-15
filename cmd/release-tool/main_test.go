package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/update"
)

type tool struct {
	t   *testing.T
	dir string
}

func (tl tool) run(args ...string) (int, string) {
	var out, errw bytes.Buffer
	code := run(args, &out, &errw)

	return code, out.String() + errw.String()
}

func (tl tool) mustRun(args ...string) {
	tl.t.Helper()
	if code, out := tl.run(args...); code != 0 {
		tl.t.Fatalf("%v: code %d %s", args, code, out)
	}
}

func (tl tool) buildPath(parts ...string) string {
	return filepath.Join(append([]string{tl.dir}, parts...)...)
}

func (tl tool) write(name, content string) {
	tl.t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		tl.t.Fatal(err)
	}

	if err := os.WriteFile(name, []byte(content), 0o755); err != nil {
		tl.t.Fatal(err)
	}
}

func (tl tool) read(name string) []byte {
	tl.t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		tl.t.Fatal(err)
	}

	return data
}

func (tl tool) build(version string) string {
	dist := tl.buildPath("dist")
	for _, target := range targets {
		tl.write(filepath.Join(dist, "bin", target.getKey(), target.getBinary()), "binary "+target.getKey())
	}

	tl.write(tl.buildPath("root", "README.md"), "# transcript1\n")
	tl.write(tl.buildPath("root", "LICENSE"), "MIT\n")
	tl.mustRun("package", "--version", version, "--dist", dist, "--root", tl.buildPath("root"))

	return dist
}

func TestReleaseToolRoundTrip(t *testing.T) {
	tl := tool{t: t, dir: t.TempDir()}
	key := tl.buildPath("keys", "release.key")
	pub := tl.buildPath("release-key.pub")
	tl.write(pub, "unset\n")

	t.Run("keygen writes a private key only the owner can read", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Dir(key), 0o700); err != nil {
			t.Fatal(err)
		}

		tl.mustRun("keygen", "--pub", pub, key)
		info, err := os.Stat(key)
		if err != nil {
			t.Fatalf("key %v %v", info, err)
		}

		if info.Mode().Perm() != 0o600 {
			t.Fatalf("key %v %v", info, err)
		}

		if _, err := update.ParsePublicKey(string(tl.read(pub))); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("keygen refuses to overwrite a key", func(t *testing.T) {
		before := tl.read(key)
		if code, _ := tl.run("keygen", "--pub", pub, key); code == 0 {
			t.Fatal("overwrote the key")
		}

		if !bytes.Equal(before, tl.read(key)) {
			t.Fatal("key changed")
		}
	})

	dist := tl.build("1.2.0")

	t.Run("package writes five archives and sha256sum checksums", func(t *testing.T) {
		lines := strings.Split(strings.TrimSuffix(string(tl.read(filepath.Join(dist, "checksums.txt"))), "\n"), "\n")
		if len(lines) != 5 {
			t.Fatalf("checksums %q", lines)
		}

		if !sort.StringsAreSorted(listNames(lines)) {
			t.Fatalf("checksums %q", lines)
		}

		for _, line := range lines {
			sum, name, ok := strings.Cut(line, "  ")
			got := sha256.Sum256(tl.read(filepath.Join(dist, name)))
			if !ok {
				t.Fatalf("line %q", line)
			}

			if sum != hex.EncodeToString(got[:]) {
				t.Fatalf("line %q", line)
			}
		}
	})

	t.Run("package puts the binary, readme and license at the archive root", func(t *testing.T) {
		entries := listTarEntries(t, tl.read(filepath.Join(dist, "transcript1_1.2.0_linux_amd64.tar.gz")))
		if entries["transcript1"] != 0o755 {
			t.Fatalf("entries %v", entries)
		}

		if len(entries) != 3 {
			t.Fatalf("entries %v", entries)
		}

		if entries["README.md"] == 0 {
			t.Fatalf("entries %v", entries)
		}

		if entries["LICENSE"] == 0 {
			t.Fatalf("entries %v", entries)
		}

		data := tl.read(filepath.Join(dist, "transcript1_1.2.0_windows_amd64.zip"))
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatalf("zip %v", err)
		}

		if len(zr.File) != 3 {
			t.Fatalf("zip %v", err)
		}

		if zr.File[0].Name != "transcript1.exe" {
			t.Fatalf("zip %v", err)
		}
	})

	t.Run("manifest is signed and lists five assets", func(t *testing.T) {
		tl.mustRun("manifest", "--version", "1.2.0", "--dist", dist, "--base-url", "https://downloads.example.com/",
			"--key", key, "--pub", pub)
		data := tl.read(filepath.Join(dist, "latest.json"))
		encoded := tl.read(filepath.Join(dist, "latest.json.sig"))
		sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
		pk, _ := update.ParsePublicKey(string(tl.read(pub)))
		if err != nil {
			t.Fatalf("signature does not verify: %v", err)
		}

		if !ed25519.Verify(pk, data, sig) {
			t.Fatalf("signature does not verify: %v", err)
		}

		if !bytes.HasPrefix(data, []byte("{\n  \"assets\": {\n")) {
			t.Fatalf("latest.json %s", data)
		}

		if !bytes.HasSuffix(data, []byte("}\n")) {
			t.Fatalf("latest.json %s", data)
		}

		var m update.Manifest
		err = json.Unmarshal(data, &m)
		if err != nil {
			t.Fatalf("manifest %+v %v", m, err)
		}

		if m.Version != "1.2.0" {
			t.Fatalf("manifest %+v %v", m, err)
		}

		if len(m.Assets) != 5 {
			t.Fatalf("manifest %+v %v", m, err)
		}

		a := m.Assets["darwin_arm64"]
		base := "https://downloads.example.com/cli/releases/1.2.0/"
		if a.URL != base+"transcript1_1.2.0_darwin_arm64.tar.gz" {
			t.Fatalf("asset %+v", a)
		}

		if a.Size == 0 {
			t.Fatalf("asset %+v", a)
		}

		if len(a.SHA256) != 64 {
			t.Fatalf("asset %+v", a)
		}
	})

	t.Run("manifest refuses a key that does not match the public key", func(t *testing.T) {
		other := tl.buildPath("keys", "other.key")
		tl.mustRun("keygen", "--pub", tl.buildPath("other.pub"), other)
		code, out := tl.run("manifest", "--version", "1.2.0", "--dist", dist, "--base-url", "https://d.example.com",
			"--key", other, "--pub", pub)
		if code == 0 {
			t.Fatalf("code %d %s", code, out)
		}

		if !strings.Contains(out, "release key does not match") {
			t.Fatalf("code %d %s", code, out)
		}
	})
}

func listNames(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		_, out[i], _ = strings.Cut(line, "  ")
	}

	return out
}

func listTarEntries(t *testing.T, data []byte) map[string]int64 {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	tr := tar.NewReader(gz)
	out := map[string]int64{}
	for {
		hdr, err := tr.Next()
		if err != nil {
			return out
		}

		out[hdr.Name] = hdr.Mode
	}
}
