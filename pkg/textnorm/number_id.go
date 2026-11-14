package textnorm

import "strings"

func numToWordsID(n int) string {
	if n == 0 {
		return "nol"
	}
	if n < 0 {
		return "minus " + numToWordsID(-n)
	}
	var parts []string
	if n >= 1_000_000_000_000 {
		t := n / 1_000_000_000_000
		parts = append(parts, numToWordsID(t)+" triliun")
		n %= 1_000_000_000_000
	}
	if n >= 1_000_000_000 {
		t := n / 1_000_000_000
		parts = append(parts, numToWordsID(t)+" miliar")
		n %= 1_000_000_000
	}
	if n >= 1_000_000 {
		t := n / 1_000_000
		if t == 1 {
			parts = append(parts, "satu juta")
		} else {
			parts = append(parts, numToWordsID(t)+" juta")
		}
		n %= 1_000_000
	}
	if n >= 1000 {
		if n < 2000 {
			parts = append(parts, "seribu")
			n -= 1000
		} else {
			t := n / 1000
			parts = append(parts, numToWordsID(t)+" ribu")
			n %= 1000
		}
	}
	if n >= 100 {
		if n < 200 {
			parts = append(parts, "seratus")
			n -= 100
		} else {
			t := n / 100
			parts = append(parts, numToWordsID(t)+" ratus")
			n %= 100
		}
	}
	if n >= 20 {
		t := n / 10
		parts = append(parts, numToWordsID(t)+" puluh")
		n %= 10
		if n > 0 {
			parts = append(parts, numToWordsID(n))
		}
		return strings.Join(parts, " ")
	}
	if n == 11 {
		return strings.Join(append(parts, "sebelas"), " ")
	}
	if n >= 12 && n <= 19 {
		return strings.Join(append(parts, numToWordsID(n%10)+" belas"), " ")
	}
	if n == 10 {
		return strings.Join(append(parts, "sepuluh"), " ")
	}
	if n > 0 && n < 10 {
		return strings.Join(append(parts, digitMapID[rune('0'+n)]), " ")
	}
	return strings.Join(parts, " ")
}
