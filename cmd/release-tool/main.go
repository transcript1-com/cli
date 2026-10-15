package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	defaultPub = "internal/update/release-key.pub"
	usageText  = "usage: release-tool keygen|package|manifest [flags]"
)

var errUsage = errors.New(usageText)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usageText)
		return 2
	}

	var err error
	switch args[0] {
	case "keygen":
		err = generateKeys(args[1:], stdout)
	case "package":
		err = pack(args[1:], stdout)
	case "manifest":
		err = writeManifest(args[1:], stdout)
	default:
		err = errUsage
	}

	if err != nil {
		fmt.Fprintf(stderr, "release-tool: %s\n", err)
		return 1
	}

	return 0
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("release-tool "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return fs
}
