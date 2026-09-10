package evaluator

import (
	"fmt"
	"math/big"

	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

func evalRange(left, right any, env *Environment) (any, error) {
	if left != nil && !numeric.IsInteger(left) {
		return nil, &JSONataError{Code: "T2003", Message: "left side of range must be an integer"}
	}
	if right != nil && !numeric.IsInteger(right) {
		return nil, &JSONataError{Code: "T2004", Message: "right side of range must be an integer"}
	}
	if left == nil || right == nil {
		return nil, nil
	}
	if numeric.Compare(left, right) > 0 {
		return nil, nil
	}
	lo, err := env.NumberLimits().Integer(left)
	if err != nil {
		return nil, err
	}
	hi, err := env.NumberLimits().Integer(right)
	if err != nil {
		return nil, err
	}
	size := new(big.Int).Sub(hi, lo)
	const maxRange = 10_000_000
	if !size.IsInt64() || size.Int64() >= maxRange {
		return nil, &JSONataError{Code: "D2014", Message: fmt.Sprintf("range must not exceed %d items", maxRange)}
	}
	count := int(size.Int64()) + 1
	result := make([]any, 0, count)
	one := big.NewInt(1)
	const maxExact = 9007199254740991
	safeIntegerRange := lo.IsInt64() && hi.IsInt64() && lo.Int64() >= -maxExact && hi.Int64() <= maxExact
	first := lo.Int64()
	for i := 0; i < count; i++ {
		if i%10000 == 0 {
			if err := env.Context().Err(); err != nil {
				return nil, err
			}
		}
		if safeIntegerRange {
			result = append(result, float64(first+int64(i)))
		} else {
			result = append(result, numeric.FromText(lo.String()))
			lo.Add(lo, one)
		}
	}
	return result, nil
}
