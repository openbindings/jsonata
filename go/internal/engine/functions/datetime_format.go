package functions

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
)

func parseTZ(v any) (*time.Location, error) {
	s, ok := v.(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "timezone must be a string"}
	}
	// The language argument is a signed HHMM offset. Do not consult ambient
	// timezone databases or admit undocumented filesystem-backed zone names.
	if s == "0000" {
		s = "+0000"
	} // retained reference UTC spelling
	if len(s) != 5 || (s[0] != '+' && s[0] != '-') {
		return nil, &evaluator.JSONataError{Code: "D3110", Message: "timezone must be a signed HHMM offset"}
	}
	offset, parseErr := parseNumericTZ(s)
	if parseErr == nil {
		return time.FixedZone(s, offset), nil
	}
	return nil, &evaluator.JSONataError{Code: "D3110", Message: fmt.Sprintf("invalid timezone %q: %v", s, parseErr)}
}

func parseNumericTZ(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty timezone")
	}
	sign := 1
	switch s[0] {
	case '+':
		s = s[1:]
	case '-':
		sign = -1
		s = s[1:]
	}
	// If no sign prefix and all digits, treat as positive offset.
	// Remove colon if present
	s = strings.ReplaceAll(s, ":", "")
	if len(s) != 4 {
		return 0, fmt.Errorf("invalid tz format: %s", s)
	}
	h, err1 := strconv.Atoi(s[:2])
	m, err2 := strconv.Atoi(s[2:])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid tz digits")
	}
	return sign * (h*3600 + m*60), nil
}

var (
	weekdayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	monthNames   = []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
)

const errTokenSentinel = "\x00ERR\x00"

func formatWithPicture(t time.Time, picture string, standardOnly bool) (string, error) {
	parts, err := parseDatePicture(picture, standardOnly)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, part := range parts {
		if part.isToken {
			s, err := formatPicturePart(t, part)
			if err != nil {
				return "", err
			}
			sb.WriteString(s)
		} else {
			sb.WriteString(part.literal)
		}
	}
	return sb.String(), nil
}

func isoWeekThursday(t time.Time) time.Time {
	return t.AddDate(0, 0, 3-(int(t.Weekday())+6)%7)
}

// weekOfMonth returns the week-of-month for a Thursday t.
// Callers always pass isoWeekThursday(t), so t.Day() mod 7 directly
// determines the ordinal Thursday position in the month.
func weekOfMonth(t time.Time) int {
	return (t.Day() + 6) / 7
}

func formatTimezone(component byte, modifier string, t time.Time) string {
	_, offset := t.Zone()

	useZ := strings.HasSuffix(modifier, "t")
	mod := strings.TrimSuffix(modifier, "t")
	if mod == "Z" && offset%3600 == 0 && offset >= -43200 && offset <= 43200 {
		if offset < 0 {
			return string("ZNOPQRSTUVWXY"[-offset/3600])
		}
		return string("ZABCDEFGHIKLM"[offset/3600])
	}
	if mod == "N" && offset == 0 {
		return "GMT"
	}
	if mod == "" || mod == "Z" || mod == "N" {
		mod = "01:01"
	}

	if offset == 0 && useZ {
		return "Z"
	}

	prefix := ""
	if component == 'z' {
		prefix = "GMT"
	}

	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	mins := (offset % 3600) / 60

	digits, separators := 0, 0
	for _, r := range mod {
		if (r >= '0' && r <= '9') || unicodeDigitZero(r) != 0 {
			digits++
		} else {
			separators++
		}
	}
	if digits < 1 || digits > 4 {
		return errTokenSentinel
	}
	value := hours*100 + mins
	if separators == 0 && digits <= 2 {
		value = hours
	}
	text, err := formatIntegerDecimal(big.NewInt(int64(value)), mod)
	if err != nil {
		return errTokenSentinel
	}
	if separators == 0 && digits <= 2 && mins != 0 {
		zero := unicodeDigitZero([]rune(mod)[0])
		text += ":" + applyDigitFamilyRune(fmt.Sprintf("%02d", mins), zero)
	}
	return prefix + sign + text
}
