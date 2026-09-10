package numeric

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

// TruncInt64 is an explicit control conversion. It truncates the exact decimal
// before conversion, rejects out-of-range values, and never rounds a token to
// a neighboring integer through binary64.
func TruncInt64(value any) (int64, bool) {
	s, ok := Text(value)
	if !ok {
		return 0, false
	}
	negative, c, e := parts(s)
	if e.Sign() < 0 || c == "0" {
		return 0, true
	}
	if !e.IsInt64() || e.Int64() > 18 {
		return 0, false
	}
	point := int(e.Int64()) + 1
	if point < len(c) {
		c = c[:point]
	} else {
		c += strings.Repeat("0", point-len(c))
	}
	if negative {
		c = "-" + c
	}
	i, err := strconv.ParseInt(c, 10, 64)
	return i, err == nil
}

func FloorInt64(value any) (int64, bool) {
	i, ok := TruncInt64(value)
	if !ok {
		return 0, false
	}
	if Compare(value, json.Number(strconv.FormatInt(i, 10))) < 0 {
		if i == (-1 << 63) {
			return 0, false
		}
		i--
	}
	return i, true
}

// ClippedTrunc is for bounded control positions, not number-value conversion.
// Clipping is deliberate (as in substring), and precedes narrowing to an int.
func ClippedTrunc(value any, lower, upper int) int {
	if Compare(value, float64(lower)) <= 0 {
		return lower
	}
	if Compare(value, float64(upper)) >= 0 {
		return upper
	}
	i, ok := TruncInt64(value)
	if !ok {
		panic("bounded control index is not representable")
	}
	return int(i)
}

// IterationLimit preserves count < limit: fractional limits admit ceil(limit)
// iterations. -1 means no language limit within this host's addressable output
// domain; independent resource limits still apply. It does not cap a result.
func IterationLimit(value any) int {
	i, ok := TruncInt64(value)
	maxInt := int64(int(^uint(0) >> 1))
	if !ok || i >= maxInt-1 {
		return -1
	}
	if Compare(value, json.Number(strconv.FormatInt(i, 10))) > 0 {
		i++
	}
	return int(i)
}

// Index implements numeric predicate indexing: floor, then count negative
// indexes from the end. An unbounded out-of-range index simply selects nothing.
func Index(value any, length int) (int, bool) {
	i, ok := FloorInt64(value)
	if !ok {
		return 0, false
	}
	if i < 0 {
		i += int64(length)
	}
	if i < 0 || i >= int64(length) {
		return 0, false
	}
	return int(i), true
}

func (l Limits) Integer(value any) (*big.Int, error) {
	if !IsInteger(value) {
		return nil, &Error{"type", "expected an integer"}
	}
	x, err := l.parse(value)
	if err != nil {
		return nil, err
	}
	i, ok := new(big.Int).SetString(x.Text('f'), 10)
	if !ok {
		return nil, &Error{"type", "invalid integer"}
	}
	return i, nil
}
