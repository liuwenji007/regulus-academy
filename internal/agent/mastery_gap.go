package agent

import "strings"

func mergeGapConcepts(parts ...[]string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, slice := range parts {
		for _, s := range slice {
			add(s)
		}
	}
	return out
}
