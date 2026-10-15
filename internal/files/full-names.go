package files

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/transcript1/cli/internal/inputs"
)

func FullNames(ins []inputs.Input) (map[int]bool, error) {
	byStem := map[string][]int{}
	byName := map[string]int{}
	for i, in := range ins {
		if in.Kind != inputs.File {
			continue
		}

		if j, ok := byName[in.Name]; ok {
			return nil, fmt.Errorf("%s and %s would write the same file; run them separately", getLabel(ins[j]), getLabel(in))
		}

		byName[in.Name] = i
		stem := strings.TrimSuffix(in.Name, filepath.Ext(in.Name))
		byStem[stem] = append(byStem[stem], i)
	}

	full := map[int]bool{}
	for _, group := range byStem {
		if len(group) < 2 {
			continue
		}

		for _, i := range group {
			full[i] = true
		}
	}

	return full, nil
}

func getLabel(in inputs.Input) string {
	if in.Raw != "" {
		return in.Raw
	}

	return in.Path
}
