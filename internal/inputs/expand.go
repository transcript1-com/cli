package inputs

import "io"

const stdinArg = "-"

func Expand(args []string, stdin io.Reader) ([]Input, error) {
	var raws []string
	for _, arg := range args {
		if arg != stdinArg {
			raws = append(raws, arg)
			continue
		}

		lines, err := readList(stdin)
		if err != nil {
			return nil, err
		}

		raws = append(raws, lines...)
	}

	var out []Input
	seen := map[string]bool{}
	for _, raw := range raws {
		found, err := classify(raw)
		if err != nil {
			return nil, err
		}

		for _, in := range found {
			key := in.URL + in.Path
			if seen[key] {
				continue
			}

			seen[key] = true
			out = append(out, in)
		}
	}

	return out, nil
}
