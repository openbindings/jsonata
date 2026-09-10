package functions

import (
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"
)

// F&O 3.1 §9.8.4.5: reverse the decimal pattern and fractional digits, reuse
// the integer formatter, reverse its output, then truncate excess right digits.
func formatFraction(millis int, modifier string) (string, error) {
	pattern, width, hasWidth := modifier, "", false
	if comma := strings.LastIndex(modifier, ","); comma >= 0 {
		pattern, width, hasWidth = modifier[:comma], modifier[comma+1:], true
	}
	if pattern == "" {
		pattern = "1"
	}
	chars := []rune(pattern)
	zero, mandatory := rune(0), 0
	for _, c := range chars {
		z := unicodeDigitZero(c)
		if c >= '0' && c <= '9' {
			z = '0'
		}
		if z != 0 {
			if zero != 0 && zero != z {
				return "", fmt.Errorf("mixed digit families")
			}
			zero = z
			mandatory++
		}
	}
	// Nondecimal fractional presentations use the ordinary decimal fallback.
	if zero == 0 {
		chars, zero, mandatory = []rune{'1'}, '0', 1
	}
	isDigit := func(c rune) bool { return c >= zero && c <= zero+9 }
	if !hasWidth && len(chars) == 1 {
		chars = append(chars, '#', '#')
	}
	if hasWidth {
		parts := strings.Split(width, "-")
		if len(parts) > 2 {
			return "", fmt.Errorf("invalid fractional width")
		}
		readWidth := func(text string) (int, error) {
			if text == "*" {
				return 0, nil
			}
			for _, c := range text {
				if c < '0' || c > '9' {
					return 0, fmt.Errorf("invalid fractional width")
				}
			}
			n, err := strconv.Atoi(text)
			if err != nil || n < 1 || n > 10000 {
				return 0, fmt.Errorf("invalid or excessive fractional width")
			}
			return n, nil
		}
		minimum, err := readWidth(parts[0])
		if err != nil {
			return "", err
		}
		maximum := 0
		if len(parts) == 2 {
			maximum, err = readWidth(parts[1])
			if err != nil {
				return "", err
			}
		}
		if maximum != 0 && minimum > maximum {
			return "", fmt.Errorf("invalid fractional width range")
		}
		for i := 0; i < len(chars) && mandatory < minimum; i++ {
			if chars[i] == '#' {
				chars[i] = zero
				mandatory++
			}
		}
		for mandatory < minimum {
			chars = append(chars, zero)
			mandatory++
		}
		count := 0
		for _, c := range chars {
			if isDigit(c) || c == '#' {
				count++
			}
		}
		for count < maximum {
			chars = append(chars, '#')
			count++
		}
	}
	maximum := 0
	for _, c := range chars {
		if isDigit(c) || c == '#' {
			maximum++
		}
	}
	slices.Reverse(chars)
	digits := []rune(fmt.Sprintf("%03d", millis))
	slices.Reverse(digits)
	n, _ := strconv.ParseInt(string(digits), 10, 64)
	rendered, err := formatIntegerDecimal(big.NewInt(n), string(chars))
	if err != nil {
		return "", err
	}
	result := []rune(rendered)
	slices.Reverse(result)
	count, end := 0, 0
	for i, c := range result {
		if isDigit(c) {
			count++
			if count > maximum {
				break
			}
			end = i + 1
		}
	}
	return string(result[:end]), nil
}
