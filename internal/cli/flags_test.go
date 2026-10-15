package cli

import (
	"slices"
	"testing"
)

func TestParseFlags(t *testing.T) {
	t.Run("collects inputs between flag runs", func(t *testing.T) {
		s, ins, err := parseFlags(false, []string{"a", "-f", "srt", "b", "--jobs=5", "-", "-q"})
		if err != nil {
			t.Fatal(err)
		}

		if !slices.Equal(ins, []string{"a", "b", "-"}) {
			t.Fatalf("settings %+v inputs %v", s, ins)
		}

		if s.format != "srt" {
			t.Fatalf("settings %+v inputs %v", s, ins)
		}

		if s.jobs != 5 {
			t.Fatalf("settings %+v inputs %v", s, ins)
		}

		if !s.quiet {
			t.Fatalf("settings %+v inputs %v", s, ins)
		}
	})

	t.Run("treats everything after -- as inputs", func(t *testing.T) {
		_, ins, err := parseFlags(false, []string{"-q", "--", "-intro.mp3", "-f"})
		if err != nil {
			t.Fatalf("inputs %v err %v", ins, err)
		}

		if !slices.Equal(ins, []string{"-intro.mp3", "-f"}) {
			t.Fatalf("inputs %v err %v", ins, err)
		}
	})

	t.Run("accepts the long and short names of a flag", func(t *testing.T) {
		s, _, err := parseFlags(false, []string{"--format", "vtt", "--out", "x", "--quiet"})
		if err != nil {
			t.Fatalf("settings %+v err %v", s, err)
		}

		if s.format != "vtt" {
			t.Fatalf("settings %+v err %v", s, err)
		}

		if s.out != "x" {
			t.Fatalf("settings %+v err %v", s, err)
		}

		if !s.quiet {
			t.Fatalf("settings %+v err %v", s, err)
		}
	})

	t.Run("defaults to text, two jobs and no source choice", func(t *testing.T) {
		s, _, err := parseFlags(false, nil)
		if err != nil {
			t.Fatalf("settings %+v err %v", s, err)
		}

		if s.format != "text" {
			t.Fatalf("settings %+v err %v", s, err)
		}

		if s.jobs != 2 {
			t.Fatalf("settings %+v err %v", s, err)
		}

		if s.source != "auto" {
			t.Fatalf("settings %+v err %v", s, err)
		}
	})

	t.Run("refuses transcribe-only flags for get", func(t *testing.T) {
		flags := []string{"--speakers", "--jobs=3", "--fresh", "--source=stt", "--language=en", "--overwrite"}
		for _, flag := range flags {
			if _, _, err := parseFlags(true, []string{flag, "tr_1"}); err == nil {
				t.Fatalf("get accepted %s", flag)
			}
		}
	})

	t.Run("names a flag that is missing its value", func(t *testing.T) {
		if _, _, err := parseFlags(false, []string{"-f"}); err == nil {
			t.Fatal("no error")
		}
	})
}

func TestValidateFlags(t *testing.T) {
	t.Run("accepts every format and source", func(t *testing.T) {
		for _, format := range []string{"text", "srt", "vtt", "json"} {
			if err := validateFlags(&settings{format: format, source: "auto", jobs: 2}); err != nil {
				t.Fatalf("%s: %v", format, err)
			}
		}

		for _, source := range []string{"auto", "captions", "stt"} {
			if err := validateFlags(&settings{format: "text", source: source, jobs: 2}); err != nil {
				t.Fatalf("%s: %v", source, err)
			}
		}
	})

	t.Run("accepts the job limits", func(t *testing.T) {
		for _, jobs := range []int{1, 20} {
			if err := validateFlags(&settings{format: "text", source: "auto", jobs: jobs}); err != nil {
				t.Fatalf("%d: %v", jobs, err)
			}
		}
	})
}
