package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/jstring"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
	"github.com/openbindings/jsonata/go/internal/engine/internal/parser"
	"github.com/openbindings/jsonata/go/internal/engine/internal/unicodecase"
)

// ── $string ──────────────────────────────────────────────────────────────────

func fnString(args []any, focus any) (any, error) {
	if len(args) == 0 {
		if focus == nil {
			return nil, nil
		}
		switch focus.(type) {
		case evaluator.BuiltinFunction, evaluator.EnvAwareBuiltin, *evaluator.Lambda, *evaluator.SignedBuiltin, *evaluator.NativeFunction, *evaluator.RegexValue:
			return nil, nil
		}
		return valueToString(focus, false)
	}
	arg := args[0]
	if arg == nil {
		return nil, nil // undefined → undefined
	}
	if len(args) > 2 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$string: takes at most 2 arguments"}
	}
	prettify := false
	if len(args) >= 2 && args[1] != nil {
		switch v := args[1].(type) {
		case bool:
			prettify = v
		case evaluator.BuiltinFunction, evaluator.EnvAwareBuiltin, *evaluator.Lambda, *evaluator.SignedBuiltin, *evaluator.NativeFunction, *evaluator.RegexValue:
			return nil, &evaluator.JSONataError{Code: "D3011", Message: "$string: second argument cannot be a function"}
		default:
			return nil, &evaluator.JSONataError{Code: "T0410", Message: fmt.Sprintf("$string: second argument must be a boolean, got %T", v)}
		}
	}
	return valueToString(arg, prettify)
}

func valueToString(v any, prettify bool) (string, error) {
	if evaluator.IsNull(v) {
		return parser.NullJSON, nil
	}
	switch val := v.(type) {
	case string:
		return val, nil
	case json.Number:
		return evaluator.FormatNumber(val), nil
	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return "", &evaluator.JSONataError{Code: "D3001", Message: "Number out of range"}
		}
		return evaluator.FormatFloat(val), nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	case nil:
		return "", nil // undefined → caller returns nil
	case evaluator.BuiltinFunction, evaluator.EnvAwareBuiltin, *evaluator.Lambda, *evaluator.SignedBuiltin, *evaluator.NativeFunction, *evaluator.RegexValue:
		return "", nil // functions serialize as empty string in JSONata
	case *evaluator.Sequence:
		return valueToString(evaluator.ExternalizeLists(val), prettify)
	default:
		sanitized := sanitizeForJSON(v)
		sanitized, err := evaluator.PrepareJSONStrings(sanitized)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if prettify {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(sanitized); err != nil {
			return "", &evaluator.JSONataError{Code: "D1001", Message: "Number out of range"}
		}
		return strings.TrimRight(buf.String(), "\n"), nil
	}
}

// sanitizeForJSON replaces function values with "" so they can be JSON-marshaled.
// For *OrderedMap, returns a new *OrderedMap preserving insertion order.
func sanitizeForJSON(v any) any {
	if evaluator.IsNull(v) {
		return nil
	}
	switch val := v.(type) {
	case *evaluator.Sequence:
		return sanitizeForJSON(evaluator.ExternalizeLists(val))
	case evaluator.BuiltinFunction, evaluator.EnvAwareBuiltin, *evaluator.Lambda, *evaluator.SignedBuiltin, *evaluator.NativeFunction, *evaluator.RegexValue:
		return ""
	case *evaluator.OrderedMap:
		out := evaluator.NewOrderedMapWithCapacity(val.Len())
		val.Range(func(k string, v any) bool {
			out.Set(k, sanitizeForJSON(v))
			return true
		})
		return out
	case map[string]any:
		out := evaluator.NewOrderedMapWithCapacity(len(val))
		for _, k := range evaluator.MapKeys(val) {
			out.Set(k, sanitizeForJSON(val[k]))
		}
		return out
	case []any:
		out := make([]any, 0, len(val))
		for _, v := range val {
			out = append(out, sanitizeForJSON(v))
		}
		return out
	default:
		return v
	}
}

// ── $length ───────────────────────────────────────────────────────────────────

func fnLength(args []any, _ any) (any, error) {
	switch len(args) {
	case 0:
		return nil, &evaluator.JSONataError{Code: "T0411", Message: "$length: argument 1 is required"}
	case 1:
		if args[0] == nil {
			return nil, nil // propagate undefined
		}
		if s, ok := args[0].(string); ok {
			return float64(len(jstring.Characters(s))), nil
		}
		return nil, &evaluator.JSONataError{Code: "T0410", Message: fmt.Sprintf("$length: argument must be a string, got %T", args[0])}
	default:
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$length: takes exactly 1 argument"}
	}
}

// ── $substring ────────────────────────────────────────────────────────────────

func fnSubstring(args []any, _ any) (any, error) {
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$substring: requires at least 2 arguments"}
	}
	if len(args) > 3 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$substring: too many arguments"}
	}
	if args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$substring: argument 1 must be a string"}
	}
	if !evaluator.IsNumeric(args[1]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$substring: argument 2 must be a number"}
	}

	hasLength := len(args) >= 3 && args[2] != nil
	if hasLength && !evaluator.IsNumeric(args[2]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$substring: argument 3 must be a number"}
	}

	runes := jstring.Characters(s)
	n := len(runes)
	// The documented substr convention truncates start and length separately.
	// Neither index may first be rounded through a host floating-point value.
	start := numeric.ClippedTrunc(args[1], -n, n)

	if start < 0 {
		start = max(n+start, 0)
	}
	if start >= n {
		return "", nil
	}

	if !hasLength {
		return jstring.Join(runes[start:], ""), nil
	}
	length := numeric.ClippedTrunc(args[2], 0, n-start)
	return jstring.Join(runes[start:min(start+length, n)], ""), nil
}

// ── $substringBefore / $substringAfter ────────────────────────────────────────

// substringCutFunc selects which part of strings.Cut to return.
type substringCutFunc func(before, after string) string

func fnSubstringCut(name string, cutFn substringCutFunc) func([]any, any) (any, error) {
	return func(args []any, focus any) (any, error) {
		if len(args) > 2 {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: name + ": too many arguments"}
		}
		var str, sep any
		fromContext := false
		switch len(args) {
		case 0:
			return nil, &evaluator.JSONataError{Code: "T0411", Message: name + ": requires 2 arguments"}
		case 1:
			str = focus
			sep = args[0]
			fromContext = true
		default:
			str = args[0]
			sep = args[1]
		}
		if str == nil {
			return nil, nil
		}
		s, ok1 := str.(string)
		if !ok1 {
			code := "T0410"
			if fromContext {
				code = "T0411"
			}
			return nil, &evaluator.JSONataError{Code: code, Message: name + ": argument 1 must be a string"}
		}
		sep2, ok2 := sep.(string)
		if !ok2 {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: name + ": argument 2 must be a string"}
		}
		units, separator := jstring.Units(s), jstring.Units(sep2)
		if i, err := jstring.IndexUnits(context.Background(), units, separator, 0); err != nil {
			return nil, err
		} else if i >= 0 {
			return cutFn(jstring.FromUnits(units[:i]), jstring.FromUnits(units[i+len(separator):])), nil
		}
		return s, nil
	}
}

var (
	fnSubstringBefore = fnSubstringCut("$substringBefore", func(before, _ string) string { return before })
	fnSubstringAfter  = fnSubstringCut("$substringAfter", func(_, after string) string { return after })
)

// ── $uppercase / $lowercase / $trim ──────────────────────────────────────────

func fnUppercase(args []any, focus any) (any, error) {
	var val any
	if len(args) == 0 {
		val = focus
	} else {
		val = args[0]
	}
	if val == nil {
		return nil, nil
	}
	s, ok := val.(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$uppercase: argument must be a string"}
	}
	return jstring.MapScalars(s, unicodecase.Upper), nil
}

func fnLowercase(args []any, focus any) (any, error) {
	var val any
	if len(args) == 0 {
		val = focus
	} else {
		val = args[0]
	}
	if val == nil {
		return nil, nil
	}
	s, ok := val.(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$lowercase: argument must be a string"}
	}
	return jstring.MapScalars(s, unicodecase.Lower), nil
}

func fnTrim(args []any, focus any) (any, error) {
	var val any
	if len(args) == 0 {
		val = focus
	} else {
		val = args[0]
	}
	if val == nil {
		return nil, nil
	}
	s, ok := val.(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$trim: argument must be a string"}
	}
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }), " "), nil
}

// ── $pad ──────────────────────────────────────────────────────────────────────

func fnPad(args []any, _ any) (any, error) {
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$pad: requires at least 2 arguments"}
	}
	if args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$pad: argument 1 must be a string"}
	}
	if !evaluator.IsNumeric(args[1]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$pad: argument 2 must be a number"}
	}
	width64, widthOk := numeric.TruncInt64(args[1])

	const maxPadWidth = 10_000
	if !widthOk || width64 > maxPadWidth || width64 < -maxPadWidth {
		return nil, &evaluator.JSONataError{Code: "U_OUTPUT_LIMIT", Message: fmt.Sprintf("$pad: width argument exceeds maximum of %d", maxPadWidth)}
	}
	width := int(width64)

	padStr := " "
	if len(args) >= 3 && args[2] != nil {
		p, ok := args[2].(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$pad: argument 3 must be a string"}
		}
		if p == "" {
			padStr = " "
		} else {
			padStr = p
		}
	}

	runes := jstring.Characters(s)
	sLen := len(runes)
	absWidth := max(width, -width)
	if sLen >= absWidth {
		return s, nil
	}
	needed := absWidth - sLen
	padRunes := jstring.Characters(padStr)
	padBuf := make([]string, needed)
	for i := range needed {
		padBuf[i] = padRunes[i%len(padRunes)]
	}
	padding := jstring.Join(padBuf, "")
	if width > 0 {
		return jstring.Concat(s, padding), nil
	}
	return jstring.Concat(padding, s), nil
}

// ── $join ─────────────────────────────────────────────────────────────────────

func fnJoin(args []any, _ any) (any, error) {
	if len(args) == 0 {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$join: argument 1 is required"}
	}
	if args[0] == nil {
		return nil, nil
	}
	if s, ok := args[0].(string); ok {
		return s, nil
	}
	arr := wrapArray(args[0])

	sep := ""
	if len(args) >= 2 && args[1] != nil {
		s, ok := args[1].(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$join: argument 2 must be a string"}
		}
		sep = s
	}

	parts := make([]string, 0, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0412", Message: fmt.Sprintf("$join: array element %d must be a string", i)}
		}
		parts = append(parts, s)
	}
	return jstring.Join(parts, sep), nil
}
