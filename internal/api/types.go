package api

import (
	"encoding/json"
	"slices"
)

const (
	StatusAwaitingUpload = "awaiting_upload"
	StatusQueued         = "queued"
	StatusFetching       = "fetching"
	StatusTranscribing   = "transcribing"
	StatusCompleted      = "completed"
	StatusFailed         = "failed"
	StatusCanceled       = "canceled"
)

type Transcription struct {
	ID               string          `json:"id"`
	Status           string          `json:"status"`
	Source           Source          `json:"source"`
	Media            *Media          `json:"media"`
	TranscriptSource *string         `json:"transcript_source"`
	Language         *string         `json:"language"`
	BilledMinutes    *int            `json:"billed_minutes"`
	Reused           bool            `json:"reused"`
	Error            *JobError       `json:"error"`
	Upload           *UploadPlan     `json:"upload"`
	Raw              json.RawMessage `json:"-"`
}

type Source struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	MediaID  string `json:"media_id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type Media struct {
	Title           string   `json:"title"`
	DurationSeconds *float64 `json:"duration_seconds"`
}

type JobError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UploadPlan struct {
	ChunkURLTemplate string            `json:"chunk_url_template"`
	Headers          map[string]string `json:"headers"`
	ExpiresAt        string            `json:"expires_at"`
	Chunks           []Chunk           `json:"chunks"`
}

type Chunk struct {
	Index  int   `json:"index"`
	Offset int64 `json:"offset"`
	Size   int64 `json:"size"`
}

type UploadProgress struct {
	UploadedBytes  int64   `json:"uploaded_bytes"`
	ReceivedChunks []int   `json:"received_chunks"`
	Chunks         []Chunk `json:"chunks"`
}

type Account struct {
	User struct {
		Email string `json:"email"`
	} `json:"user"`
	Plan *struct {
		Name string `json:"name"`
	} `json:"plan"`
	Minutes struct {
		Remaining int `json:"remaining"`
	} `json:"minutes"`
	Limits struct {
		MaxActiveJobs int `json:"max_active_jobs"`
	} `json:"limits"`
}

type CreateOptions struct {
	Source   string
	Language string
	Diarize  bool
}

func (t *Transcription) Terminal() bool {
	return slices.Contains([]string{StatusCompleted, StatusFailed, StatusCanceled}, t.Status)
}

func (o CreateOptions) getSource() string {
	if o.Source == "" {
		return "auto"
	}

	return o.Source
}

func (o CreateOptions) getLanguage() string {
	if o.Language == "" {
		return "auto"
	}

	return o.Language
}
