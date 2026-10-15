package fakeapi

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

func needsSpeechToText(body createBody) bool {
	if !body.Diarize {
		return false
	}

	return body.Source == "captions"
}

func (s *Server) createLink(w http.ResponseWriter, key string, reuse bool, body createBody) {
	if body.Platform != "youtube" {
		writeError(w, http.StatusUnprocessableEntity, api.CodeUnsupportedPlatform, "Unsupported platform",
			map[string]any{"supported_platforms": []string{"youtube"}})
		return
	}

	videoID, canonical, ok := inputs.ParseYouTube(body.URL)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidURL, "Not a YouTube video link", nil)
		return
	}

	if !slices.Contains([]string{"auto", "captions", "stt"}, body.Source) {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "source must be auto, captions or stt", nil)
		return
	}

	if needsSpeechToText(body) {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "diarize needs speech-to-text", nil)
		return
	}

	params := fmt.Sprintf("link|%s|%s|%s|%t", videoID, body.Source, body.Language, body.Diarize)
	if s.replay(w, key, params) {
		return
	}

	if reuse {
		if j := s.findRecent(params); j != nil {
			s.writeCreated(w, j, true)
			return
		}
	}

	if s.refuseStart(w) {
		return
	}

	title, ok := s.Titles[videoID]
	if !ok {
		title = "Video " + videoID
	}

	j := &job{
		status:   api.StatusQueued,
		platform: "youtube",
		url:      canonical,
		videoID:  videoID,
		title:    title,
		source:   body.Source,
		language: body.Language,
		diarize:  body.Diarize,
		idemKey:  key,
		params:   params,
	}
	s.addJob(j)
	s.charge(j)
	s.recordPeak()
	s.writeCreated(w, j, false)
}

func (s *Server) findRecent(params string) *job {
	for i := len(s.order) - 1; i >= 0; i-- {
		j := s.order[i]
		if j.params != params {
			continue
		}

		if slices.Contains([]string{api.StatusFailed, api.StatusCanceled}, j.status) {
			continue
		}

		if time.Since(j.createdAt) < 24*time.Hour {
			return j
		}
	}

	return nil
}
