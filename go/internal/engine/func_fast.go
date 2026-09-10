package gnata

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/openbindings/jsonata/go/internal/engine/internal/parser"
	"github.com/tidwall/gjson"
)

func isJSONArray(r *gjson.Result) bool {
	return r.Type == gjson.JSON && r.Raw != "" && r.Raw[0] == '['
}

func isJSONObject(r *gjson.Result) bool {
	return r.Type == gjson.JSON && r.Raw != "" && r.Raw[0] == '{'
}

type funcFastHandler func(*gjson.Result, *parser.FuncFastPath) (any, bool, error)

// Only qualified optimizations are installed. Numeric functions use the same
// assigned-value implementation as full evaluation, with no legacy float code.
var funcFastHandlers = map[parser.FuncFastKind]funcFastHandler{
	parser.FuncFastExists:    evalFuncExists,
	parser.FuncFastContains:  evalFuncContains,
	parser.FuncFastKeys:      evalFuncKeys,
	parser.FuncFastLowercase: evalFuncLowercase,
	parser.FuncFastUppercase: evalFuncUppercase,
	parser.FuncFastTrim:      evalFuncTrim,
	parser.FuncFastLength:    evalFuncLength,
	parser.FuncFastType:      evalFuncType,
	parser.FuncFastCount:     evalFuncCount,
	parser.FuncFastReverse:   evalFuncReverse,
}

func evalFunc(f *parser.FuncFastPath, data json.RawMessage, mapData map[string]json.RawMessage) (result any, handled bool, err error) {
	// Unqualified numeric optimizations are absent from the dispatch table.
	r := resolveGjsonPath(data, mapData, f.Path)
	if !r.Exists() {
		// Fall through to full evaluator — gjson doesn't auto-map through
		// arrays, so the path might still resolve via the AST walker.
		return nil, false, nil
	}
	if h, ok := funcFastHandlers[f.Kind]; ok {
		return h(&r, f)
	}
	return nil, false, nil
}

func evalFuncExists(_ *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	return true, true, nil
}

func evalFuncContains(r *gjson.Result, f *parser.FuncFastPath) (result any, handled bool, err error) {
	if r.Type == gjson.String {
		// Non-ASCII can match a single surrogate half and must use the UTF-16 path.
		if !asciiOnly(r.Str) || !asciiOnly(f.StrArg) {
			return nil, false, nil
		}
		return strings.Contains(r.Str, f.StrArg), true, nil
	}
	return nil, false, nil
}

func asciiOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

func evalFuncKeys(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if isJSONObject(r) {
		var keys []any
		r.ForEach(func(key, _ gjson.Result) bool {
			keys = append(keys, key.String())
			return true
		})
		switch len(keys) {
		case 0:
			return nil, true, nil
		case 1:
			return keys[0], true, nil
		default:
			return keys, true, nil
		}
	}
	return nil, false, nil
}

func evalFuncLowercase(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if r.Type == gjson.String && asciiOnly(r.Str) {
		return strings.ToLower(r.Str), true, nil
	}
	return nil, false, nil
}

func evalFuncUppercase(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if r.Type == gjson.String && asciiOnly(r.Str) {
		return strings.ToUpper(r.Str), true, nil
	}
	return nil, false, nil
}

func evalFuncTrim(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if r.Type == gjson.String {
		return strings.Join(strings.FieldsFunc(r.Str, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }), " "), true, nil
	}
	return nil, false, nil
}

func evalFuncLength(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if r.Type == gjson.String && asciiOnly(r.Str) {
		return float64(utf8.RuneCountInString(r.Str)), true, nil
	}
	return nil, false, nil
}

func evalFuncType(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	switch r.Type {
	case gjson.String:
		return "string", true, nil
	case gjson.Number:
		return "number", true, nil
	case gjson.True, gjson.False:
		return "boolean", true, nil
	case gjson.Null:
		return parser.NullJSON, true, nil
	case gjson.JSON:
		if r.Raw != "" {
			switch r.Raw[0] {
			case '[':
				return "array", true, nil
			case '{':
				return "object", true, nil
			}
		}
		return nil, false, nil
	default:
		return nil, false, nil
	}
}

func evalFuncCount(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if isJSONArray(r) {
		count := 0
		r.ForEach(func(_, _ gjson.Result) bool {
			count++
			return true
		})
		return float64(count), true, nil
	}
	return float64(1), true, nil
}

func evalFuncReverse(r *gjson.Result, _ *parser.FuncFastPath) (result any, handled bool, err error) {
	if isJSONArray(r) {
		elems := make([]any, 0)
		r.ForEach(func(_, elem gjson.Result) bool {
			elems = append(elems, gjsonValueToAny(&elem))
			return true
		})
		slices.Reverse(elems)
		return elems, true, nil
	}
	return nil, false, nil
}
