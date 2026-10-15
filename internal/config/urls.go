package config

import "strings"

func APIURL(getenv func(string) string) string {
	return getURL(getenv, EnvAPIURL, DefaultAPIURL)
}

func DownloadsURL(getenv func(string) string) string {
	return getURL(getenv, EnvDownloadsURL, DefaultDownloadsURL)
}

func getURL(getenv func(string) string, name, fallback string) string {
	value := strings.TrimRight(getenv(name), "/")
	if value == "" {
		return fallback
	}

	return value
}
