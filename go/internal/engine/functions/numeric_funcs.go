package functions

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
)

// ── $number ───────────────────────────────────────────────────────────────────

func fnNumber(args []any, focus any) (any, error) {
	var arg any
	switch len(args) {
	case 0:
		arg = focus
	case 1:
		arg = args[0]
	default:
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$number: too many arguments"}
	}
	if arg == nil {
		return nil, nil
	}
	if evaluator.IsNull(arg) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$number: cannot cast null to number"}
	}
	switch v := arg.(type) {
	case float64:
		return v, nil
	case json.Number:
		if evaluator.IsNumeric(v) {
			return numeric.FromText(string(v)), nil
		}
		return nil, &evaluator.JSONataError{Code: "D3030", Message: "$number: invalid number"}
	case string:
		if value, ok := numeric.CastString(v); ok {
			return value, nil
		}
		return nil, &evaluator.JSONataError{Code: "D3030", Message: fmt.Sprintf("$number: unable to cast %q to a number", v)}
	case bool:
		if v {
			return float64(1), nil
		}
		return float64(0), nil
	case []any:
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$number: cannot cast array to number"}
	case *evaluator.OrderedMap, map[string]any:
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$number: cannot cast object to number"}
	default:
		return nil, &evaluator.JSONataError{Code: "T0410", Message: fmt.Sprintf("$number: unsupported type %T", v)}
	}
}

// ── $abs ──────────────────────────────────────────────────────────────────────

func fnAbs(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		args = []any{focus}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$abs: argument must be a number"}
	}
	return env.NumberLimits().Calculate("abs", args[0], nil, 0)
}

// ── $floor ────────────────────────────────────────────────────────────────────

func fnFloor(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		args = []any{focus}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$floor: argument must be a number"}
	}
	return env.NumberLimits().Calculate("floor", args[0], nil, 0)
}

// ── $ceil ─────────────────────────────────────────────────────────────────────

func fnCeil(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		args = []any{focus}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$ceil: argument must be a number"}
	}
	return env.NumberLimits().Calculate("ceil", args[0], nil, 0)
}

// ── $round ────────────────────────────────────────────────────────────────────

func fnRound(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		args = []any{focus}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$round: argument must be a number"}
	}
	scale := int32(0)
	if len(args) > 1 && args[1] != nil {
		if !evaluator.IsNumeric(args[1]) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$round: scale must be an integer"}
		}
		// Compare the exact control before narrowing; a too-wide valid integer
		// is resource rejection, not a malformed numerical argument.
		bound := int(env.NumberLimits().MaxExponent)
		if numeric.Compare(args[1], -float64(bound)) < 0 || numeric.Compare(args[1], float64(bound)) > 0 {
			return nil, &numeric.Error{Kind: "work-limit", Detail: "round scale exceeds decimal work budget"}
		}
		if !numeric.IsInteger(args[1]) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$round: scale must be an integer"}
		}
		scale = int32(numeric.ClippedTrunc(args[1], -bound, bound))
	}
	return env.NumberLimits().Calculate("round", args[0], nil, scale)
}

// ── $power ────────────────────────────────────────────────────────────────────

func fnPower(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 1 {
		args = []any{focus, args[0]}
	}
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$power: requires 2 arguments"}
	}
	if args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) || !evaluator.IsNumeric(args[1]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$power: arguments must be numbers"}
	}
	if numeric.Compare(args[0], float64(0)) < 0 && !numeric.IsInteger(args[1]) || numeric.Zero(args[0]) && numeric.Compare(args[1], float64(0)) < 0 {
		return nil, &evaluator.JSONataError{Code: "D3061", Message: "$power: no finite real result"}
	}
	if numeric.Compare(args[1], float64(0.5)) == 0 {
		return env.NumberLimits().Calculate("sqrt", args[0], nil, 0)
	}
	return env.NumberLimits().Calculate("pow", args[0], args[1], 0)
}

// ── $sqrt ─────────────────────────────────────────────────────────────────────

func fnSqrt(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		args = []any{focus}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$sqrt: expected a number"}
	}
	if numeric.Compare(args[0], float64(0)) < 0 {
		return nil, &evaluator.JSONataError{Code: "D3060", Message: "$sqrt: square root of a negative number"}
	}
	return env.NumberLimits().Calculate("sqrt", args[0], nil, 0)
}

// ── $random ───────────────────────────────────────────────────────────────────

func fnRandom(_ []any, _ any) (any, error) {
	return rand.Float64(), nil
}

// ── $sum ──────────────────────────────────────────────────────────────────────

func fnSum(args []any, focus any, env *evaluator.Environment) (any, error) {
	return aggregateNumbers("sum", args, focus, env)
}

// ── $max ──────────────────────────────────────────────────────────────────────

func fnMax(args []any, focus any, env *evaluator.Environment) (any, error) {
	return aggregateNumbers("max", args, focus, env)
}

// ── $min ──────────────────────────────────────────────────────────────────────

func fnMin(args []any, focus any, env *evaluator.Environment) (any, error) {
	return aggregateNumbers("min", args, focus, env)
}

// ── $average ──────────────────────────────────────────────────────────────────

func fnAverage(args []any, focus any, env *evaluator.Environment) (any, error) {
	return aggregateNumbers("average", args, focus, env)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// toNumberArray normalises a single number or []any into []any.
// Returns nil if the input is neither.
func toNumberArray(v any) []any {
	switch val := v.(type) {
	case []any:
		return val
	case float64:
		return []any{val}
	case json.Number:
		return []any{val}
	case *evaluator.Sequence:
		return evaluator.CollapseToSlice(val)
	default:
		return nil
	}
}

func aggregateNumbers(op string, args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) == 0 {
		args = []any{focus}
	}
	if len(args) != 1 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$" + op + ": expects one argument"}
	}
	if args[0] == nil {
		return nil, nil
	}
	arr := toNumberArray(args[0])
	if arr == nil {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$" + op + ": expected numbers"}
	}
	if len(arr) == 0 {
		if op == "sum" {
			return float64(0), nil
		}
		return nil, nil
	}
	var result any = float64(0)
	for i, v := range arr {
		if i%1024 == 0 {
			if err := env.Context().Err(); err != nil {
				return nil, err
			}
		}
		if !evaluator.IsNumeric(v) {
			return nil, &evaluator.JSONataError{Code: "T0412", Message: "$" + op + ": expected numbers"}
		}
		if op == "max" || op == "min" {
			if i == 0 || (op == "max" && numeric.Compare(v, result) > 0) || (op == "min" && numeric.Compare(v, result) < 0) {
				result = v
			}
		} else {
			var err error
			result, err = env.NumberLimits().Calculate("+", result, v, 0)
			if err != nil {
				return nil, err
			}
		}
	}
	if op == "average" {
		return env.NumberLimits().Calculate("/", result, float64(len(arr)), 0)
	}
	return result, nil
}
