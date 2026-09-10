// Package unicodecase implements Unicode 16 default full upper/lower casing
// from the pinned Unicode data. It does not normalize or apply locale tailoring.
// Callers pass valid scalar strings; the evaluator's jstring boundary preserves
// isolated UTF-16 units separately as uncased, non-ignorable boundaries.
package unicodecase

import "strings"

func contains(cp rune, ranges [][2]rune) bool {
	low, high := 0, len(ranges)
	for low < high {
		middle := (low + high) / 2
		if cp < ranges[middle][0] {
			high = middle
		} else if cp > ranges[middle][1] {
			low = middle + 1
		} else {
			return true
		}
	}
	return false
}

func Upper(source string) string { return convert(source, false) }
func Lower(source string) string { return convert(source, true) }

// DigitZero returns the zero character of a Unicode 16 Nd decimal family,
// or zero when the character is not a decimal digit. No host Unicode table is used.
func DigitZero(cp rune) rune {
	for _, zero := range decimalZeros {
		if cp >= zero && cp <= zero+9 {
			return zero
		}
	}
	return 0
}

func convert(source string, lower bool) string {
	characters := []rune(source)
	mapping := upperMapping
	var following []bool
	if lower {
		mapping = lowerMapping
		following = make([]bool, len(characters))
		next := false
		for i := len(characters) - 1; i >= 0; i-- {
			following[i] = next
			if !contains(characters[i], ignorableRanges) {
				next = contains(characters[i], casedRanges)
			}
		}
	}
	var output strings.Builder
	output.Grow(len(source))
	previous := false
	for i, cp := range characters {
		if lower && cp == 0x3a3 && previous && !following[i] {
			output.WriteRune(0x3c2)
		} else if mapped, ok := mapping[cp]; ok {
			output.WriteString(mapped)
		} else {
			output.WriteRune(cp)
		}
		// Ignore first when properties overlap: Unicode 3.13 specifies a
		// possessive, non-backtracking Case_Ignorable match for Final_Sigma.
		if lower && !contains(cp, ignorableRanges) {
			previous = contains(cp, casedRanges)
		}
	}
	return output.String()
}
