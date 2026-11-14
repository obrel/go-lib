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
		// 10:00 -> sepuluh (per spec)
		if m == 0 && (!hasSec || sec == 0) {
			return hW
		}
		if m == 0 && hasSec && sec != 0 {
			sW := numToWordsID(sec)
			return hW + " lewat " + sW + " detik"
		}
		mW := numToWordsID(m)
		if !hasSec || sec == 0 {
			// 10:30 -> sepuluh lewat tiga puluh menit
			return hW + " lewat " + mW + " menit"
		}
		// with seconds: 10:30:15 -> sepuluh lewat tiga puluh menit lima belas detik
		sW := numToWordsID(sec)
		return hW + " lewat " + mW + " menit " + sW + " detik"
	}
	// EN
	hW := numToWordsEN(h)
	if m == 0 && (!hasSec || sec == 0) {
		return hW
	}
	mW := numToWordsEN(m)
	if !hasSec || sec == 0 {
		// keep simple: 22:22 -> twenty two twenty two
		// but could also be "twenty two past twenty two" – keep space separated for backward compat
		return hW + " " + mW
	}
	sW := numToWordsEN(sec)
	return hW + " " + mW + " " + sW
}
