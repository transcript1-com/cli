package api

import "testing"

func TestChunkURL(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "keeps an absolute template",
			base:     "https://api.transcript1.com",
			template: "https://api.transcript1.com/v1/transcriptions/tr_1/upload/chunks/{index}",
			want:     "https://api.transcript1.com/v1/transcriptions/tr_1/upload/chunks/3",
		},
		{
			name:     "resolves a path-only template against the origin of the base url",
			base:     "http://127.0.0.1:3000/mock-api",
			template: "/mock-api/v1/transcriptions/tr_1/upload/chunks/{index}",
			want:     "http://127.0.0.1:3000/mock-api/v1/transcriptions/tr_1/upload/chunks/3",
		},
		{
			name:     "rejects a template without a placeholder",
			base:     "http://x",
			template: "http://x/chunks",
			wantErr:  true,
		},
		{
			name:     "rejects an empty template",
			base:     "http://x",
			template: "",
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.base, "t1_x")
			got, err := c.ChunkURL(&UploadPlan{ChunkURLTemplate: tc.template}, 3)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %q", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("got %q, %v", got, err)
			}

			if got != tc.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}

	t.Run("rejects a missing plan", func(t *testing.T) {
		if _, err := New("http://x", "t1_x").ChunkURL(nil, 0); err == nil {
			t.Fatal("want an error")
		}
	})
}
