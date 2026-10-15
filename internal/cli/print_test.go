package cli

import "testing"

func TestStdoutWriteErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "the transcript", args: []string{rickLink}},
		{name: "a json line", args: []string{"--json", rickLink}},
		{name: "a no-wait line", args: []string{"--no-wait", rickLink}},
	}
	for _, tc := range cases {
		t.Run("exits 1 when "+tc.name+" cannot be written", func(t *testing.T) {
			h := newHarness(t)
			h.stdout = failingWriter{}
			got := h.run(tc.args...)
			if got.code != 1 {
				t.Fatalf("code %d stderr %q", got.code, got.stderr)
			}

			expectContains(t, got.stderr, "transcript1: "+rickLink+": write /dev/stdout: broken pipe")
		})
	}

	t.Run("reports a failed write once", func(t *testing.T) {
		h := newHarness(t)
		h.stdout = failingWriter{}
		got := h.run("--json", "-o", "out", rickLink, otherURL)
		if got.code != 1 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		if n := countLines(got.stderr, "broken pipe"); n != 1 {
			t.Fatalf("%d error lines in %q", n, got.stderr)
		}
	})

	t.Run("keeps exit 2 for a rejected key", func(t *testing.T) {
		h := newHarness(t)
		h.stdout = failingWriter{}
		h.env["TRANSCRIPT1_API_KEY"] = "t1_wrong"
		got := h.run("--json", rickLink)
		if got.code != 2 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}
	})
}
