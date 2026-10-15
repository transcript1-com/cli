package inputs

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var (
	youtubeHosts = map[string]bool{
		"youtube.com":       true,
		"m.youtube.com":     true,
		"music.youtube.com": true,
		"youtu.be":          true,
	}
	youtubePrefixes = map[string]bool{
		"shorts": true,
		"embed":  true,
		"live":   true,
	}
	youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
)

func findShortLinkID(segments []string) string {
	if len(segments) == 0 {
		return ""
	}

	return segments[0]
}

func findVideoID(u *url.URL, host string, segments []string) string {
	if host == "youtu.be" {
		return findShortLinkID(segments)
	}

	if len(segments) == 1 {
		if segments[0] != "watch" {
			return ""
		}

		return u.Query().Get("v")
	}

	if len(segments) != 2 {
		return ""
	}

	if !youtubePrefixes[segments[0]] {
		return ""
	}

	return segments[1]
}

func ParseYouTube(raw string) (videoID, canonical string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", false
	}

	if !slices.Contains([]string{"http", "https"}, u.Scheme) {
		return "", "", false
	}

	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if !youtubeHosts[host] {
		return "", "", false
	}

	var segments []string
	for _, s := range strings.Split(u.Path, "/") {
		if s != "" {
			segments = append(segments, s)
		}
	}

	id := findVideoID(u, host, segments)
	if !youtubeID.MatchString(id) {
		return "", "", false
	}

	return id, "https://www.youtube.com/watch?v=" + id, true
}
