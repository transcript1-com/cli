package api

import (
	"errors"
	"net/http"
)

func SlotBusy(err error) (*Error, bool) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return nil, false
	}

	if apiErr.Status != http.StatusTooManyRequests {
		return nil, false
	}

	return apiErr, IsCode(err, CodeTooManyActiveJobs, CodeRateLimited)
}
