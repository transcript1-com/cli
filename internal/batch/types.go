package batch

import (
	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

type Options struct {
	Format               string
	Timestamps, Speakers bool
	Language, Source     string
	Fresh, Overwrite     bool
	NoWait, JSON         bool
	Jobs                 int
	Dir, File            string
}

type Status int

const (
	NotStarted Status = iota
	Done
	Failed
	Skipped
	SkippedMinutes
	Started
	StillRunning
	Interrupted
)

type Result struct {
	Index  int
	Input  inputs.Input
	Status Status
	T      *api.Transcription
	Body   []byte
	File   string
	Err    error
}

type Stage int

const (
	WaitingSlot Stage = iota
	Uploading
	Queued
	Transcribing
	Finished
)

type Event struct {
	Index       int
	Input       inputs.Input
	Stage       Stage
	Sent, Total int64
	Result      *Result
}
