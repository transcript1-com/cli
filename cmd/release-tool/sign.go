package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/transcript1-com/cli/internal/update"
)

const errKeyMismatchTemplate = "release key does not match %s"

var errBadPrivateKey = errors.New("private key is not a base64 ed25519 key")

func sign(priv ed25519.PrivateKey, data []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data)) + "\n")
}

func readPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, errBadPrivateKey
	}

	if len(raw) != ed25519.PrivateKeySize {
		return nil, errBadPrivateKey
	}

	return ed25519.PrivateKey(raw), nil
}

func loadMatchingKey(keyPath, pubPath string) (ed25519.PrivateKey, error) {
	priv, err := readPrivateKey(keyPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, err
	}

	pub, err := update.ParsePublicKey(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", pubPath, err)
	}

	if !pub.Equal(priv.Public()) {
		return nil, fmt.Errorf(errKeyMismatchTemplate, pubPath)
	}

	return priv, nil
}
