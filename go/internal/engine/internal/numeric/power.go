package numeric

import "github.com/cockroachdb/apd/v3"

// fractionalPower is a function-specific approximation, not exact arithmetic.
// Two fixed guard precisions must agree after assignment to 34 significant
// digits. This detects unstable rounding; agreement is not a mathematical
// proof of correct rounding for every input. Qualification uses an independent
// higher-precision oracle and records this limitation explicitly.
func (l Limits) fractionalPower(x, y *apd.Decimal) (*apd.Decimal, error) {
	if x.Sign() < 0 || (x.IsZero() && y.Sign() < 0) {
		return nil, &Error{"domain", "power has no finite real result"}
	}
	if x.IsZero() {
		return apd.New(0, 0), nil
	}
	if x.Cmp(apd.New(1, 0)) == 0 {
		return apd.New(1, 0), nil
	}
	if l.MaxDigits < 120 {
		return nil, &Error{"work-limit", "fractional power requires the fixed guard work budget"}
	}
	var assigned *apd.Decimal
	for _, precision := range []uint32{80, 120} {
		c := l.context(precision)
		result := new(apd.Decimal)
		f, err := c.Pow(result, x, y)
		if err = checkedCondition(f, err); err != nil {
			return nil, err
		}
		if result.IsZero() {
			return nil, &Error{"work-limit", "nonzero power underflowed the supported work domain"}
		}
		c = l.context(34)
		f, err = c.Round(result, result)
		if err = checkedCondition(f, err); err != nil {
			return nil, err
		}
		result.Reduce(result)
		if assigned != nil && assigned.Cmp(result) != 0 {
			return nil, &Error{"work-limit", "fractional power rounding is unstable at the fixed guard precisions"}
		}
		assigned = result
	}
	return assigned, l.checked(assigned)
}
