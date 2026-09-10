package functions

import (
	"fmt"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
)

// ── $keys ─────────────────────────────────────────────────────────────────────

func fnKeys(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	seq := evaluator.CreateSequence()
	seen := make(map[string]bool)
	var visit func(any)
	visit = func(value any) {
		if arr, ok := evaluator.ArrayValues(value); ok {
			for _, item := range arr {
				visit(item)
			}
		} else if evaluator.IsMap(value) {
			for _, key := range evaluator.MapKeys(value) {
				if !seen[key] {
					seen[key] = true
					seq.Values = append(seq.Values, key)
				}
			}
		}
	}
	visit(args[0])
	return seq, nil
}

// ── $spread ───────────────────────────────────────────────────────────────────

func fnSpread(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	spreadOne := func(obj any) []any {
		keys := evaluator.MapKeys(obj)
		result := make([]any, len(keys))
		for i, k := range keys {
			om := evaluator.NewOrderedMap()
			v, _ := evaluator.MapGet(obj, k)
			om.Set(k, v)
			result[i] = om
		}
		return result
	}
	if evaluator.IsMap(args[0]) {
		return &evaluator.Sequence{Values: spreadOne(args[0])}, nil
	}
	if arr, ok := evaluator.ArrayValues(args[0]); ok {
		var result []any
		for _, item := range arr {
			spread, err := fnSpread([]any{item}, nil)
			if err != nil {
				return nil, err
			}
			if values, ok := evaluator.ArrayValues(spread); ok {
				result = append(result, values...)
			} else if spread != nil {
				result = append(result, spread)
			}
		}
		if result == nil {
			return []any{}, nil
		}
		return result, nil
	}
	return args[0], nil
}

// ── $merge ────────────────────────────────────────────────────────────────────

func fnMerge(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	arr, ok := evaluator.ArrayValues(args[0])
	if !ok {
		if evaluator.IsMap(args[0]) {
			return args[0], nil
		}
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$merge: argument must be an array of objects"}
	}
	result := evaluator.NewOrderedMap()
	for _, item := range arr {
		if !evaluator.IsMap(item) {
			return nil, &evaluator.JSONataError{Code: "T0412", Message: "$merge: array elements must be objects"}
		}
		evaluator.MapRange(item, func(k string, v any) bool {
			result.Set(k, v)
			return true
		})
	}
	return result, nil
}

func fillSiftArgs(buf []any, value any, key string, obj any) {
	switch len(buf) {
	case 0:
	case 1:
		buf[0] = value
	case 2:
		buf[0] = value
		buf[1] = key
	default:
		buf[0] = value
		buf[1] = key
		buf[2] = obj
	}
}

// ── $sift ─────────────────────────────────────────────────────────────────────

func makeFnSift(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, focus any, env *evaluator.Environment) (any, error) {
		var objVal any
		var fn any
		switch len(args) {
		case 0:
			return nil, &evaluator.JSONataError{Code: "D3006", Message: "$sift: requires at least 1 argument"}
		case 1:
			objVal = focus
			fn = args[0]
		default:
			objVal = args[0]
			fn = args[1]
		}
		if objVal == nil {
			return nil, nil
		}
		if !evaluator.IsMap(objVal) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$sift: argument 1 must be an object"}
		}

		result := evaluator.NewOrderedMap()
		keys := evaluator.MapKeys(objVal)
		callArgs := hofArgsBuf(hofArity(fn))
		for _, ks := range keys {
			val, _ := evaluator.MapGet(objVal, ks)
			fillSiftArgs(callArgs, val, ks, objVal)
			res, err := evalFn(fn, callArgs, focus, env)
			if err != nil {
				return nil, err
			}
			if evaluator.ToBoolean(res) {
				result.Set(ks, val)
			}
		}
		if result.Len() == 0 {
			return nil, nil
		}
		return result, nil
	}
}

// ── $each ─────────────────────────────────────────────────────────────────────

func makeFnEach(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, focus any, env *evaluator.Environment) (any, error) {
		var objVal any
		var fn any
		switch len(args) {
		case 0:
			return nil, &evaluator.JSONataError{Code: "D3006", Message: "$each: requires at least 1 argument"}
		case 1:
			objVal = focus
			fn = args[0]
		default:
			objVal = args[0]
			fn = args[1]
		}
		if objVal == nil {
			return nil, nil
		}
		if !evaluator.IsMap(objVal) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$each: argument 1 must be an object"}
		}

		keys := evaluator.MapKeys(objVal)
		seq := evaluator.CreateSequence()
		callArgs := hofArgsBuf(hofArity(fn))
		for _, ks := range keys {
			val, _ := evaluator.MapGet(objVal, ks)
			fillSiftArgs(callArgs, val, ks, objVal)
			res, err := evalFn(fn, callArgs, focus, env)
			if err != nil {
				return nil, err
			}
			if res != nil {
				seq.Values = append(seq.Values, res)
			}
		}
		return seq, nil
	}
}

// ── $error ────────────────────────────────────────────────────────────────────

func fnError(args []any, _ any) (any, error) {
	msg := "an error was thrown"
	if len(args) > 1 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$error: takes at most 1 argument"}
	}
	if len(args) == 1 && args[0] != nil {
		s, ok := args[0].(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$error: argument must be a string"}
		}
		msg = s
	}
	return nil, &evaluator.JSONataError{Code: "D3137", Message: msg}
}

// ── $lookup ───────────────────────────────────────────────────────────────────

func fnLookup(args []any, _ any) (any, error) {
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$lookup: requires 2 arguments"}
	}
	if args[0] == nil || args[1] == nil {
		return nil, nil
	}
	key, ok := args[1].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: fmt.Sprintf("$lookup: key must be a string, got %T", args[1])}
	}

	return lookupValue(args[0], key), nil
}

// Lookup over arrays collects selected values using the existing JSONata
// reference convention: flatten one selected array level at each traversal
// boundary. Lookup on a single object retains its property's array unchanged.
func lookupValue(input any, key string) any {
	if evaluator.IsMap(input) {
		val, exists := evaluator.MapGet(input, key)
		if !exists {
			return nil
		}
		return val
	}
	if arr, ok := evaluator.ArrayValues(input); ok {
		result := evaluator.CreateSequence()
		for _, item := range arr {
			value := lookupValue(item, key)
			if values, ok := evaluator.ArrayValues(value); ok {
				result.Values = append(result.Values, values...)
			} else if value != nil {
				result.Values = append(result.Values, value)
			}
		}
		return result
	}
	return nil
}
