package textnorm

import "strings"

func verbalizeAbbr(s string, opts Options) string {
	// spell one by one: WIB -> W I B
	// keep original case letters spaced
	var parts []string
	for _, r := range s {
		parts = append(parts, string(r))
	}
	return strings.Join(parts, " ")
}
