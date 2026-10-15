package output

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/batch"
)

const codeError = "error"

func JSONLine(r batch.Result) ([]byte, error) {
	obj, err := buildJSONObject(r)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func hasJob(r batch.Result) bool {
	if r.T == nil {
		return false
	}

	return r.T.ID != ""
}

func needsErrorObject(r batch.Result, obj map[string]any) bool {
	if !isMissing(obj["error"]) {
		return false
	}

	if r.Err != nil {
		return true
	}

	return r.Status == batch.Interrupted
}

func buildJSONObject(r batch.Result) (map[string]any, error) {
	switch r.Status {
	case batch.Skipped:
		obj := map[string]any{"input": r.Input.Raw, "skipped": "exists"}
		if r.File != "" {
			obj["file"] = r.File
		}

		return obj, nil
	case batch.SkippedMinutes:
		return map[string]any{"input": r.Input.Raw, "skipped": "out_of_minutes"}, nil
	}

	if !hasJob(r) {
		return map[string]any{"input": r.Input.Raw, "error": buildErrorObject(r)}, nil
	}

	obj, err := buildAPIObject(r.T)
	if err != nil {
		return nil, err
	}

	obj["input"] = r.Input.Raw
	obj["reused"] = r.T.Reused
	if r.File != "" {
		obj["file"] = r.File
		delete(obj, "transcript")
	}

	if needsErrorObject(r, obj) {
		obj["error"] = buildErrorObject(r)
	}

	return obj, nil
}

func buildAPIObject(t *api.Transcription) (map[string]any, error) {
	raw := t.Raw
	if len(raw) == 0 {
		raw = json.RawMessage(`{"id":` + quote(t.ID) + `,"status":` + quote(t.Status) + `}`)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}

	obj := make(map[string]any, len(fields)+3)
	for key, value := range fields {
		obj[key] = value
	}

	return obj, nil
}

func isMissing(value any) bool {
	raw, ok := value.(json.RawMessage)
	if !ok {
		return true
	}

	return string(raw) == "null"
}

func quote(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func buildErrorObject(r batch.Result) map[string]string {
	if r.Err == nil {
		if r.Status == batch.Failed {
			return map[string]string{"code": codeError, "message": "failed"}
		}

		return map[string]string{"code": "interrupted", "message": "interrupted before the transcription started"}
	}

	var apiErr *api.Error
	if !errors.As(r.Err, &apiErr) {
		return map[string]string{"code": codeError, "message": r.Err.Error()}
	}

	message := apiErr.Message
	if message == "" {
		message = apiErr.Code
	}

	return map[string]string{"code": apiErr.Code, "message": message}
}
