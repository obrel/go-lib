package textnorm

import "strings"

func verbalizeEmail(s string, opts Options) string {
	atIdx := strings.LastIndex(s, "@")
	if atIdx == -1 {
		return s
	}
	local := s[:atIdx]
	domain := s[atIdx+1:]

	local = verbalizeEmailLocal(local, opts)
	domain = verbalizeDomainPart(domain, opts)

	atWord := "at"
	// same for id/en, keep "at" per spec; could be "et" for id but user wants "at"
	return local + " " + atWord + " " + domain
}

func verbalizeEmailLocal(local string, opts Options) string {
	// handle numbers inside local part and dots
	// split by dot, then for each part handle numbers
	var out []string
	for _, part := range strings.Split(local, ".") {
		if part == "" {
			continue
		}
		// handle numbers inside part like user123 -> user satu dua tiga
		part = reDigits.ReplaceAllStringFunc(part, func(m string) string {
			return " " + spellDigits(m, opts) + " "
		})
		part = strings.Join(strings.Fields(part), " ")
		if part != "" {
			out = append(out, part)
		}
	}
	sep := " dot "
	return strings.Join(out, sep)
}

func verbalizeDomainPart(domain string, opts Options) string {
	// domain like gmail.com -> gmail dot com, with numbers handled
	parts := strings.Split(domain, ".")
	var clean []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		clean = append(clean, verbalizeDomainLabel(p, opts))
	}
	dotWord := "dot"
	return strings.Join(clean, " "+dotWord+" ")
}
