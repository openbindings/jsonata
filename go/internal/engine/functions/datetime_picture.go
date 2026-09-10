package functions

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
)

// parseDatePicture is shared by formatting and parsing: a malformed picture
// must not become a literal on one path and an error on the other. X/x are raw
// reference compatibility extensions, not F&O 3.1 component specifiers.
func parseDatePicture(picture string, standardOnly bool) ([]picturePart, error) {
	parts := make([]picturePart, 0)
	runes := []rune(picture)
	for i := 0; i < len(runes); {
		ch := runes[i]
		if (ch == '[' || ch == ']') && i+1 < len(runes) && runes[i+1] == ch {
			parts = append(parts, picturePart{literal: string(ch)})
			i += 2
			continue
		}
		if ch == ']' {
			return nil, pictureSyntaxError()
		}
		if ch != '[' {
			parts = append(parts, picturePart{literal: string(ch)})
			i++
			continue
		}
		end := i + 1
		for end < len(runes) && runes[end] != ']' {
			end++
		}
		if end == len(runes) {
			return nil, pictureSyntaxError()
		}
		token := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, string(runes[i+1:end]))
		if token == "" || !validComponents[rune(token[0])] || (standardOnly && strings.ContainsRune("Xx", rune(token[0]))) {
			return nil, &evaluator.JSONataError{Code: "D3132", Message: "unknown date/time picture component"}
		}
		modifier := token[1:]
		presentation, width := modifier, ""
		if comma := strings.LastIndexByte(modifier, ','); comma >= 0 {
			presentation, width = modifier[:comma], modifier[comma:]
		}
		if (presentation == "N" || presentation == "n" || presentation == "Nn") && strings.ContainsRune("YDdWwXHhms", rune(token[0])) {
			// Unsupported names fall back to the component's default (F&O
			// 3.1 §9.8.4.8), not an undocumented language exclusion.
			presentation = "1"
			if token[0] == 'm' || token[0] == 's' {
				presentation = "01"
			}
			modifier = presentation + width
			token = token[:1] + modifier
		}
		if comma := strings.LastIndexByte(modifier, ','); comma >= 0 {
			if err := validatePictureWidth(modifier[comma+1:]); err != nil {
				return nil, err
			}
		}
		parts = append(parts, picturePart{isToken: true, token: token, component: rune(token[0]), modifier: modifier})
		i = end + 1
	}
	return parts, nil
}

func pictureSyntaxError() error {
	return &evaluator.JSONataError{Code: "D3135", Message: "malformed date/time picture"}
}

func validatePictureWidth(width string) error {
	limits := strings.Split(width, "-")
	if len(limits) > 2 {
		return pictureSyntaxError()
	}
	values := [2]int{}
	for i, part := range limits {
		if part == "*" {
			continue
		}
		if part == "" {
			return pictureSyntaxError()
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return pictureSyntaxError()
			}
		}
		n, err := strconv.Atoi(part)
		if err != nil || n > 10000 {
			return &evaluator.JSONataError{Code: "D3137", Message: "date picture width exceeds materialization budget"}
		}
		if n < 1 {
			return pictureSyntaxError()
		}
		values[i] = n
	}
	if values[0] > 0 && values[1] > 0 && values[0] > values[1] {
		return pictureSyntaxError()
	}
	return nil
}
