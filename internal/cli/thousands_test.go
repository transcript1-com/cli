package cli

import "testing"

func TestThousands(t *testing.T) {
	cases := map[int]string{0: "0", 999: "999", 1000: "1,000", 12345: "12,345", 1234567: "1,234,567", -1500: "-1,500"}
	for n, want := range cases {
		t.Run("writes "+want, func(t *testing.T) {
			if got := formatThousands(n); got != want {
				t.Fatalf("got %q", got)
			}
		})
	}
}
