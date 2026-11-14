package textnorm

import (
	"strconv"
)

func verbalizeTime(hourStr, minStr, secStr string, opts Options) string {
	h, _ := strconv.Atoi(hourStr)
	m, _ := strconv.Atoi(minStr)
	hasSec := secStr != ""
	sec := 0
	if hasSec {
		sec, _ = strconv.Atoi(secStr)
	}

	if opts.isID() {
		hW := numToWordsID(h)
		mW := numToWordsID(m)
		if hasSec {
			sW := numToWordsID(sec)
			// e.g. 22:22:30 -> dua puluh dua dua puluh dua tiga puluh
			return hW + " " + mW + " " + sW
		}
		// 22:22 -> dua puluh dua dua puluh dua, or with jam prefix? keep simple
		return hW + " " + mW
	}
	// EN
	hW := numToWordsEN(h)
	mW := numToWordsEN(m)
	if hasSec {
		sW := numToWordsEN(sec)
		return hW + " " + mW + " " + sW
	}
	return hW + " " + mW
}
