package files

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/transcript1-com/cli/internal/inputs"
)

const errSameFileTemplate = "%s and %s would write the same file; run them separately"

func FullNames(ins []inputs.Input) (map[int]bool, error) {
	byStem := map[string][]int{}
	byName := map[string]int{}
	for i, in := range ins {
		if in.Kind != inputs.File {
			continue
		}

		if j, ok := byName[in.Name]; ok {
			return nil, fmt.Errorf(errSameFileTemplate, getLabel(ins[j]), getLabel(in))
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

	return full, checkSameOutput(ins, full)
}

// macos and windows file systems ignore case
func checkSameOutput(ins []inputs.Input, full map[int]bool) error {
	seen := map[string]int{}
	for i, in := range ins {
		if in.Kind != inputs.File {
			continue
		}

		name := strings.ToLower(Name(in, "", FormatText, full[i]))
		if j, ok := seen[name]; ok {
			return fmt.Errorf(errSameFileTemplate, getLabel(ins[j]), getLabel(in))
		}

		seen[name] = i
	}

	return nil
}

func getLabel(in inputs.Input) string {
	if in.Raw != "" {
		return in.Raw
	}

	return in.Path
}
