package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	errReplaceKeyTemplate = "refusing to replace the private key: %w"
	msgKeysWritten        = "wrote %s (back it up, keep it out of the repo) and %s\n"
)

var errKeygenUsage = errors.New("usage: release-tool keygen [--pub <file>] <private-key-path>")

func generateKeys(args []string, stdout io.Writer) error {
	fs := newFlags("keygen")
	pubPath := fs.String("pub", defaultPub, "")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errKeygenUsage
	}

	keyPath := fs.Arg(0)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf(errReplaceKeyTemplate, err)
	}

	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(priv) + "\n"); err != nil {
		f.Close()
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	if err := os.WriteFile(*pubPath, []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(stdout, msgKeysWritten, keyPath, *pubPath)

	return nil
}
