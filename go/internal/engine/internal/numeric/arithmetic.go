package numeric

import (
	"fmt"

	"github.com/cockroachdb/apd/v3"
)

// Limits bound work, not numerical precision. These are provisional candidate
// safety defaults pending the loop's resource/performance qualification.
type Limits struct {
	MaxDigits   uint32
	MaxExponent int32
}

func DefaultLimits() Limits { return Limits{MaxDigits: 4096, MaxExponent: 4096} }

func (l Limits) Validate() error {
	if l.MaxDigits < 1 || l.MaxDigits > 100000 || l.MaxExponent < 1 || l.MaxExponent > apd.MaxExponent {
		return fmt.Errorf("numeric work limits require 1..100000 coefficient digits and adjusted exponent magnitude")
	}
	return nil
}

type Error struct {
	Kind   string
	Detail string
}

func (e *Error) Error() string {
	code := "U_NUMERIC_LIMIT"
	if e.Kind == "domain" {
		code = "D1001"
	} else if e.Kind == "type" {
		code = "T0410"
	}
	return code + ": JSONata numeric " + e.Kind + ": " + e.Detail
}

func (l Limits) context(precision uint32) apd.Context {
	return apd.Context{Precision: precision, MaxExponent: l.MaxExponent, MinExponent: -l.MaxExponent, Traps: apd.DefaultTraps, Rounding: apd.RoundHalfEven}
}

func (l Limits) checked(x *apd.Decimal) error {
	if x.Form != apd.Finite {
		return &Error{"domain", "non-finite result"}
	}
	if x.IsZero() {
		return nil
	}
	e := int64(x.Exponent) + x.NumDigits() - 1
	if x.NumDigits() > int64(l.MaxDigits) || e > int64(l.MaxExponent) || e < -int64(l.MaxExponent) {
		return &Error{"work-limit", "decimal work budget exceeded"}
	}
	return nil
}

func (l Limits) parse(v any) (*apd.Decimal, error) {
	s, ok := Text(v)
	if !ok {
		return nil, &Error{"type", "expected a finite JSON number"}
	}
	// Normalize first, so harmless trailing zeroes do not consume precision and
	// do not trigger false exponent-overflow errors in the arithmetic library.
	_, coefficient, exponent := parts(s)
	if len(coefficient) > int(l.MaxDigits) || !exponent.IsInt64() || exponent.Int64() > int64(l.MaxExponent) || exponent.Int64() < -int64(l.MaxExponent) {
		return nil, &Error{"work-limit", "operand exceeds decimal work budget"}
	}
	x, _, err := apd.NewFromString(Format(s))
	if err != nil {
		return nil, err
	}
	x.Reduce(x)
	return x, l.checked(x)
}

func checkedCondition(flags apd.Condition, err error) error {
	if err != nil || flags&(apd.Overflow|apd.Underflow|apd.Subnormal|apd.InvalidOperation|apd.DivisionImpossible) != 0 {
		return &Error{"work-limit", fmt.Sprintf("decimal operation rejected (%s): %v", flags, err)}
	}
	return nil
}

func (l Limits) divide(x, y *apd.Decimal) (*apd.Decimal, error) {
	if y.IsZero() {
		return nil, &Error{"domain", "division by zero"}
	}
	// A terminating quotient needs no more than this many coefficient digits.
	// Probe it exactly; otherwise assign a single 34-significant-digit result.
	p := x.NumDigits() + 4*y.NumDigits() + 2
	if p > int64(l.MaxDigits) {
		return nil, &Error{"work-limit", "exact quotient probe exceeds decimal work budget"}
	}
	c := l.context(uint32(p))
	r := new(apd.Decimal)
	f, err := c.Quo(r, x, y)
	if err = checkedCondition(f, err); err != nil {
		return nil, err
	}
	if f.Inexact() {
		c = l.context(34)
		f, err = c.Quo(r, x, y)
		if err = checkedCondition(f, err); err != nil {
			return nil, err
		}
	}
	r.Reduce(r)
	return r, l.checked(r)
}

func (l Limits) Calculate(op string, a, b any, places int32) (any, error) {
	x, err := l.parse(a)
	if err != nil {
		return nil, err
	}
	var y *apd.Decimal
	if b != nil {
		y, err = l.parse(b)
		if err != nil {
			return nil, err
		}
	}
	r := new(apd.Decimal)
	c := l.context(0)
	var f apd.Condition
	switch op {
	case "+":
		f, err = c.Add(r, x, y)
	case "-":
		f, err = c.Sub(r, x, y)
	case "*":
		f, err = c.Mul(r, x, y)
	case "/":
		r, err = l.divide(x, y)
	case "%":
		if y.IsZero() {
			return nil, &Error{"domain", "modulo by zero"}
		}
		c = l.context(l.MaxDigits)
		f, err = c.Rem(r, x, y)
	case "neg":
		r.Neg(x)
	case "abs":
		r.Abs(x)
	case "floor":
		c = l.context(l.MaxDigits)
		f, err = c.Floor(r, x)
	case "ceil":
		c = l.context(l.MaxDigits)
		f, err = c.Ceil(r, x)
	case "round":
		if places > l.MaxExponent || places < -l.MaxExponent {
			return nil, &Error{"work-limit", "round scale exceeds decimal work budget"}
		}
		c = l.context(l.MaxDigits)
		f, err = c.Quantize(r, x, -places)
	case "sqrt":
		if x.Sign() < 0 {
			return nil, &Error{"domain", "square root of a negative number"}
		}
		p := x.NumDigits() + 2
		if p > int64(l.MaxDigits) {
			return nil, &Error{"work-limit", "exact root probe exceeds decimal work budget"}
		}
		c = l.context(uint32(p))
		f, err = c.Sqrt(r, x)
		if err = checkedCondition(f, err); err != nil {
			return nil, err
		}
		probe := new(apd.Decimal)
		exact := l.context(0)
		pf, pe := exact.Mul(probe, r, r)
		if pe = checkedCondition(pf, pe); pe != nil {
			return nil, pe
		}
		if probe.Cmp(x) != 0 {
			c = l.context(34)
			f, err = c.Sqrt(r, x)
		}
	case "pow", "**":
		if !IsInteger(b) {
			r, err = l.fractionalPower(x, y)
			break
		}
		i, intErr := y.Int64()
		if intErr != nil {
			return nil, &Error{"work-limit", "integral exponent exceeds the supported work domain"}
		}
		// Bound the exponent before repeated squaring. Large decimal exponents
		// must never overflow a native loop counter or select float arithmetic.
		if i > int64(l.MaxDigits) || i < -int64(l.MaxDigits) {
			return nil, &Error{"work-limit", "power exceeds decimal work budget"}
		}
		negative := i < 0
		if negative {
			i = -i
		}
		r.SetInt64(1)
		base := new(apd.Decimal).Set(x)
		for i > 0 {
			if i&1 == 1 {
				f, err = c.Mul(r, r, base)
				if err = checkedCondition(f, err); err != nil {
					return nil, err
				}
				if err = l.checked(r); err != nil {
					return nil, err
				}
			}
			i >>= 1
			if i > 0 {
				f, err = c.Mul(base, base, base)
				if err = checkedCondition(f, err); err != nil {
					return nil, err
				}
				if err = l.checked(base); err != nil {
					return nil, err
				}
			}
		}
		if negative {
			r, err = l.divide(apd.New(1, 0), r)
		}
	default:
		return nil, &Error{"unsupported", op}
	}
	if err != nil {
		return nil, err
	}
	if err = checkedCondition(f, err); err != nil {
		return nil, err
	}
	r.Reduce(r)
	if err = l.checked(r); err != nil {
		return nil, err
	}
	return FromText(r.String()), nil
}
