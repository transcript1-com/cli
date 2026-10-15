package upload

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/fakeapi"
)

func holdTheOnlySlot(t *testing.T, srv *fakeapi.Server, c *api.Client) *api.Transcription {
	t.Helper()
	srv.ActiveLimit = 1
	held, err := c.CreateLink(context.Background(), "held", "https://youtu.be/aaaaaaaaaaa", api.CreateOptions{}, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	return held
}

func TestUploadAtTheJobLimitWithoutWaitSlot(t *testing.T) {
	srv, c, in, _ := setup(t, "talk.mp3")
	holdTheOnlySlot(t, srv, c)
	_, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"})

	if !api.IsCode(err, api.CodeTooManyActiveJobs) {
		t.Fatalf("got %v", err)
	}
}

func TestUploadWaitsForASlotAtComplete(t *testing.T) {
	srv, c, in, data := setup(t, "talk.mp3")
	held := holdTheOnlySlot(t, srv, c)
	var waits []*api.Error
	opt := Options{
		IdempotencyKey: "idem-1",
		WaitSlot: func(ctx context.Context, err *api.Error) error {
			waits = append(waits, err)
			_, getErr := c.Get(ctx, held.ID, 0)

			return getErr
		},
	}

	tr, err := Run(context.Background(), c, in, opt)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("calls WaitSlot with the 429", func(t *testing.T) {
		if len(waits) != 1 {
			t.Fatalf("waits %+v", waits)
		}

		if waits[0].Status != 429 {
			t.Fatalf("waits %+v", waits)
		}

		if waits[0].Code != api.CodeTooManyActiveJobs {
			t.Fatalf("waits %+v", waits)
		}
	})

	t.Run("never sends a chunk twice because of the wait", func(t *testing.T) {
		for index, statuses := range listChunkPuts(srv) {
			if !slices.Equal(statuses, []int{200}) {
				t.Fatalf("chunk %d answered %v", index, statuses)
			}
		}

		if n := countPrefix(srv.Log(), "POST /v1/transcriptions/"+tr.ID+"/upload/complete"); n < 2 {
			t.Fatalf("%d completes", n)
		}
	})

	t.Run("returns the queued transcription", func(t *testing.T) {
		if tr.Status != api.StatusQueued {
			t.Fatalf("status %q", tr.Status)
		}

		if !bytes.Equal(srv.Uploaded(tr.ID), data) {
			t.Fatalf("status %q", tr.Status)
		}
	})
}
