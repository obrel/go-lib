package textnorm

import (
	"fmt"
	"strconv"
)

var monthID = map[int]string{
	1: "Januari", 2: "Februari", 3: "Maret", 4: "April",
	5: "Mei", 6: "Juni", 7: "Juli", 8: "Agustus",
	9: "September", 10: "Oktober", 11: "November", 12: "Desember",
}

var monthEN = map[int]string{
	1: "January", 2: "February", 3: "March", 4: "April",
	5: "May", 6: "June", 7: "July", 8: "August",
	9: "September", 10: "October", 11: "November", 12: "December",
}

func verbalizeDate(yearStr, monthStr, dayStr string, opts Options) string {
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	day, _ := strconv.Atoi(dayStr)

	if month < 1 || month > 12 || day < 1 || day > 31 {
		// fallback: return as is if invalid
		return fmt.Sprintf("%s-%s-%s", yearStr, monthStr, dayStr)
	}

	if opts.isID() {
		mName := monthID[month]
		dayWords := numToWordsID(day)
		yearWords := numToWordsID(year)
		// Indonesian common: "15 Januari 2024" -> words
		return fmt.Sprintf("%s %s %s", dayWords, mName, yearWords)
	}
	// EN: January 15 2024
	mName := monthEN[month]
	dayWords := numToWordsEN(day)
	yearWords := numToWordsEN(year)
	return fmt.Sprintf("%s %s %s", mName, dayWords, yearWords)
}
