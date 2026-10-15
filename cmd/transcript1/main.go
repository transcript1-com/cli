package main

import (
	"fmt"
	"os"

	"github.com/transcript1/cli/internal/version"
)

func isVersionFlag(args []string) bool {
	if len(args) != 2 {
		return false
	}

	return args[1] == "--version"
}

func main() {
	if isVersionFlag(os.Args) {
		fmt.Println("transcript1 " + version.Version)
		return
	}

	os.Exit(2)
}
