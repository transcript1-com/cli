package fakeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

type segment struct {
	start, end float64
	speaker    string
	text       string
}

var segments = []segment{
	{0, 3.2, "Speaker 1", "Welcome to the show."},
	{3.2, 7.5, "Speaker 2", "Thanks for having me."},
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func isRetryable(status int) bool {
	if status == 429 {
		return true
	}

	return status >= 500
}

func writeError(w http.ResponseWriter, status int, code, message string, extra map[string]any) {
	e := map[string]any{
		"code":      code,
		"message":   message,
		"retryable": isRetryable(status),
	}
	for key, value := range extra {
		e[key] = value
	}

	writeJSON(w, status, map[string]any{"error": e})
}

func formatTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}

	return t.Format(time.RFC3339)
}

func isCaptionsTranscript(j *job) bool {
	if j.platform != "youtube" {
		return false
	}

	if j.source == "stt" {
		return false
	}

	return !j.diarize
}

func (s *Server) serialize(j *job, transcript bool) map[string]any {
	obj := map[string]any{
		"id":     j.id,
		"object": "transcription",
		"status": j.status,
		"options": map[string]any{
			"source":   j.source,
			"language": j.language,
			"diarize":  j.diarize,
		},
		"media":             nil,
		"transcript_source": nil,
		"language":          nil,
		"billed_minutes":    nil,
		"error":             nil,
		"created_at":        formatTime(j.createdAt),
		"completed_at":      formatTime(j.completedAt),
	}
	obj["source"] = map[string]any{
		"platform": "upload",
		"filename": j.filename,
		"size":     j.size,
	}
	if j.platform != "upload" {
		var mediaID any
		if j.videoID != "" {
			mediaID = j.videoID
		}

		obj["source"] = map[string]any{
			"platform": j.platform,
			"url":      j.url,
			"media_id": mediaID,
		}
		obj["media"] = map[string]any{"title": j.title, "duration_seconds": 60 * s.MinutesPerJob}
	}

	if j.status == api.StatusFailed {
		obj["error"] = map[string]any{"code": j.errCode, "message": "The job failed with " + j.errCode}
	}

	if j.status != api.StatusCompleted {
		return obj
	}

	obj["transcript_source"] = "stt"
	if isCaptionsTranscript(j) {
		obj["transcript_source"] = "youtube_captions"
	}

	obj["language"] = "en"
	obj["billed_minutes"] = j.charged
	if transcript {
		obj["transcript"] = j.buildTranscript()
	}

	return obj
}

func (j *job) listSegments() []segment {
	out := make([]segment, len(segments))
	for i, seg := range segments {
		out[i] = seg
		if !j.diarize {
			out[i].speaker = ""
		}
	}

	return out
}

func (j *job) buildTranscript() map[string]any {
	var texts []string
	var rows []map[string]any
	for _, seg := range j.listSegments() {
		texts = append(texts, seg.text)
		var speaker any
		if seg.speaker != "" {
			speaker = seg.speaker
		}

		rows = append(rows, map[string]any{
			"start":    seg.start,
			"end":      seg.end,
			"speaker":  speaker,
			"language": "en",
			"text":     seg.text,
		})
	}

	return map[string]any{
		"text":     strings.Join(texts, " "),
		"speakers": []string{},
		"segments": rows,
	}
}

func formatClock(seconds float64) string {
	total := int(seconds)
	return fmt.Sprintf("%02d:%02d:%02d", total/3600, total/60%60, total%60)
}

func formatCueTime(seconds float64, sep string) string {
	ms := int(seconds*1000+0.5) % 1000
	return fmt.Sprintf("%s%s%03d", formatClock(seconds), sep, ms)
}

func buildPrefix(seg segment) string {
	if seg.speaker == "" {
		return ""
	}

	return seg.speaker + ": "
}

func (j *job) render(format string, timestamps bool) (string, string) {
	var b strings.Builder
	switch format {
	case "srt":
		for i, seg := range j.listSegments() {
			if i > 0 {
				b.WriteString("\n")
			}

			fmt.Fprintf(&b, "%d\n%s --> %s\n%s%s\n", i+1, formatCueTime(seg.start, ","), formatCueTime(seg.end, ","), buildPrefix(seg), seg.text)
		}

		return "application/x-subrip", b.String()
	case "vtt":
		b.WriteString("WEBVTT\n\n")
		for i, seg := range j.listSegments() {
			if i > 0 {
				b.WriteString("\n")
			}

			voice := ""
			if seg.speaker != "" {
				voice = "<v " + seg.speaker + ">"
			}

			fmt.Fprintf(&b, "%s --> %s\n%s%s\n", formatCueTime(seg.start, "."), formatCueTime(seg.end, "."), voice, seg.text)
		}

		return "text/vtt", b.String()
	}

	for _, seg := range j.listSegments() {
		if timestamps {
			b.WriteString("[" + formatClock(seg.start) + "] ")
		}

		b.WriteString(buildPrefix(seg) + seg.text + "\n")
	}

	return "text/plain; charset=utf-8", b.String()
}
