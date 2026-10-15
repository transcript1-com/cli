package update

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"1.0.1", "1.0.0", true},
		{"1.10.0", "1.9.9", true},
		{"2.0.0", "1.99.99", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"1.2.0", "dev", false},
		{"dev", "1.0.0", false},
		{"1.2", "1.0.0", false},
		{"1.2.0-rc1", "1.0.0", false},
		{"1.2.x", "1.0.0", false},
		{"+1.2.0", "1.0.0", false},
	}
	for _, c := range cases {
		t.Run(c.latest+" over "+c.current, func(t *testing.T) {
			if got := Newer(c.latest, c.current); got != c.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}
