package textnorm

import (
	"regexp"
	"strings"
)

type Options struct {
	Language   string // only "id" supported, other langs passthrough as-is, default "id"
	NumberMode string // auto|currency|digit|none, default auto
}

func (o Options) lang() string {
	l := strings.ToLower(strings.TrimSpace(o.Language))
	if l == "" {
		return "id"
	}
	return l
}

func (o Options) isID() bool {
	return strings.HasPrefix(o.lang(), "id")
}

func (o Options) numMode() numberMode {
	return parseNumberMode(o.NumberMode)
}

var (
	// order matters: email -> url -> domain -> date -> time -> number -> abbr
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

	// URL with http(s)
	reURL = regexp.MustCompile(`https?://[^\s]+`)

	// bare domain like google.com, sub.domain.co.id, without protocol
	// avoid matching email (already replaced) and avoid matching inside verbalized text
	reDomain = regexp.MustCompile(`\b(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}(?:\.[a-zA-Z]{2,})*\b`)

	// date YYYY-MM-DD or YYYY/MM/DD
	reDate = regexp.MustCompile(`\b(\d{4})[-/](\d{2})[-/](\d{2})\b`)

	// time HH:MM or HH:MM:SS, 00-23 hours, optional seconds
	reTime = regexp.MustCompile(`\b([01]?\d|2[0-3]):([0-5]\d)(?::([0-5]\d))?\b`)

	// number detection (from existing tts logic, includes currency, decimal comma, dash)
	reNum = regexp.MustCompile(`\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d+,\d+|\d[\d\-\.]*\d|\d`)

	// abbreviation: all caps 2+ length, word boundaries
	reAbbr = regexp.MustCompile(`\b[A-Z]{2,}\b`)
)

func Normalize(text string, opts Options) string {
	if text == "" {
		return ""
	}
	// Only Indonesian is supported for now; other languages passthrough as-is.
	if !opts.isID() {
		return text
	}
	// email first
	text = reEmail.ReplaceAllStringFunc(text, func(m string) string {
		return verbalizeEmail(m, opts)
	})
	// URL next (host only per spec: google.com from https://google.com)
	text = reURL.ReplaceAllStringFunc(text, func(m string) string {
		return verbalizeURL(m, opts)
	})
	// bare domain (after email and url)
	text = reDomain.ReplaceAllStringFunc(text, func(m string) string {
		return verbalizeDomain(m, opts)
	})
	// date
	text = reDate.ReplaceAllStringFunc(text, func(m string) string {
		parts := reDate.FindStringSubmatch(m)
		if len(parts) != 4 {
			return m
		}
		return verbalizeDate(parts[1], parts[2], parts[3], opts)
	})
	// time
	text = reTime.ReplaceAllStringFunc(text, func(m string) string {
		parts := reTime.FindStringSubmatch(m)
		if len(parts) < 3 {
			return m
		}
		sec := ""
		if len(parts) >= 4 {
			sec = parts[3]
		}
		return verbalizeTime(parts[1], parts[2], sec, opts)
	})
	// number (mode aware)
	text = normalizeNumbers(text, opts)

	// abbreviation last
	text = reAbbr.ReplaceAllStringFunc(text, func(m string) string {
		return verbalizeAbbr(m, opts)
	})

	return text
}
