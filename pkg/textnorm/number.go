package textnorm

import (
	"regexp"
	"strconv"
	"strings"
)

type numberMode string

const (
	modeAuto     numberMode = "auto"
	modeCurrency numberMode = "currency"
	modeDigit    numberMode = "digit"
	modeNone     numberMode = "none"
)

var (
	reCurrency = regexp.MustCompile(`\d{1,3}(?:\.\d{3})+(?:,\d+)?`)
)

var digitMapID = map[rune]string{
	'0': "nol", '1': "satu", '2': "dua", '3': "tiga", '4': "empat",
	'5': "lima", '6': "enam", '7': "tujuh", '8': "delapan", '9': "sembilan",
}

var digitMapEN = map[rune]string{
	'0': "zero", '1': "one", '2': "two", '3': "three", '4': "four",
	'5': "five", '6': "six", '7': "seven", '8': "eight", '9': "nine",
}

func parseNumberMode(s string) numberMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "currency":
		return modeCurrency
	case "digit":
		return modeDigit
	case "none", "off", "false":
		return modeNone
	case "", "auto":
		return modeAuto
	default:
		return modeAuto
	}
}

func normalizeNumbers(s string, opts Options) string {
	mode := opts.numMode()
	if mode == modeNone {
		return s
	}
	return reNum.ReplaceAllStringFunc(s, func(m string) string {
		clean := strings.ReplaceAll(m, "-", "")
		clean = strings.ReplaceAll(clean, " ", "")

		hasDot := strings.Contains(clean, ".")
		hasComma := strings.Contains(clean, ",")

		if mode == modeDigit {
			digitsOnly := strings.ReplaceAll(strings.ReplaceAll(clean, ".", ""), ",", "")
			return spellDigits(digitsOnly, opts)
		}
		if mode == modeCurrency {
			return normalizeCurrency(clean, opts)
		}

		// auto
		if hasDot {
			if reCurrency.MatchString(clean) {
				return normalizeCurrency(clean, opts)
			}
			return spellDigits(strings.ReplaceAll(clean, ".", ""), opts)
		}
		if hasComma {
			parts := strings.Split(clean, ",")
			if len(parts) == 2 {
				leftStr := parts[0]
				rightStr := parts[1]
				var leftWords string
				if strings.HasPrefix(leftStr, "0") || len(leftStr) > 4 {
					leftWords = spellDigits(leftStr, opts)
				} else {
					leftNum, _ := strconv.Atoi(leftStr)
					leftWords = numToWords(leftNum, opts)
				}
				commaWord := "koma"
				if !opts.isID() {
					commaWord = "point"
				}
				return leftWords + " " + commaWord + " " + spellDigits(rightStr, opts)
			}
		}
		if strings.HasPrefix(clean, "0") && len(clean) > 1 {
			return spellDigits(clean, opts)
		}
		if len(clean) > 4 {
			return spellDigits(clean, opts)
		}
		n, _ := strconv.Atoi(clean)
		return numToWords(n, opts)
	})
}

func normalizeCurrency(s string, opts Options) string {
	parts := strings.Split(s, ",")
	intPartStr := strings.ReplaceAll(parts[0], ".", "")
	n, err := strconv.Atoi(intPartStr)
	if err != nil {
		return spellDigits(intPartStr, opts)
	}
	words := numToWords(n, opts)
	if len(parts) == 2 {
		commaWord := "koma"
		if !opts.isID() {
			commaWord = "point"
		}
		words += " " + commaWord + " " + spellDigits(parts[1], opts)
	}
	return words
}

func spellDigits(s string, opts Options) string {
	var out []string
	m := digitMapID
	if !opts.isID() {
		m = digitMapEN
	}
	for _, r := range s {
		if w, ok := m[r]; ok {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func numToWords(n int, opts Options) string {
	if opts.isID() {
		return numToWordsID(n)
	}
	return numToWordsEN(n)
}
