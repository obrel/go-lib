package textnorm

import (
	"net/url"
	"regexp"
	"strings"
)

var reDigits = regexp.MustCompile(`\d+`)

func verbalizeURL(raw string, opts Options) string {
	// extract host, ignore protocol per spec
	// https://google.com -> google dot com
	parsed, err := url.Parse(raw)
	if err != nil {
		// fallback: strip protocol manually
		s := strings.TrimPrefix(raw, "https://")
		s = strings.TrimPrefix(s, "http://")
		// take host until / ? #
		if idx := strings.IndexAny(s, "/?#"); idx != -1 {
			s = s[:idx]
		}
		return verbalizeDomain(s, opts)
	}
	host := parsed.Host
	if host == "" {
		// maybe url without // like google.com/path handled elsewhere; treat raw as domain
		return verbalizeDomain(raw, opts)
	}
	// remove port
	if strings.Contains(host, ":") {
		if h, err := splitHostPort(host); err == nil {
			host = h
		} else {
			// fallback: strip after :
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
		}
	}
	// strip www.
	host = strings.TrimPrefix(host, "www.")

	// Only host per spec, not path
	return verbalizeDomain(host, opts)
}

func splitHostPort(host string) (string, error) {
	// simple split
	if strings.HasPrefix(host, "[") {
		// ipv6
		return host, nil
	}
	// find last colon
	idx := strings.LastIndex(host, ":")
	if idx == -1 {
		return host, nil
	}
	// check if after colon is numeric
	return host[:idx], nil
}

func verbalizeDomainLabel(label string, opts Options) string {
	// handle numbers inside label: example123 -> example satu dua tiga
	// replace digit runs with spaced spelled digits
	label = reDigits.ReplaceAllStringFunc(label, func(m string) string {
		// spell digits to avoid later number regex matching
		return " " + spellDigits(m, opts) + " "
	})
	// collapse multiple spaces
	label = strings.Join(strings.Fields(label), " ")
	return label
}

func verbalizeDomain(raw string, opts Options) string {
	// raw could be google.com or sub.domain.co.id
	// strip any path if present
	if idx := strings.IndexAny(raw, "/?#"); idx != -1 {
		raw = raw[:idx]
	}
	// strip www.
	raw = strings.TrimPrefix(raw, "www.")
	raw = strings.ToLower(raw) // keep lower for consistency

	dotWord := "dot"

	parts := strings.Split(raw, ".")
	var clean []string
	for _, p := range parts {
		if p != "" {
			// verbalize numbers inside each label
			clean = append(clean, verbalizeDomainLabel(p, opts))
		}
	}
	return strings.Join(clean, " "+dotWord+" ")
}
