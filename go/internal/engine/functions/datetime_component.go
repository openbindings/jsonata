package functions

import (
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

// Formatting uses the existing integer engine; date components do not have a
// second, reduced implementation of digit families, grouping or word formats.
func formatPicturePart(t time.Time, part picturePart) (string, error) {
	component, primary := part.component, part.modifier
	minWidth, maxWidth := 0, 0
	if comma := strings.LastIndexByte(primary, ','); comma >= 0 {
		width := strings.Split(primary[comma+1:], "-")
		minWidth, _ = strconv.Atoi(width[0]) // shared picture parser validated it
		if len(width) > 1 {
			maxWidth, _ = strconv.Atoi(width[1])
		}
		primary = primary[:comma]
	}
	if component == 'f' {
		return formatFraction(t.Nanosecond()/1e6, part.modifier)
	}
	if component == 'Z' || component == 'z' {
		result := formatTimezone(byte(component), primary, t)
		if result == errTokenSentinel {
			return "", &evaluator.JSONataError{Code: "D3134", Message: "invalid timezone picture"}
		}
		if minWidth > len(result) {
			result += strings.Repeat(" ", minWidth-len(result))
		}
		return result, nil
	}
	modifier := ""
	if len(primary) > 1 && strings.ContainsRune("atco", rune(primary[len(primary)-1])) {
		modifier, primary = primary[len(primary)-1:], primary[:len(primary)-1]
	}
	if primary == "" {
		primary = defaultDatePresentation(component)
	}
	name := primary == "n" || primary == "N" || primary == "Nn"
	var value int
	var text string
	switch component {
	case 'Y':
		value = t.Year()
		if value < 0 {
			value = -value
		}
	case 'M':
		value = int(t.Month())
		if name {
			text = monthNames[value-1]
		}
	case 'D':
		value = t.Day()
	case 'd':
		value = t.YearDay()
	case 'F':
		value = (int(t.Weekday())+6)%7 + 1
		if name {
			text = weekdayNames[t.Weekday()]
		}
	case 'W':
		_, value = t.ISOWeek()
	case 'w':
		value = weekOfMonth(isoWeekThursday(t))
	case 'X':
		value, _ = t.ISOWeek()
	case 'x':
		value = int(isoWeekThursday(t).Month())
		if name {
			text = monthNames[value-1]
		}
	case 'H':
		value = t.Hour()
	case 'h':
		value = (t.Hour()+11)%12 + 1
	case 'm':
		value = t.Minute()
	case 's':
		value = t.Second()
	case 'P':
		text = "am"
		if t.Hour() >= 12 {
			text = "pm"
		}
		// F&O leaves P rendering implementation-defined. Retain the
		// reference convention: case selection, without width truncation.
		if primary == "N" {
			text = strings.ToUpper(text)
		}
		return text, nil
	case 'C':
		return "ISO", nil
	case 'E':
		if t.Year() < 0 {
			return "-", nil
		}
		return "", nil
	}
	if text != "" {
		if primary == "n" {
			text = strings.ToLower(text)
		} else if primary == "N" {
			text = strings.ToUpper(text)
		}
		if maxWidth > 0 && len(text) > maxWidth {
			text = text[:maxWidth]
		}
		if minWidth > len(text) {
			text += strings.Repeat(" ", minWidth-len(text))
		}
		return text, nil
	}
	if name {
		primary = defaultDatePresentation(component)
	}
	chars := []rune(primary)
	zero, mandatory, total := rune(0), 0, 0
	for _, r := range chars {
		z := unicodeDigitZero(r)
		if r >= '0' && r <= '9' {
			z = '0'
		}
		if z != 0 {
			zero = z
			mandatory++
			total++
		} else if r == '#' {
			total++
		}
	}
	if zero == 0 && !strings.Contains("|a|A|i|I|w|W|Ww|", "|"+primary+"|") {
		if component == 'F' {
			part.modifier = "n"
			if comma := strings.LastIndexByte(part.token, ','); comma >= 0 {
				part.modifier += part.token[comma:]
			}
			return formatPicturePart(t, part)
		}
		primary = defaultDatePresentation(component)
		chars = []rune(primary)
		zero = '0'
		mandatory = len(chars)
		total = mandatory
	}
	if component == 'Y' {
		// Retain the established picture-width precedence for the year:
		// explicit maximum, otherwise the adjusted decimal presentation.
		width := maxWidth
		if width == 0 && max(total, minWidth) >= 2 {
			width = max(total, minWidth)
		}
		if width > 0 && width < 7 {
			divisor := 1
			for range width {
				divisor *= 10
			}
			value %= divisor
		}
		if maxWidth > 0 && zero != 0 {
			chars = []rune(strings.Repeat(string(zero), maxWidth))
			mandatory = maxWidth
		}
	}
	if zero != 0 {
		for i := len(chars) - 1; i >= 0 && mandatory < minWidth; i-- {
			if chars[i] == '#' {
				chars[i] = zero
				mandatory++
			}
		}
		if mandatory < minWidth {
			chars = append([]rune(strings.Repeat(string(zero), minWidth-mandatory)), chars...)
		}
		primary = string(chars)
	}
	if modifier != "" {
		primary += ";" + modifier
	}
	result, err := formatIntegerWithPicture(big.NewInt(int64(value)), primary, numeric.DefaultLimits())
	if err != nil {
		return "", err
	}
	if zero == 0 && minWidth > len(result) {
		result += strings.Repeat(" ", minWidth-len(result))
	}
	return result, nil
}

func defaultDatePresentation(component rune) string {
	switch component {
	case 'F', 'P', 'C', 'E':
		return "n"
	case 'm', 's':
		return "01"
	default:
		return "1"
	}
}
