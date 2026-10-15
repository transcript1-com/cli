package output

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type liveLine struct {
	index int
	text  string
}

func (r *Reporter) setLive(index int, text string) {
	for i := range r.live {
		if r.live[i].index != index {
			continue
		}

		if r.live[i].text == text {
			return
		}

		r.live[i].text = text
		r.redraw("")

		return
	}

	r.live = append(r.live, liveLine{index: index, text: text})
	slices.SortFunc(r.live, func(a, b liveLine) int { return cmp.Compare(a.index, b.index) })
	r.redraw("")
}

func (r *Reporter) removeLive(index int, final string) {
	r.live = slices.DeleteFunc(r.live, func(l liveLine) bool { return l.index == index })
	r.redraw(final)
}

func (r *Reporter) redraw(final string) {
	var b strings.Builder
	if r.drawn > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", r.drawn)
	}

	if final != "" {
		b.WriteString("\r\x1b[K" + final + "\n")
	}

	for _, l := range r.live {
		b.WriteString("\r\x1b[K" + l.text + "\n")
	}

	b.WriteString("\x1b[J")
	r.drawn = len(r.live)
	fmt.Fprint(r.w, b.String())
}
