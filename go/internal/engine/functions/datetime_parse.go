package functions

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
)

var validComponents = map[rune]bool{
	'Y': true, 'M': true, 'D': true, 'd': true, 'H': true, 'h': true,
	'm': true, 's': true, 'f': true, 'F': true, 'Z': true, 'z': true,
	'P': true, 'C': true, 'E': true, 'W': true, 'w': true, 'X': true, 'x': true,
}

// Extract picture tokens in order.
type picturePart struct {
	isToken   bool
	token     string // if isToken
	component rune
	modifier  string
	literal   string // if !isToken
}

func parseWithPicture(input, picture string, now time.Time, standardOnly bool) (time.Time, bool, error) { //nolint:gocyclo,funlen // dispatch
	parts, pictureErr := parseDatePicture(picture, standardOnly)
	if pictureErr != nil {
		return time.Time{}, false, pictureErr
	}

	// Track which components appear in the picture for validation.
	hasToken := false
	for _, p := range parts {
		if p.isToken {
			hasToken = true
		}
	}
	if !hasToken {
		return time.Time{}, false, nil
	}
	present := make(map[rune]bool)
	for _, p := range parts {
		if !p.isToken {
			continue
		}
		present[p.component] = true
	}
	// Week-based components without full calendar date specification.
	if present['X'] && !present['Y'] {
		return time.Time{}, false, &evaluator.JSONataError{
			Code:    "D3136",
			Message: "$toMillis: the date/time picture is underspecified; week-based year requires full calendar date",
		}
	}

	// Parse the input using the parts.
	var year, month, day, hour, minute, second, millisec, dayOfYear, tzOffset, pos, week, monthWeek, weekday int
	var isPM, is12h, hasTZ, negativeEra bool
	inputRunes := []rune(input)
	supplied := make(map[rune]int)

	for _, part := range parts {
		if !part.isToken {
			// Consume literal from input.
			for _, lr := range part.literal {
				if pos >= len(inputRunes) {
					return time.Time{}, false, nil
				}
				if unicode.ToLower(inputRunes[pos]) != unicode.ToLower(lr) {
					return time.Time{}, false, nil
				}
				pos++
			}
			continue
		}

		component := part.component
		modifier := part.modifier

		switch component {
		case 'C':
			if pos+3 > len(inputRunes) || !strings.EqualFold(string(inputRunes[pos:pos+3]), "ISO") {
				return time.Time{}, false, nil
			}
			pos += 3
		case 'E':
			negativeEra = false
			if pos < len(inputRunes) && inputRunes[pos] == '-' {
				negativeEra = true
				pos++
			}
		case 'Y', 'X', 'M', 'D', 'd', 'H', 'h', 'm', 's', 'W', 'w':
			v, n := parseTokenValue(inputRunes[pos:], modifier)
			if n == calendarFieldOverflow {
				return time.Time{}, false, invalidPictureTimestamp()
			}
			if n < 0 {
				return time.Time{}, false, nil
			}
			pos += n
			switch component {
			case 'Y', 'X':
				year = v
			case 'M':
				month = v
			case 'D':
				day = v
			case 'd':
				dayOfYear = v
			case 'H':
				hour = v
			case 'h':
				is12h = true
				hour = v
			case 'm':
				minute = v
			case 's':
				second = v
			case 'W':
				week = v
			case 'w':
				monthWeek = v
			}
		case 'f': // Fractional seconds
			digits, n := parsePictureDigits(inputRunes[pos:], modifier)
			if n < 0 {
				return time.Time{}, false, nil
			}
			// Digits, not grouping characters, determine decimal places.
			millisec, _ = strconv.Atoi((digits + "000")[:3])
			pos += n
		case 'P': // AM/PM
			if pos >= len(inputRunes) {
				return time.Time{}, false, nil
			}
			if pos+2 <= len(inputRunes) {
				part2 := strings.ToLower(string(inputRunes[pos : pos+2]))
				switch part2 {
				case "am":
					isPM = false
					pos += 2
				case "pm":
					isPM = true
					pos += 2
				default:
					return time.Time{}, false, nil
				}
			} else {
				return time.Time{}, false, nil
			}
		case 'F':
			v, n := parseWeekday(inputRunes[pos:], modifier)
			if n == calendarFieldOverflow {
				return time.Time{}, false, invalidPictureTimestamp()
			}
			if n < 0 {
				return time.Time{}, false, nil
			}
			weekday = v
			pos += n
		case 'Z', 'z': // Timezone - parse and apply
			offset, n := parseTZFromInput(inputRunes[pos:], modifier, component)
			if n < 0 {
				return time.Time{}, false, invalidPictureTimestamp()
			}
			if n == 0 {
				return time.Time{}, false, nil
			}
			if n > 0 {
				tzOffset = offset
				hasTZ = true
				pos += n
			}
		case 'x': // Raw compatibility extension; not exposed in the closed path.
			n := consumeNameOrNumber(inputRunes[pos:], modifier)
			if n > 0 {
				pos += n
			}
		}
		var value int
		switch component {
		case 'Y', 'X':
			value = year
		case 'M':
			value = month
		case 'D':
			value = day
		case 'd':
			value = dayOfYear
		case 'H', 'h':
			value = hour
		case 'm':
			value = minute
		case 's':
			value = second
		case 'f':
			value = millisec
		case 'P':
			if isPM {
				value = 1
			}
		case 'E':
			if negativeEra {
				value = -1
			}
		case 'F':
			value = weekday
		case 'W':
			value = week
		case 'w':
			value = monthWeek
		case 'Z', 'z':
			value = tzOffset
		}
		if previous, seen := supplied[component]; seen && previous != value {
			return time.Time{}, false, invalidPictureTimestamp()
		}
		supplied[component] = value
	}

	if pos != len(inputRunes) {
		return time.Time{}, false, nil
	}

	// Preserve the established partial-picture convention: leading omissions
	// use this evaluation's frozen clock, trailing omissions use minimum values,
	// and an internal gap is ambiguous. Presence is independent of numeric zero.
	dateFields, timeFields := "YMD", "Hmsf"
	weekDate := !present['D'] && !present['d'] && (present['W'] || present['w'] || present['F']) && !present['X'] && !present['x']
	if weekDate {
		dateFields = "Y"
		if present['M'] || present['w'] {
			dateFields += "M"
		}
		if present['w'] {
			dateFields += "w"
		} else {
			dateFields += "W"
		}
		dateFields += "F"
	}
	if present['d'] && !present['M'] && !present['D'] {
		dateFields = "Yd"
	}
	if !present['H'] && (is12h || present['P']) {
		timeFields = "Phmsf"
		is12h = true
	} else {
		is12h = false
		if present['H'] {
			hour = supplied['H']
		}
	}
	halfDay := 0
	if isPM {
		halfDay = 1
	}
	values := map[rune]*int{'Y': &year, 'M': &month, 'D': &day, 'd': &dayOfYear, 'H': &hour, 'h': &hour, 'P': &halfDay, 'm': &minute, 's': &second, 'f': &millisec}
	current := map[rune]int{'Y': now.Year(), 'M': int(now.Month()), 'D': now.Day(), 'd': now.YearDay(), 'H': now.Hour(), 'h': (now.Hour()+11)%12 + 1, 'P': now.Hour() / 12, 'm': now.Minute(), 's': now.Second(), 'f': now.Nanosecond() / 1e6}
	values['W'], values['w'], values['F'] = &week, &monthWeek, &weekday
	_, current['W'] = now.ISOWeek()
	current['w'] = weekOfMonth(isoWeekThursday(now))
	current['F'] = (int(now.Weekday())+6)%7 + 1
	started, ended := false, false
	for _, field := range dateFields + timeFields {
		if present[field] {
			if ended {
				return time.Time{}, false, &evaluator.JSONataError{Code: "D3136", Message: "$toMillis: internal gap in date/time picture"}
			}
			started = true
		} else if !started {
			*values[field] = current[field]
		} else {
			*values[field] = 0
			if strings.ContainsRune("MDdWwF", field) {
				*values[field] = 1
			}
			ended = true
		}
	}

	// Adjust for 12-hour clock.
	if is12h {
		if hour < 1 || hour > 12 {
			return time.Time{}, false, invalidPictureTimestamp()
		}
		isPM = halfDay == 1
		if isPM && hour != 12 {
			hour += 12
		} else if !isPM && hour == 12 {
			hour = 0
		}
	}

	if negativeEra {
		year = -year
	}
	if weekDate {
		if year < -271821 || year > 275760 {
			return time.Time{}, false, invalidPictureTimestamp()
		}
		var selected time.Time
		found := false
		for ordinal := 1; ordinal <= 366; ordinal++ {
			date := time.Date(year, 1, ordinal, 0, 0, 0, 0, time.UTC)
			if date.Year() != year {
				break
			}
			_, dateWeek := date.ISOWeek()
			if ((present['M'] || present['w']) && int(date.Month()) != month) ||
				((present['W'] || !present['w']) && dateWeek != week) ||
				(present['w'] && weekOfMonth(isoWeekThursday(date)) != monthWeek) || (int(date.Weekday())+6)%7+1 != weekday {
				continue
			}
			if found {
				return time.Time{}, false, &evaluator.JSONataError{Code: "D3136", Message: "ambiguous week date in calendar year"}
			}
			selected = date
			found = true
		}
		if !found {
			return time.Time{}, false, invalidPictureTimestamp()
		}
		month, day = int(selected.Month()), selected.Day()
	}
	if year < -271821 || year > 275760 || hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return time.Time{}, false, invalidPictureTimestamp()
	}

	if dateFields == "Yd" {
		if dayOfYear < 1 || dayOfYear > time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC).YearDay() {
			return time.Time{}, false, invalidPictureTimestamp()
		}
		derived := time.Date(year, 1, dayOfYear, 0, 0, 0, 0, time.UTC)
		month, day = int(derived.Month()), derived.Day()
	}
	if month < 1 || month > 12 || day < 1 || day > time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day() {
		return time.Time{}, false, invalidPictureTimestamp()
	}

	t := time.Date(year, time.Month(month), day, hour, minute, second, millisec*1e6, time.UTC)
	if (present['h'] && (hour+11)%12+1 != supplied['h']) || (present['P'] && hour/12 != supplied['P']) ||
		(present['Z'] && present['z'] && supplied['Z'] != supplied['z']) {
		return time.Time{}, false, invalidPictureTimestamp()
	}
	_, actualWeek := t.ISOWeek()
	if (present['W'] && actualWeek != week) || (present['w'] && weekOfMonth(isoWeekThursday(t)) != monthWeek) ||
		(present['F'] && (int(t.Weekday())+6)%7+1 != weekday) || (present['d'] && t.YearDay() != dayOfYear) {
		return time.Time{}, false, invalidPictureTimestamp()
	}
	if hasTZ {
		// tzOffset is seconds from UTC, negative = west (e.g. +02:00 = +7200s from UTC, so we subtract it)
		t = t.Add(-time.Duration(tzOffset) * time.Second)
	}
	if t.UnixMilli() < -8640000000000000 || t.UnixMilli() > 8640000000000000 {
		return time.Time{}, false, invalidPictureTimestamp()
	}
	return t, true, nil
}

func invalidPictureTimestamp() error {
	return &evaluator.JSONataError{Code: "D3110", Message: "$toMillis: invalid date/time field or timestamp outside the supported range"}
}

func parseWeekday(input []rune, modifier string) (value, consumed int) {
	primary, width, _ := strings.Cut(modifier, ",")
	if primary != "" && primary != "n" && primary != "N" && primary != "Nn" {
		return parseTokenValue(input, modifier)
	}
	maximum := 0
	minimumText, _, _ := strings.Cut(width, "-")
	minimum, _ := strconv.Atoi(minimumText)
	if _, maxText, ok := strings.Cut(width, "-"); ok {
		maximum, _ = strconv.Atoi(maxText)
	}
	// Match the reference's last-entry lookup for ambiguous abbreviations.
	// Full or unambiguous names are preferable when a date must round-trip.
	for isoDay := 7; isoDay >= 1; isoDay-- {
		name := weekdayNames[isoDay%7]
		if maximum > 0 && len(name) > maximum {
			name = name[:maximum]
		}
		if len(input) >= len(name) && strings.EqualFold(string(input[:len(name)]), name) {
			n := len(name)
			for n < minimum && n < len(input) && input[n] == ' ' {
				n++
			}
			return isoDay, n
		}
	}
	return -1, -1
}

func consumeNameOrNumber(runes []rune, modifier string) int {
	if len(runes) == 0 {
		return 0
	}
	// Try name match (weekday names).
	for _, name := range weekdayNames {
		if modifier == "N" || modifier == "n" || modifier == "Nn" || strings.HasPrefix(modifier, "Nn") || strings.HasPrefix(modifier, "N") {
			maxLen := 0
			if strings.Contains(modifier, ",") {
				parts := strings.SplitN(modifier, ",", 2)
				if len(parts) == 2 {
					rangePart := parts[1]
					rangeParts := strings.Split(rangePart, "-")
					if v, err := strconv.Atoi(rangeParts[0]); err == nil {
						maxLen = v
					}
					if len(rangeParts) == 2 {
						if v, err2 := strconv.Atoi(rangeParts[1]); err2 == nil {
							maxLen = v
						}
					}
				}
			}
			nameRunes := []rune(name)
			if maxLen > 0 && maxLen < len(nameRunes) {
				// Abbreviated match: compare against the first maxLen runes.
				abbr := string(nameRunes[:maxLen])
				if len(runes) >= maxLen && strings.EqualFold(string(runes[:maxLen]), abbr) {
					return maxLen
				}
			} else if len(runes) >= len(nameRunes) && strings.EqualFold(string(runes[:len(nameRunes)]), name) {
				// Full-name match: covers maxLen==0 and maxLen>=len(nameRunes).
				return len(nameRunes)
			}
		}
	}
	// Try numeric.
	i := 0
	for i < len(runes) && unicode.IsDigit(runes[i]) {
		i++
	}
	return i
}

func parseTZFromInput(input []rune, modifier string, component rune) (offset, consumed int) {
	primary, width, _ := strings.Cut(modifier, ",")
	minText, _, _ := strings.Cut(width, "-")
	minimum, _ := strconv.Atoi(minText)
	pad := func(n int) int {
		for n < minimum && n < len(input) && input[n] == ' ' {
			n++
		}
		return n
	}
	if len(input) == 0 {
		return 0, 0
	}
	useZ := strings.HasSuffix(primary, "t")
	primary = strings.TrimSuffix(primary, "t")
	if primary == "Z" && !(len(input) >= 3 && strings.EqualFold(string(input[:3]), "GMT")) {
		letter := unicode.ToUpper(input[0])
		if n := strings.IndexRune("ZABCDEFGHIKLM", letter); n >= 0 {
			return n * 3600, pad(1)
		}
		if n := strings.IndexRune("ZNOPQRSTUVWXY", letter); n >= 0 {
			return -n * 3600, pad(1)
		}
	}
	if primary == "N" && len(input) >= 3 && strings.EqualFold(string(input[:3]), "GMT") {
		return 0, pad(3)
	}
	if useZ && unicode.ToUpper(input[0]) == 'Z' {
		return 0, pad(1)
	}
	if primary == "" || primary == "N" || primary == "Z" {
		primary = "01:01"
	}
	zero, count, separators := rune('0'), 0, ""
	for _, r := range primary {
		if r >= '0' && r <= '9' {
			zero = '0'
			count++
		} else if z := unicodeDigitZero(r); z != 0 {
			zero = z
			count++
		} else {
			separators += string(r)
		}
	}
	smallHours := separators == "" && count <= 2
	if smallHours {
		separators = ":"
	}
	i := 0
	if component == 'z' {
		if len(input) < 3 || !strings.EqualFold(string(input[:3]), "GMT") {
			return 0, 0
		}
		i = 3
	}
	if i == len(input) || (input[i] != '+' && input[i] != '-') {
		return 0, 0
	}
	sign := 1
	if input[i] == '-' {
		sign = -1
	}
	i++
	digits, value, separated := 0, 0, false
	for i < len(input) {
		r := input[i]
		if r >= zero && r <= zero+9 {
			digits++
			if digits > 4 {
				return 0, -1
			}
			value = value*10 + int(r-zero)
		} else if strings.ContainsRune(separators, r) {
			if digits == 0 || i+1 == len(input) || input[i-1] < zero || input[i-1] > zero+9 || input[i+1] < zero || input[i+1] > zero+9 {
				return 0, -1
			}
			separated = true
		} else {
			break
		}
		i++
	}
	if digits == 0 {
		return 0, 0
	}
	hours, minutes := value/100, value%100
	if smallHours && !separated {
		hours, minutes = value, 0
	}
	if hours > 23 || minutes > 59 {
		return 0, -1
	}
	return sign * (hours*3600 + minutes*60), pad(i)
}

// A lexically valid integer too large for a calendar field is distinct from
// text that does not match its picture. Never let host integer overflow pick a date.
const calendarFieldOverflow = -2

func parseTokenValue(runes []rune, modifier string) (value, consumed int) {
	if len(runes) == 0 {
		return -1, -1
	}

	primary, _, _ := strings.Cut(modifier, ",")
	representation := primary
	if len(representation) > 1 && strings.ContainsRune("atco", rune(representation[len(representation)-1])) {
		representation = representation[:len(representation)-1]
	}
	// Determine representation type independently of the width modifier.
	// Roman numeral: modifier ends with 'I' or 'i'
	if representation == "I" || representation == "i" {
		return parseRoman(runes)
	}
	// Alphabetic: modifier 'a' or 'A'
	if representation == "a" || representation == "A" {
		return parseAlphabetic(runes)
	}
	// Month name: modifier starts with 'N' or 'Nn' or 'n'
	if modifier == "N" || modifier == "n" || modifier == "Nn" || strings.HasPrefix(modifier, "Nn") || strings.HasPrefix(modifier, "N") {
		return parseMonthName(runes, modifier)
	}
	// Word-based: modifier starts with 'w' or 'W' (e.g., "w", "W", "wo", "Wo", "Wwo", "wwo")
	if primary == "w" || primary == "W" ||
		strings.HasPrefix(primary, "wo") || strings.HasPrefix(primary, "Wo") ||
		strings.HasPrefix(primary, "Ww") || strings.HasPrefix(primary, "ww") {
		return parseWordNumber(runes, modifier)
	}
	// Ordinal suffix: modifier ends with 'o'
	if strings.HasSuffix(primary, "o") {
		return parseOrdinalNumber(runes, modifier)
	}
	// Default: numeric
	return parseNumericValue(runes, modifier)
}

func modifierFieldWidth(modifier string) int {
	if modifier == "" {
		return -1
	}
	// Handle grouping/truncation modifier: starts with ","
	// Pattern: ",[minWidth]-[maxWidth]" or ",*-[maxWidth]"
	if strings.HasPrefix(modifier, ",") {
		// Look for "-N" at the end (max width).
		if idx := strings.LastIndex(modifier, "-"); idx >= 0 {
			part := modifier[idx+1:]
			if n, err := strconv.Atoi(part); err == nil && n > 0 {
				return n
			}
		}
		return -1
	}
	// All-digit modifier: scalar count, not UTF-8 bytes, is the width.
	chars := []rune(modifier)
	if len(chars) < 2 {
		return -1 // "1" = variable width
	}
	for _, r := range chars {
		if unicodeDigitZero(r) == 0 {
			return -1
		}
	}
	return len(chars)
}

func parseNumericValue(runes []rune, modifier string) (value, consumed int) {
	digits, n := parsePictureDigits(runes, modifier)
	if n < 0 {
		return -1, -1
	}
	value, err := strconv.Atoi(digits)
	if err != nil {
		return 0, calendarFieldOverflow
	}
	return value, n
}

func parsePictureDigits(runes []rune, modifier string) (string, int) {
	primary, _, _ := strings.Cut(modifier, ",")
	zero, separators := rune('0'), ""
	for _, r := range primary {
		if z := unicodeDigitZero(r); z != 0 {
			zero = z
		} else if r != '#' {
			separators += string(r)
		}
	}
	var digits strings.Builder
	i := 0
	if i < len(runes) && runes[i] == '-' {
		digits.WriteByte('-')
		i++
	}
	count := 0
	maxW := modifierFieldWidth(modifier)
	for i < len(runes) {
		if maxW > 0 && count >= maxW {
			break
		}
		r := runes[i]
		if r >= zero && r <= zero+9 {
			digits.WriteRune('0' + r - zero)
			count++
		} else if strings.ContainsRune(separators, r) {
		} else {
			break
		}
		i++
	}
	if count == 0 {
		return "", -1
	}
	return digits.String(), i
}

func parseOrdinalNumber(runes []rune, modifier string) (value, consumed int) {
	primary, width, hasWidth := strings.Cut(modifier, ",")
	decimalPicture := strings.TrimSuffix(primary, "o")
	if hasWidth {
		decimalPicture += "," + width
	}
	n, i := parseNumericValue(runes, decimalPicture)
	if i < 0 {
		return n, i
	}
	// Consume optional ordinal suffix (st/nd/rd/th).
	if i+2 <= len(runes) {
		suffix := strings.ToLower(string(runes[i : i+2]))
		if suffix == "st" || suffix == "nd" || suffix == "rd" || suffix == "th" {
			i += 2
		}
	}
	return n, i
}

func parseRoman(runes []rune) (value, consumed int) {
	romanVals := map[rune]int{
		'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000,
		'i': 1, 'v': 5, 'x': 10, 'l': 50, 'c': 100, 'd': 500, 'm': 1000,
	}
	i := 0
	for i < len(runes) {
		if _, ok := romanVals[runes[i]]; !ok {
			break
		}
		i++
	}
	if i == 0 {
		return -1, -1
	}
	// Evaluate Roman numeral.
	total := 0
	prev := 0
	for j := i - 1; j >= 0; j-- {
		v := romanVals[runes[j]]
		if v < prev {
			total -= v
		} else {
			total += v
			prev = v
		}
	}
	return total, i
}

func parseAlphabetic(runes []rune) (value, consumed int) {
	i := 0
	result := 0
	for i < len(runes) {
		r := runes[i]
		if unicode.ToLower(r) < 'a' || unicode.ToLower(r) > 'z' {
			break
		}
		digit := int(unicode.ToLower(r) - 'a' + 1)
		if result > (math.MaxInt-digit)/26 {
			return 0, calendarFieldOverflow
		}
		result = result*26 + digit
		i++
	}
	if i == 0 {
		return -1, -1
	}
	return result, i
}

func parseMonthName(runes []rune, modifier string) (month, consumed int) {
	// Determine max length to match.
	maxLen := 0
	minLen := 0
	if strings.Contains(modifier, ",") {
		parts := strings.SplitN(modifier, ",", 2)
		if len(parts) == 2 {
			rangePart := parts[1]
			rangeParts := strings.Split(rangePart, "-")
			if v, err := strconv.Atoi(rangeParts[0]); err == nil {
				minLen = v
			}
			if len(rangeParts) == 2 {
				if v, err := strconv.Atoi(rangeParts[1]); err == nil {
					maxLen = v
				}
			}
		}
	}

	// Last calendar entry wins a duplicate abbreviation, as in the reference
	// lookup table. This is a parsing convention, not a reversible abbreviation.
	for mi := len(monthNames) - 1; mi >= 0; mi-- {
		name := monthNames[mi]
		if maxLen > 0 && maxLen < len([]rune(name)) {
			// Abbreviated: use first maxLen chars.
			abbr := string([]rune(name)[:maxLen])
			if len(runes) >= maxLen && strings.EqualFold(string(runes[:maxLen]), abbr) {
				n := maxLen
				for n < minLen && n < len(runes) && runes[n] == ' ' {
					n++
				}
				return mi + 1, n
			}
		} else {
			nameRunes := []rune(name)
			if len(runes) >= len(nameRunes) && strings.EqualFold(string(runes[:len(nameRunes)]), name) {
				n := len(nameRunes)
				for n < minLen && n < len(runes) && runes[n] == ' ' {
					n++
				}
				return mi + 1, n
			}
		}
	}
	return -1, -1
}

func parseWordNumber(runes []rune, _ string) (value, consumed int) {
	// Consume up to the end of a word-number expression.
	// Word numbers end at a non-word char that's not '-' or space.
	s := string(runes)
	n, v := parseWordNumberFromString(s)
	return v, n
}

func parseWordNumberFromString(s string) (consumed, value int) {
	ones := []string{
		"zero", "one", "two", "three", "four", "five", "six", "seven",
		"eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen",
		"sixteen", "seventeen", "eighteen", "nineteen",
	}
	tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

	slower := strings.ToLower(s)

	result, n := parseCardinalNumber(slower, ones, tens)
	if n > 0 {
		return n, result
	}
	return 0, -1
}

func parseCardinalNumber(s string, ones, tens []string) (value, consumed int) {
	val, n := parseComplexNumber(s, ones, tens)
	if n > 0 {
		return val, n
	}
	return 0, -1
}

type wordNumberParser struct {
	s    string // lowercase input
	pos  int
	ones []string
	tens []string
}

var onesWithOrdinals = [][]string{
	{"zero", "zeroth"},
	{"one", "first"},
	{"two", "second"},
	{"three", "third"},
	{"four", "fourth"},
	{"five", "fifth"},
	{"six", "sixth"},
	{"seven", "seventh"},
	{"eight", "eighth"},
	{"nine", "ninth"},
	{"ten", "tenth"},
	{"eleven", "eleventh"},
	{"twelve", "twelfth"},
	{"thirteen", "thirteenth"},
	{"fourteen", "fourteenth"},
	{"fifteen", "fifteenth"},
	{"sixteen", "sixteenth"},
	{"seventeen", "seventeenth"},
	{"eighteen", "eighteenth"},
	{"nineteen", "nineteenth"},
}

var tensWithOrdinals = [][]string{
	nil, nil,
	{"twenty", "twentieth"},
	{"thirty", "thirtieth"},
	{"forty", "fortieth"},
	{"fifty", "fiftieth"},
	{"sixty", "sixtieth"},
	{"seventy", "seventieth"},
	{"eighty", "eightieth"},
	{"ninety", "ninetieth"},
}

func (p *wordNumberParser) skipSep() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == ',') {
		p.pos++
	}
	if p.pos+4 <= len(p.s) && strings.HasPrefix(p.s[p.pos:], "and ") {
		p.pos += 4
	}
}

func (p *wordNumberParser) tryWord(word string) bool {
	return p.tryWordOrOrdinal(word, "")
}

func (p *wordNumberParser) tryWordOrOrdinal(word, ordinal string) bool {
	save := p.pos
	p.skipSep()
	for _, candidate := range []string{word, ordinal} {
		if candidate == "" {
			continue
		}
		if !strings.HasPrefix(p.s[p.pos:], candidate) {
			continue
		}
		after := p.s[p.pos+len(candidate):]
		if r, _ := utf8.DecodeRuneInString(after); after != "" && unicode.IsLetter(r) {
			continue
		}
		p.pos += len(candidate)
		return true
	}
	p.pos = save
	return false
}

func (p *wordNumberParser) parseSub100() (int, bool) {
	save := p.pos
	// Try teens/ones (19 down to 10) - with ordinal forms.
	for i := 19; i >= 10; i-- {
		ord := ""
		if i < len(onesWithOrdinals) {
			ord = onesWithOrdinals[i][1]
		}
		if p.tryWordOrOrdinal(p.ones[i], ord) {
			return i, true
		}
	}
	// Try tens (ninety down to twenty) - with ordinal forms.
	for i := 9; i >= 2; i-- {
		tenOrd := ""
		if tensWithOrdinals[i] != nil {
			tenOrd = tensWithOrdinals[i][1]
		}
		if p.tryWordOrOrdinal(p.tens[i], tenOrd) {
			v := i * 10
			// Check if this was ordinal form (standalone, no ones follow).
			// tenOrd already handled by tryWordOrOrdinal above.
			// Optional dash.
			dashSave := p.pos
			if p.pos < len(p.s) && p.s[p.pos] == '-' {
				p.pos++
			}
			// Try ones (with ordinal forms).
			for j := 9; j >= 1; j-- {
				onesOrd := ""
				if j < len(onesWithOrdinals) {
					onesOrd = onesWithOrdinals[j][1]
				}
				if p.tryWordOrOrdinal(p.ones[j], onesOrd) {
					v += j
					return v, true
				}
			}
			// No ones after dash - backtrack dash.
			p.pos = dashSave
			return v, true
		}
	}
	// Include zero/zeroth: year zero is a valid calendar field.
	for i := 9; i >= 0; i-- {
		ord := ""
		if i < len(onesWithOrdinals) {
			ord = onesWithOrdinals[i][1]
		}
		if p.tryWordOrOrdinal(p.ones[i], ord) {
			return i, true
		}
	}
	p.pos = save
	return 0, false
}

func (p *wordNumberParser) parseSub1000() (int, bool) {
	save := p.pos
	// Try ones/teens as hundreds.
	for i := 9; i >= 1; i-- {
		if p.tryWord(p.ones[i]) {
			if p.tryWordOrOrdinal("hundred", "hundredth") {
				v := i * 100
				rem, ok := p.parseSub100()
				if ok {
					v += rem
				}
				return v, true
			}
			// Not hundred - backtrack.
			p.pos = save
			break
		}
	}
	// No hundreds - try direct sub100.
	return p.parseSub100()
}

func parseComplexNumber(s string, ones, tens []string) (total, consumed int) {
	p := &wordNumberParser{s: strings.ToLower(s), ones: ones, tens: tens}
	save := p.pos

	// Try "X hundred" style directly (e.g., "nineteen hundred").
	for i := 19; i >= 1; i-- {
		if p.tryWord(p.ones[i]) {
			if p.tryWordOrOrdinal("hundred", "hundredth") {
				total = i * 100
				rem, ok := p.parseSub100()
				if ok {
					total += rem
				}
				return total, p.pos
			}
			p.pos = save
			break
		}
	}

	// Try "X thousand, Y hundred and Z" style.
	thousandPart, ok := p.parseSub1000()
	if ok {
		if p.tryWordOrOrdinal("thousand", "thousandth") {
			total += thousandPart * 1000
			// Parse hundreds part.
			hundredPart, ok2 := p.parseSub1000()
			if ok2 {
				total += hundredPart
			}
			return total, p.pos
		}
		// No "thousand" - just sub1000.
		total = thousandPart
		return total, p.pos
	}

	return 0, 0
}
