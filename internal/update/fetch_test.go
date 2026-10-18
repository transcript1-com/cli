package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"net/http"
	"strings"
	"testing"
)

func TestFetchVerifies(t *testing.T) {
	pub, priv := newKey(t)

	t.Run("returns the manifest when the signature verifies", func(t *testing.T) {
		d := newDownloads(t)
		d.publish(t, priv, "1.2.0", []byte("archive"))
		m, err := Fetch(context.Background(), http.DefaultClient, d.URL, pub)
		if err != nil {
			t.Fatal(err)
		}

		if m.Version != "1.2.0" {
			t.Fatalf("manifest %+v", m)
		}

		if m.Assets[Platform()].Size != 7 {
			t.Fatalf("manifest %+v", m)
		}
	})

	t.Run("refuses a manifest with one changed byte", func(t *testing.T) {
		d := newDownloads(t)
		d.publish(t, priv, "1.2.0", []byte("archive"))
		d.manifest = bytes.Replace(d.manifest, []byte("1.2.0"), []byte("1.3.0"), 1)
		if _, err := Fetch(context.Background(), http.DefaultClient, d.URL, pub); err == nil {
			t.Fatal("no error")
		}
	})

	t.Run("refuses a signature by another key", func(t *testing.T) {
		d := newDownloads(t)
		_, other := newKey(t)
		d.publish(t, other, "1.2.0", []byte("archive"))
		_, err := Fetch(context.Background(), http.DefaultClient, d.URL, pub)
		hint := "(a release may be in progress; try again in a few minutes)"
		if err == nil {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), hint) {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("refuses a missing signature", func(t *testing.T) {
		d := newDownloads(t)
		d.publish(t, priv, "1.2.0", []byte("archive"))
		d.sig = nil
		if _, err := Fetch(context.Background(), http.DefaultClient, d.URL, pub); err == nil {
			t.Fatal("no error")
		}
	})
}

func TestPublicKey(t *testing.T) {
	t.Run("reports an unset key as not configured", func(t *testing.T) {
		if _, err := ParsePublicKey("unset\n"); err != ErrNotConfigured {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("refuses a key of the wrong length", func(t *testing.T) {
		if _, err := ParsePublicKey("YWJj"); err == nil {
			t.Fatal("no error")
		}
	})

	t.Run("ships a release key", func(t *testing.T) {
		key, err := PublicKey()
		if err != nil {
			t.Fatalf("err %v", err)
		}

		if len(key) != ed25519.PublicKeySize {
			t.Fatalf("key length %d", len(key))
		}
	})
}
