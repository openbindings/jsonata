// Package jstring preserves JSONata's UTF-16 string values in native Go strings.
// Scalar Unicode uses UTF-8. Isolated surrogate code units use their three-byte
// WTF-8 encoding; they must be escaped, not replaced, at a JSON codec boundary.
package jstring

import (
	"context"
	"unicode/utf16"
	"unicode/utf8"
)

// Units converts the admitted UTF-8/WTF-8 representation without replacement.
func Units(s string) []uint16 {
	out := make([]uint16, 0, len(s))
	for len(s) > 0 {
		if len(s) >= 3 {
			if u := surrogate(s[:3]); u != 0 {
				out = append(out, u)
				s = s[3:]
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(s)
		s = s[n:]
		if r > 0xffff {
			a, b := utf16.EncodeRune(r)
			out = append(out, uint16(a), uint16(b))
		} else {
			out = append(out, uint16(r))
		}
	}
	return out
}

// FromUnits canonicalizes pairs but keeps each isolated code unit intact.
func FromUnits(units []uint16) string {
	out := make([]byte, 0, len(units))
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u >= 0xd800 && u <= 0xdbff && i+1 < len(units) && units[i+1] >= 0xdc00 && units[i+1] <= 0xdfff {
			out = utf8.AppendRune(out, utf16.DecodeRune(rune(u), rune(units[i+1])))
			i++
		} else {
			out = append(out, CodeUnit(u)...)
		}
	}
	return string(out)
}

// SliceUnits is for APIs (notably regex offsets) defined in UTF-16 code units.
func SliceUnits(s string, start, end int) string { return FromUnits(Units(s)[start:end]) }

// Characters splits Unicode scalars and isolated code units without replacement.
// An astral scalar is one character, matching JSONata's substring/length rules.
func Characters(s string) []string {
	var out []string
	for len(s) > 0 {
		var size int
		if len(s) >= 3 && surrogate(s[:3]) != 0 {
			size = 3
		} else {
			_, size = utf8.DecodeRuneInString(s)
		}
		out = append(out, s[:size])
		s = s[size:]
	}
	return out
}

// Join canonicalizes newly adjacent pairs while preserving all other units.
func Join(parts []string, separator string) string {
	var out []byte
	for i, part := range parts {
		if i > 0 {
			out = appendCanonical(out, separator)
		}
		out = appendCanonical(out, part)
	}
	return string(out)
}

func appendCanonical(out []byte, s string) []byte {
	if len(out) >= 3 && len(s) >= 3 {
		hi, lo := surrogate(string(out[len(out)-3:])), surrogate(s[:3])
		if hi >= 0xd800 && hi <= 0xdbff && lo >= 0xdc00 && lo <= 0xdfff {
			out = utf8.AppendRune(out[:len(out)-3], rune(0x10000+(uint32(hi)-0xd800)*0x400+uint32(lo)-0xdc00))
			return append(out, s[3:]...)
		}
	}
	return append(out, s...)
}

// MapScalars applies a Unicode operation without passing isolated units to a
// Go rune decoder, which would replace them before the operation starts.
func MapScalars(s string, operation func(string) string) string {
	if utf8.ValidString(s) {
		return operation(s)
	}
	// Keep valid runs together: casing may depend on neighbouring characters
	// (for example Greek final sigma). Isolated units are uncased boundaries.
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(s); {
		if len(s)-i >= 3 && surrogate(s[i:i+3]) != 0 {
			parts = append(parts, operation(s[start:i]), s[i:i+3])
			i += 3
			start = i
		} else {
			_, n := utf8.DecodeRuneInString(s[i:])
			i += n
		}
	}
	parts = append(parts, operation(s[start:]))
	return Join(parts, "")
}

// IndexUnits is a linear-time literal search over code units, with cooperative
// cancellation. It avoids quadratic scans on adversarial repeated prefixes.
func IndexUnits(ctx context.Context, s, pattern []uint16, start int) (int, error) {
	if err := ctx.Err(); err != nil {
		return -1, err
	}
	if start < 0 {
		start = 0
	}
	if start > len(s) {
		return -1, nil
	}
	if len(pattern) == 0 {
		return start, nil
	}
	failure := make([]int, len(pattern))
	for i, j := 1, 0; i < len(pattern); i++ {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return -1, err
			}
		}
		for j > 0 && pattern[i] != pattern[j] {
			j = failure[j-1]
		}
		if pattern[i] == pattern[j] {
			j++
		}
		failure[i] = j
	}
	for i, j := start, 0; i < len(s); i++ {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return -1, err
			}
		}
		for j > 0 && s[i] != pattern[j] {
			j = failure[j-1]
		}
		if s[i] == pattern[j] {
			j++
		}
		if j == len(pattern) {
			return i + 1 - j, nil
		}
	}
	return -1, nil
}

// CodeUnit encodes one UTF-16 code unit, including an isolated surrogate.
func CodeUnit(unit uint16) string {
	if unit < 0xd800 || unit > 0xdfff {
		return string(rune(unit))
	}
	return string([]byte{0xe0 | byte(unit>>12), 0x80 | byte(unit>>6)&0x3f, 0x80 | byte(unit)&0x3f})
}

func surrogate(s string) uint16 {
	if len(s) != 3 || s[0] != 0xed || s[1] < 0xa0 || s[1] > 0xbf || s[2]&0xc0 != 0x80 {
		return 0
	}
	return uint16(s[0]&0xf)<<12 | uint16(s[1]&0x3f)<<6 | uint16(s[2]&0x3f)
}

// Concat preserves the canonical encoding when concatenation joins two halves
// of an astral character. Other byte sequences are unchanged.
func Concat(a, b string) string {
	if len(a) >= 3 && len(b) >= 3 {
		hi, lo := surrogate(a[len(a)-3:]), surrogate(b[:3])
		if hi >= 0xd800 && hi <= 0xdbff && lo >= 0xdc00 && lo <= 0xdfff {
			r := rune(0x10000 + (uint32(hi)-0xd800)*0x400 + uint32(lo) - 0xdc00)
			return a[:len(a)-3] + string(r) + b[3:]
		}
	}
	return a + b
}

// Valid reports whether s uses canonical UTF-8/WTF-8, rather than arbitrary
// malformed bytes. A surrogate pair must use its scalar UTF-8 encoding.
func Valid(s string) bool {
	var previous uint16
	for len(s) > 0 {
		if len(s) >= 3 {
			if unit := surrogate(s[:3]); unit != 0 {
				if previous >= 0xd800 && previous <= 0xdbff && unit >= 0xdc00 {
					return false
				}
				previous = unit
				s = s[3:]
				continue
			}
		}
		_, size := utf8.DecodeRuneInString(s)
		if size == 1 && s[0] >= 0x80 {
			return false
		}
		previous = 0
		s = s[size:]
	}
	return true
}
