package update

import (
	"crypto/ed25519"
	_ "embed"
	"encoding/base64"
	"errors"
	"strings"
)

const unsetKey = "unset"

//go:embed release-key.pub
var releaseKey string

var errBadKey = errors.New("release public key is not a base64 ed25519 key")

func PublicKey() (ed25519.PublicKey, error) {
	return ParsePublicKey(releaseKey)
}

func ParsePublicKey(text string) (ed25519.PublicKey, error) {
	text = strings.TrimSpace(text)
	if text == unsetKey {
		return nil, ErrNotConfigured
	}

	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, errBadKey
	}

	if len(raw) != ed25519.PublicKeySize {
		return nil, errBadKey
	}

	return ed25519.PublicKey(raw), nil
}
