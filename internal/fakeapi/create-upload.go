package fakeapi

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

const msgUploadRequired = "upload.filename and a positive upload.size are required"

func (s *Server) createUpload(w http.ResponseWriter, key string, body createBody) {
	u := body.Upload
	if u.Filename == "" {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, msgUploadRequired, nil)
		return
	}

	if u.Size < 1 {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, msgUploadRequired, nil)
		return
	}

	if inputs.MimeType(u.Filename) == "" {
		writeError(w, http.StatusUnprocessableEntity, api.CodeUnsupportedMedia, "Unsupported file type", nil)
		return
	}

	if u.Size > inputs.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, api.CodeMediaTooLarge, "File exceeds the maximum size",
			map[string]any{"max_bytes": inputs.MaxUploadBytes})
		return
	}

	if !slices.Contains([]string{"auto", "stt"}, body.Source) {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "source must be auto or stt for uploads", nil)
		return
	}

	params := fmt.Sprintf("upload|%s|%d|%s|%s|%s|%t",
		u.Filename, u.Size, u.MimeType, body.Source, body.Language, body.Diarize)
	if s.replay(w, key, params) {
		return
	}

	if s.refuseMinutes(w) {
		return
	}

	j := &job{
		status:   api.StatusAwaitingUpload,
		platform: "upload",
		filename: u.Filename,
		size:     u.Size,
		source:   body.Source,
		language: body.Language,
		diarize:  body.Diarize,
		idemKey:  key,
		params:   params,
		chunks:   planChunks(u.Size, s.ChunkSize),
		received: map[int]bool{},
	}
	s.addJob(j)
	s.issueToken(j)
	s.writeCreated(w, j, false)
}
