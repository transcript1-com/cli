package api

import (
	"context"
	"net/http"
	"testing"
)

func TestAccount(t *testing.T) {
	var got *http.Request
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		writeJSON(w, 200, `{"object":"account","user":{"id":"u1","email":"a@b.c"},"plan":{"id":"p1","name":"Pro"},`+
			`"minutes":{"remaining":42,"cycle":40,"topup":2,"topup_usable":true,"resets_at":null},"renews_at":null,`+
			`"limits":{"requests_per_minute":60,"max_active_jobs":4}}`)
	})

	acc, err := c.Account(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("reads the account endpoint", func(t *testing.T) {
		if got.Method != "GET" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.URL.Path != "/v1/account" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}
	})

	t.Run("parses user, plan, minutes and limits", func(t *testing.T) {
		if acc.User.Email != "a@b.c" {
			t.Fatalf("got %+v", acc)
		}

		if acc.Plan.Name != "Pro" {
			t.Fatalf("got %+v", acc)
		}

		if acc.Minutes.Remaining != 42 {
			t.Fatalf("got %+v", acc)
		}

		if acc.Limits.MaxActiveJobs != 4 {
			t.Fatalf("got %+v", acc)
		}
	})

	t.Run("leaves the plan nil when there is none", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, `{"user":{"email":"a@b.c"},"plan":null,"minutes":{"remaining":60},"limits":{"max_active_jobs":1}}`)
		})

		free, err := c.Account(context.Background())
		if err != nil {
			t.Fatalf("got %+v, %v", free, err)
		}

		if free.Plan != nil {
			t.Fatalf("got %+v, %v", free, err)
		}

		if free.Minutes.Remaining != 60 {
			t.Fatalf("got %+v, %v", free, err)
		}
	})
}
