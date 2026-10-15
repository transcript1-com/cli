package api

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

func TestTransient(t *testing.T) {
	t.Run("counts network errors and 429 or 5xx answers", func(t *testing.T) {
		for _, err := range []error{
			&url.Error{Op: "Get", URL: "x", Err: errors.New("EOF")},
			&Error{Status: 429},
			&Error{Status: 500},
			&Error{Status: 502},
			&Error{Status: 503},
			&Error{Status: 504},
		} {
			if !Transient(err) {
				t.Fatalf("%v is not transient", err)
			}
		}
	})

	t.Run("does not count other answers, a failed job or a canceled run", func(t *testing.T) {
		for _, err := range []error{
			&Error{Status: 401},
			&Error{Status: 404},
			&Error{Status: 422},
			&Error{Code: CodeMediaPrivate},
			context.Canceled,
			errors.New("unreadable"),
		} {
			if Transient(err) {
				t.Fatalf("%v is transient", err)
			}
		}
	})
}
