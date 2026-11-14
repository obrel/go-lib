package textnorm

import "strings"

var enUnits = []string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
var enTens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

func numToWordsEN(n int) string {
	if n == 0 {
		return "zero"
	}
	if n < 0 {
		return "minus " + numToWordsEN(-n)
	}
	var parts []string
	if n >= 1_000_000_000_000 {
		t := n / 1_000_000_000_000
		parts = append(parts, numToWordsEN(t)+" trillion")
		n %= 1_000_000_000_000
	}
	if n >= 1_000_000_000 {
		t := n / 1_000_000_000
		parts = append(parts, numToWordsEN(t)+" billion")
		n %= 1_000_000_000
	}
	if n >= 1_000_000 {
		t := n / 1_000_000
		parts = append(parts, numToWordsEN(t)+" million")
		n %= 1_000_000
	}
	if n >= 1000 {
		t := n / 1000
		parts = append(parts, numToWordsEN(t)+" thousand")
		n %= 1000
	}
	if n >= 100 {
		t := n / 100
		parts = append(parts, enUnits[t]+" hundred")
		n %= 100
	}
	if n >= 20 {
		t := n / 10
		parts = append(parts, enTens[t])
		n %= 10
		if n > 0 {
			parts = append(parts, enUnits[n])
		}
		return strings.Join(parts, " ")
	}
	if n > 0 {
		parts = append(parts, enUnits[n])
	}
	return strings.Join(parts, " ")
}
