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
	host := strings.Map(dropControl, hostname)
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

func dropControl(r rune) rune {
	if unicode.IsControl(r) {
		return -1
	}

	return r
}
