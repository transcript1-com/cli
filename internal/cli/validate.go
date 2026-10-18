package cli

import (
	"errors"
	"fmt"
	"slices"

	"github.com/transcript1-com/cli/internal/files"
	"github.com/transcript1-com/cli/internal/inputs"
)

const (
	sourceAuto     = "auto"
	sourceCaptions = "captions"
	sourceSTT      = "stt"

	errFormatTemplate   = "unknown format %q: use text, srt, vtt or json"
	errSourceTemplate   = "unknown source %q: use auto, captions or stt"
	errJobsTemplate     = "--jobs must be between %d and %d"
	errCaptionsTemplate = "--source captions works with YouTube links only, and %s is a file"
)

var (
	formats = []string{files.FormatText, files.FormatSRT, files.FormatVTT, files.FormatJSON}
	sources = []string{sourceAuto, sourceCaptions, sourceSTT}

	errNoInputs   = errors.New("no inputs: pass a YouTube link, a file or a folder")
	errTimestamps = errors.New("--timestamps works with --format text only")
	errSpeakers   = errors.New("--speakers needs speech-to-text and cannot be used with --source captions")
)

func hasMisusedTimestamps(s *settings) bool {
	if !s.timestamps {
		return false
	}

	return s.format != files.FormatText
}

func hasMisusedSpeakers(s *settings) bool {
	if !s.speakers {
		return false
	}

	return s.source == sourceCaptions
}

func validateFlags(s *settings) error {
	if !slices.Contains(formats, s.format) {
		return fmt.Errorf(errFormatTemplate, s.format)
	}

	if !slices.Contains(sources, s.source) {
		return fmt.Errorf(errSourceTemplate, s.source)
	}

	if hasMisusedTimestamps(s) {
		return errTimestamps
	}

	if hasMisusedSpeakers(s) {
		return errSpeakers
	}

	if s.jobs < minJobs {
		return fmt.Errorf(errJobsTemplate, minJobs, maxJobs)
	}

	if s.jobs > maxJobs {
		return fmt.Errorf(errJobsTemplate, minJobs, maxJobs)
	}

	return nil
}

func validateInputs(s *settings, ins []inputs.Input) error {
	if s.source != sourceCaptions {
		return nil
	}

	for _, in := range ins {
		if in.Kind == inputs.File {
			return fmt.Errorf(errCaptionsTemplate, in.Raw)
		}
	}

	return nil
}
