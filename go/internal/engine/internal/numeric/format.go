package numeric

import "github.com/cockroachdb/apd/v3"

// Fixed implements deliberate half-even decimal-place formatting. It is not
// a binary64 conversion or a restriction on carried numerical values.
func (l Limits) Fixed(value any, places int) (string, error) {
	if places < 0 || places > int(l.MaxExponent) {
		return "", &Error{"work-limit", "format width exceeds decimal work budget"}
	}
	x, err := l.parse(value)
	if err != nil {
		return "", err
	}
	x.Abs(x)
	c := l.context(l.MaxDigits)
	r := new(apd.Decimal)
	f, err := c.Quantize(r, x, -int32(places))
	if err = checkedCondition(f, err); err != nil {
		return "", err
	}
	if err = l.checked(r); err != nil {
		return "", err
	}
	return r.Text('f'), nil
}

// Mantissa chooses the exponent before explicit formatting-rounding, as
// XPath F&O 3.1 format-number steps 5 and 6 require. It does not renormalize
// after rounding across a power-of-ten boundary.
func (l Limits) Mantissa(value any, leading int) (any, int, error) {
	if leading < 0 || leading > int(l.MaxDigits) {
		return nil, 0, &Error{"work-limit", "picture width exceeds decimal work budget"}
	}
	x, err := l.parse(value)
	if err != nil {
		return nil, 0, err
	}
	if x.IsZero() {
		return float64(0), 0, nil
	}
	exponent := int(x.Exponent) + int(x.NumDigits()) - leading
	x.Exponent -= int32(exponent)
	return FromText(x.String()), exponent, nil
}
