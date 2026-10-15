package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type fakeDownloads struct {
	*httptest.Server
	manifest, sig []byte
	archive       []byte
	requests      atomic.Int32
}

func newKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return pub, priv
}

func useKey(t *testing.T, pub ed25519.PublicKey) {
	t.Helper()
	saved := releaseKey
	releaseKey = base64.StdEncoding.EncodeToString(pub) + "\n"
	t.Cleanup(func() { releaseKey = saved })
}

func newDownloads(t *testing.T) *fakeDownloads {
	t.Helper()
	d := &fakeDownloads{}
	d.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.requests.Add(1)
		switch r.URL.Path {
		case "/cli/latest.json":
			w.Write(d.manifest)
		case "/cli/latest.json.sig":
			w.Write(d.sig)
		case "/archive":
			w.Write(d.archive)
		default:
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(d.Close)

	return d
}

func (d *fakeDownloads) publish(t *testing.T, priv ed25519.PrivateKey, version string, archive []byte) *Manifest {
	t.Helper()
	sum := sha256.Sum256(archive)
	asset := Asset{
		URL:    d.URL + "/archive",
		SHA256: hex.EncodeToString(sum[:]),
		Size:   int64(len(archive)),
	}
	m := &Manifest{
		Version:     version,
		PublishedAt: "2026-10-15T12:00:00Z",
		Assets:      map[string]Asset{Platform(): asset},
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	d.manifest = data
	d.archive = archive
	d.sig = []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data)) + "\n")

	return m
}

func buildTarGz(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}
