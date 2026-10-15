package inputs

import (
	"path/filepath"
	"strings"
)

var mimeTypes = map[string]string{
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"m4a":  "audio/mp4",
	"flac": "audio/flac",
	"ogg":  "audio/ogg",
	"opus": "audio/opus",
	"mp4":  "video/mp4",
	"mov":  "video/quicktime",
	"webm": "video/webm",
	"mkv":  "video/x-matroska",
}

func MimeType(name string) string {
	return mimeTypes[strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))]
}
