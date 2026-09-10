package functions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

// ── $count ────────────────────────────────────────────────────────────────────

func fnCount(args []any, _ any) (any, error) {
	if len(args) > 1 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$count: takes exactly 1 argument"}
	}
	if len(args) == 0 || args[0] == nil {
		return float64(0), nil
	}
	switch v := args[0].(type) {
	case *evaluator.Sequence:
		return float64(len(v.Values)), nil
	case []any:
		return float64(len(v)), nil
	default:
		return float64(1), nil
	}
}

// ── $append ───────────────────────────────────────────────────────────────────

func fnAppend(args []any, _ any) (any, error) {
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$append: requires 2 arguments"}
	}
	// If either argument is undefined, return the other unchanged.
	if args[0] == nil {
		return args[1], nil
	}
	if args[1] == nil {
		return args[0], nil
	}
	a := wrapArray(args[0])
	b := wrapArray(args[1])
	const maxAppendSize = 10_000_000
	if len(a)+len(b) > maxAppendSize {
		return nil, &evaluator.JSONataError{
			Code:    "D3010",
			Message: fmt.Sprintf("$append: result array exceeds maximum size of %d elements", maxAppendSize),
		}
	}
	return slices.Concat(a, b), nil
}

func wrapArray(v any) []any {
	if v == nil {
		return []any{}
	}
	if arr, ok := evaluator.ArrayValues(v); ok {
		return arr
	}
	return []any{v}
}

// Preserve a caller's array/sequence identity through higher-order callback
// arguments; only scalar-to-array coercion creates a new container.
func arrayArgument(v any) any {
	if _, ok := evaluator.ArrayValues(v); ok {
		return v
	}
	return evaluator.NewArray(wrapArray(v))
}

// ── $sort ─────────────────────────────────────────────────────────────────────

func makeFnSort(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, focus any, env *evaluator.Environment) (any, error) {
		var arrVal any
		var fn any
		switch len(args) {
		case 0:
			arrVal = focus
		case 1:
			switch args[0].(type) {
			case evaluator.BuiltinFunction, evaluator.EnvAwareBuiltin, *evaluator.Lambda, *evaluator.SignedBuiltin, *evaluator.NativeFunction, *evaluator.RegexValue:
				// arg is a function → use focus as the array
				arrVal = focus
				fn = args[0]
			default:
				arrVal = args[0]
			}
		default:
			arrVal = args[0]
			fn = args[1]
		}
		if arrVal == nil {
			return nil, nil
		}
		arr := wrapArray(arrVal)
		if len(arr) <= 1 {
			return arrayArgument(arrVal), nil
		}

		if fn == nil && len(arr) > 0 {
			allNum := true
			allStr := true
			for _, item := range arr {
				if !evaluator.IsNumeric(item) {
					allNum = false
				}
				if _, ok := item.(string); !ok {
					allStr = false
				}
			}
			if !allNum && !allStr {
				return nil, &evaluator.JSONataError{
					Code:    "D3070",
					Message: "$sort: each element of the array must be of the same type - strings or numbers",
				}
			}
		}
		sorted := slices.Clone(arr)
		var cmpFn func(a, b any) (int, error)
		if fn != nil {
			// JSONata $sort comparator: fn(a, b) returns true when a should come
			// before b. We call fn(b, a) and map true→-1 (a<b), false→0 (a>=b).
			// SortItemsErr only tests < 0, so +1 is unnecessary.
			sortArgs := make([]any, 2)
			cmpFn = func(a, b any) (int, error) {
				sortArgs[0] = b
				sortArgs[1] = a
				result, err := evalFn(fn, sortArgs, focus, env)
				if err != nil {
					return 0, err
				}
				if evaluator.ToBoolean(result) {
					return -1, nil
				}
				return 0, nil
			}
		} else {
			cmpFn = defaultCompare
		}
		if err := evaluator.SortItemsErr(sorted, cmpFn); err != nil {
			return nil, err
		}
		return sorted, nil
	}
}

func defaultCompare(a, b any) (int, error) {
	if evaluator.IsNumeric(a) && evaluator.IsNumeric(b) {
		return numeric.Compare(a, b), nil
	}
	if av, aOk := a.(string); aOk {
		if bv, bOk := b.(string); bOk {
			if av < bv {
				return -1, nil
			} else if av > bv {
				return 1, nil
			}
			return 0, nil
		}
	}
	return 0, fmt.Errorf("$sort: cannot compare %T and %T", a, b)
}

// ── $reverse ──────────────────────────────────────────────────────────────────

func fnReverse(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	arr := wrapArray(args[0])
	if len(arr) <= 1 {
		return arrayArgument(args[0]), nil
	}
	result := slices.Clone(arr)
	slices.Reverse(result)
	return result, nil
}

// ── $shuffle ──────────────────────────────────────────────────────────────────

func fnShuffle(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	arr := wrapArray(args[0])
	if len(arr) <= 1 {
		return arrayArgument(args[0]), nil
	}
	result := slices.Clone(arr)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result, nil
}

// ── $distinct ─────────────────────────────────────────────────────────────────

func fnDistinct(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	if evaluator.IsNull(args[0]) {
		return evaluator.Null, nil
	}
	arr, ok := evaluator.ArrayValues(args[0])
	if !ok {
		if seq, ok2 := args[0].(*evaluator.Sequence); ok2 {
			arr = evaluator.CollapseToSlice(seq)
		} else {
			return args[0], nil
		}
	}
	if len(arr) <= 1 {
		// No dedup needed — return as plain array. Sequence wrapping
		// (and thus singleton-unwrap) only applies when dedup actually
		// reduced a multi-element input.
		return args[0], nil
	}
	result := make([]any, 0, len(arr))
	seen := make(map[any]bool, len(arr))
	type numberKey string
	type nullKey struct{}
	var complexItems []any
	for _, v := range arr {
		switch v.(type) {
		case string, bool:
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		case json.Number, float64:
			s, ok := numeric.Text(v)
			if !ok {
				return nil, &evaluator.JSONataError{Code: "D1001", Value: v}
			}
			key := numberKey(numeric.Format(s))
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		default:
			if v == nil || evaluator.IsNull(v) {
				var key any = nullKey{}
				if v == nil {
					key = nil
				}
				if !seen[key] {
					seen[key] = true
					result = append(result, v)
				}
				continue
			}
			found := false
			for _, existing := range complexItems {
				if evaluator.DeepEqual(v, existing) {
					found = true
					break
				}
			}
			if !found {
				complexItems = append(complexItems, v)
				result = append(result, v)
			}
		}
	}
	if evaluator.IsSequence(args[0]) {
		return &evaluator.Sequence{Values: result}, nil
	}
	return evaluator.NewArray(result), nil
}

// ── $zip ──────────────────────────────────────────────────────────────────────

func fnZip(args []any, _ any) (any, error) {
	if len(args) == 0 {
		return []any{}, nil
	}
	// Determine shortest length.
	minLen := -1
	arrays := make([][]any, 0, len(args))
	for _, arg := range args {
		if arg == nil {
			arr := []any{}
			arrays = append(arrays, arr)
			if minLen < 0 || len(arr) < minLen {
				minLen = len(arr)
			}
			continue
		}
		arr := wrapArray(arg)
		arrays = append(arrays, arr)
		if minLen < 0 || len(arr) < minLen {
			minLen = len(arr)
		}
	}
	if minLen <= 0 {
		return []any{}, nil
	}
	result := make([]any, minLen)
	for i := range minLen {
		tuple := make([]any, len(arrays))
		for j, arr := range arrays {
			if i < len(arr) {
				tuple[j] = arr[i]
			}
		}
		result[i] = tuple
	}
	return result, nil
}
