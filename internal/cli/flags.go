package cli

import (
	"flag"
	"io"
)

const (
	defaultJobs = 2
	minJobs     = 1
	maxJobs     = 20
)

type settings struct {
	format, out, language, source string
	speakers, timestamps          bool
	fresh, overwrite              bool
	noWait, json, quiet           bool
	jobs                          int
	version                       bool
}

func parseFlags(get bool, args []string) (*settings, []string, error) {
	s := &settings{}
	fs := flag.NewFlagSet("transcript1", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	defineFlags(fs, s, get)
	var raws []string
	for {
		if err := fs.Parse(args); err != nil {
			return s, nil, err
		}

		rest := fs.Args()
		if len(rest) == 0 {
			return s, raws, nil
		}

		if hasEndedFlags(args, rest) {
			return s, append(raws, rest...), nil
		}

		raws = append(raws, rest[0])
		args = rest[1:]
	}
}

func hasEndedFlags(args, rest []string) bool {
	consumed := args[:len(args)-len(rest)]
	if len(consumed) == 0 {
		return false
	}

	return consumed[len(consumed)-1] == "--"
}

func defineFlags(fs *flag.FlagSet, s *settings, get bool) {
	for _, name := range []string{"f", "format"} {
		fs.StringVar(&s.format, name, "text", "")
	}

	for _, name := range []string{"o", "out"} {
		fs.StringVar(&s.out, name, "", "")
	}

	for _, name := range []string{"q", "quiet"} {
		fs.BoolVar(&s.quiet, name, false, "")
	}

	fs.BoolVar(&s.timestamps, "timestamps", false, "")
	fs.BoolVar(&s.noWait, "no-wait", false, "")
	fs.BoolVar(&s.json, "json", false, "")
	fs.BoolVar(&s.version, "version", false, "")
	s.source = "auto"
	s.jobs = defaultJobs
	if get {
		return
	}

	fs.BoolVar(&s.speakers, "speakers", false, "")
	fs.BoolVar(&s.fresh, "fresh", false, "")
	fs.BoolVar(&s.overwrite, "overwrite", false, "")
	fs.StringVar(&s.language, "language", "auto", "")
	fs.StringVar(&s.source, "source", "auto", "")
	fs.IntVar(&s.jobs, "jobs", defaultJobs, "")
}
