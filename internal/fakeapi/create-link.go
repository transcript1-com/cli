package fakeapi

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/inputs"
)

func needsSpeechToText(body createBody) bool {
	if !body.Diarize {
		return false
	}

	return body.Source == "captions"
}

func (s *Server) createLink(w http.ResponseWriter, key string, reuse bool, body createBody) {
	platform, ok := s.checkPlatform(w, body)
	if !ok {
		return
	}

	canonical, videoID, ok := readLink(platform, body.URL)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidURL, msgUnsupportedLink, nil)
		return
	}

	if !slices.Contains([]string{"auto", "captions", "stt"}, body.Source) {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "source must be auto, captions or stt", nil)
		return
	}

	if body.Source == "captions" && platform != inputs.PlatformYouTube {
		message := platform + " does not support source captions"
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, message, nil)
		return
	}

	if needsSpeechToText(body) {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "diarize needs speech-to-text", nil)
		return
	}

	params := fmt.Sprintf("link|%s|%s|%s|%t", canonical, body.Source, body.Language, body.Diarize)
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
		title = strings.TrimSpace("Video " + videoID)
	}

	j := &job{
		status:   api.StatusQueued,
		platform: platform,
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
