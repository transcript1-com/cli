package main

import "fmt"

type target struct{ os, arch string }

var targets = []target{
	{"darwin", "arm64"},
	{"darwin", "amd64"},
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"windows", "amd64"},
}

func (t target) getKey() string {
	return t.os + "_" + t.arch
}

func (t target) getBinary() string {
	if t.os == "windows" {
		return "transcript1.exe"
	}

	return "transcript1"
}

func (t target) getArchive(version string) string {
	ext := ".tar.gz"
	if t.os == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("transcript1_%s_%s%s", version, t.getKey(), ext)
}
