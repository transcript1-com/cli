package cli

import (
	"strings"
	"unicode"
)

const (
	keyNamePrefix = "transcript1 CLI ("
	maxHostRunes  = 64
)

// mirrors the name the api gives a key created by the browser login
func buildKeyName(hostname string) string {
	host := strings.Map(dropInvisible, hostname)
	host = strings.TrimSpace(host)
	runes := []rune(host)
	if len(runes) > maxHostRunes {
		runes = runes[:maxHostRunes]
	}

	host = strings.TrimRightFunc(string(runes), unicode.IsSpace)
	if host == "" {
		host = unknownHost
	}

	return keyNamePrefix + host + ")"
}

// same set as the api: Cc, Cf, Zl and Zp
func dropInvisible(r rune) rune {
	if unicode.Is(unicode.Cc, r) {
		return -1
	}

	if unicode.Is(unicode.Cf, r) {
		return -1
	}

	if unicode.Is(unicode.Zl, r) {
		return -1
	}

	if unicode.Is(unicode.Zp, r) {
		return -1
	}

	return r
}
