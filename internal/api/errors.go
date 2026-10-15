package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Error struct {
	Status     int
	Code       string
	Message    string
	Retryable  bool
	RetryAfter time.Duration
	Details    map[string]any
}

func (e *Error) Error() string {
	message := strings.TrimRight(e.Message, ".")
	if message == "" {
		return e.Code
	}

	return e.Code + ": " + message
}

func IsCode(err error, codes ...string) bool {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return false
	}

	return slices.Contains(codes, apiErr.Code)
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}

	return false
}

func parseRetryAfter(h http.Header) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(h.Get("Retry-After")))
	if err != nil {
		return 0
	}

	if seconds < 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

func buildResponseError(res *response) *Error {
	apiErr := parseEnvelope(res.Body)
	if apiErr == nil {
		apiErr = &Error{
			Code:      fmt.Sprintf("http_%d", res.Status),
			Message:   fmt.Sprintf("the server answered HTTP %d", res.Status),
			Retryable: isRetryableStatus(res.Status),
		}
	}

	apiErr.Status = res.Status
	apiErr.RetryAfter = parseRetryAfter(res.Header)

	return apiErr
}

func parseEnvelope(body []byte) *Error {
	var env struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil
	}

	if env.Error == nil {
		return nil
	}

	code, _ := env.Error["code"].(string)
	if code == "" {
		return nil
	}

	message, _ := env.Error["message"].(string)
	retryable, _ := env.Error["retryable"].(bool)
	for _, key := range []string{"code", "message", "retryable"} {
		delete(env.Error, key)
	}

	apiErr := &Error{
		Code:      code,
		Message:   message,
		Retryable: retryable,
	}
	if len(env.Error) > 0 {
		apiErr.Details = env.Error
	}

	return apiErr
}
