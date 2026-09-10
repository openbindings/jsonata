// Package numeric implements the candidate's assigned-decimal value model.
// json.Number is a JSON number, never a JSON object or a string-valued escape.
package numeric

import (
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var (
	grammar      = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
	castGrammar  = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
	radixGrammar = regexp.MustCompile(`^(0[xX][0-9a-fA-F]+|0[oO][0-7]+|0[bB][01]+)$`)
)

func Valid(s string) bool { return grammar.MatchString(s) }

// CastString implements the decimal and radix spellings accepted by $number.
func CastString(s string) (any, bool) {
	if radixGrammar.MatchString(s) {
		n, ok := new(big.Int).SetString(s, 0)
		if ok {
			return FromText(n.String()), true
		}
	}
	if castGrammar.MatchString(s) {
		return FromText(Format(s)), true
	}
	return nil, false
}

func Text(v any) (string, bool) {
	switch n := v.(type) {
	case json.Number:
		return string(n), Valid(string(n))
	case float64:
		if !math.IsInf(n, 0) && !math.IsNaN(n) {
			return strconv.FormatFloat(n, 'g', -1, 64), true
		}
	}
	return "", false
}

// parts requires a validated token. Its exponent is logarithmic: comparison
// and carriage never expand 1e100000000 into a hundred million digit integer.
func parts(s string) (negative bool, coefficient string, adjusted *big.Int) {
	negative = strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	head, exp, hasExp := strings.Cut(strings.ToLower(s), "e")
	adjusted = new(big.Int)
	if hasExp {
		adjusted.SetString(strings.TrimPrefix(exp, "+"), 10)
	}
	point := strings.IndexByte(head, '.')
	fraction := 0
	if point >= 0 {
		fraction = len(head) - point - 1
		head = head[:point] + head[point+1:]
	}
	head = strings.TrimLeft(head, "0")
	if head == "" {
		return false, "0", new(big.Int)
	}
	adjusted.Add(adjusted, big.NewInt(int64(len(head)-fraction-1)))
	return negative, strings.TrimRight(head, "0"), adjusted
}

// CompareText compares assigned values, not their lexical spellings. Both
// arguments must be valid JSON number tokens.
func CompareText(a, b string) int {
	an, ac, ae := parts(a)
	bn, bc, be := parts(b)
	if ac == "0" && bc == "0" {
		return 0
	}
	if an != bn {
		if an {
			return -1
		}
		return 1
	}
	var order int
	switch {
	case ac == "0":
		order = -1
	case bc == "0":
		order = 1
	default:
		order = ae.Cmp(be)
		if order == 0 {
			for i := 0; i < max(len(ac), len(bc)); i++ {
				x, y := byte('0'), byte('0')
				if i < len(ac) {
					x = ac[i]
				}
				if i < len(bc) {
					y = bc[i]
				}
				if x < y {
					order = -1
					break
				}
				if x > y {
					order = 1
					break
				}
			}
		}
	}
	if an {
		return -order
	}
	return order
}

func Compare(a, b any) int {
	x, ok := Text(a)
	if !ok {
		panic("numeric.Compare: invalid left operand")
	}
	y, ok := Text(b)
	if !ok {
		panic("numeric.Compare: invalid right operand")
	}
	return CompareText(x, y)
}

func Zero(v any) bool {
	s, ok := Text(v)
	if !ok {
		return false
	}
	_, c, _ := parts(s)
	return c == "0"
}

func IsInteger(v any) bool {
	s, ok := Text(v)
	if !ok {
		return false
	}
	_, c, e := parts(s)
	return c == "0" || e.Cmp(big.NewInt(int64(len(c)-1))) >= 0
}

// Format uses decimal notation in the usual JSONata/ECMAScript display range,
// without converting the assigned value through binary floating point.
func Format(s string) string {
	negative, c, e := parts(s)
	if c == "0" {
		return "0"
	}
	prefix := ""
	if negative {
		prefix = "-"
	}
	if e.IsInt64() && e.Int64() >= -6 && e.Int64() < 21 {
		point := int(e.Int64()) + 1
		if point <= 0 {
			return prefix + "0." + strings.Repeat("0", -point) + c
		}
		if point >= len(c) {
			return prefix + c + strings.Repeat("0", point-len(c))
		}
		return prefix + c[:point] + "." + c[point:]
	}
	mantissa := c[:1]
	if len(c) > 1 {
		mantissa += "." + c[1:]
	}
	sign := ""
	if e.Sign() >= 0 {
		sign = "+"
	}
	return prefix + mantissa + "e" + sign + e.String()
}

// FromText uses a native value only if its JSON decimal image is identical.
// It does not make binary64 rounding part of the expression's semantics.
func FromText(s string) any {
	if !Valid(s) {
		panic("numeric.FromText: invalid JSON number")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err == nil && !math.IsInf(f, 0) && CompareText(s, strconv.FormatFloat(f, 'g', -1, 64)) == 0 {
		return f
	}
	return json.Number(Format(s))
}
