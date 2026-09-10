package functions

import (
	"strconv"
	"strings"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/ecmaregex"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/jstring"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
)

func callable(v any) bool {
	switch v.(type) {
	case evaluator.BuiltinFunction, evaluator.EnvAwareBuiltin, *evaluator.Lambda, *evaluator.SignedBuiltin, *evaluator.NativeFunction, *evaluator.RegexValue:
		return true
	}
	return false
}

func matcherError() error {
	return &evaluator.JSONataError{Code: "T1010", Message: "matcher must return undefined or a match object with string match, numeric start/end, array groups and callable next"}
}

type matcherResult struct {
	value, groups, next any
	text                string
	start, end          int
}

func evaluateMatcher(fn any, args []any, s string, evalFn EvalFn, env *evaluator.Environment) (*matcherResult, error) {
	if !callable(fn) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "argument must be a matcher function"}
	}
	if err := env.Context().Err(); err != nil {
		return nil, err
	}
	value, err := evalFn(fn, args, nil, env)
	if err != nil || value == nil {
		return nil, err
	}
	if !evaluator.IsMap(value) {
		return nil, matcherError()
	}
	text, _ := evaluator.MapGet(value, "match")
	start, _ := evaluator.MapGet(value, "start")
	end, _ := evaluator.MapGet(value, "end")
	groups, _ := evaluator.MapGet(value, "groups")
	next, _ := evaluator.MapGet(value, "next")
	str, ok := text.(string)
	if !ok || !evaluator.IsNumeric(start) || !evaluator.IsNumeric(end) || !callable(next) {
		return nil, matcherError()
	}
	if _, ok := evaluator.ArrayValues(groups); !ok {
		return nil, matcherError()
	}
	n := len(jstring.Units(s))
	if !numeric.IsInteger(start) || !numeric.IsInteger(end) || numeric.Compare(start, 0.0) < 0 || numeric.Compare(end, start) < 0 || numeric.Compare(end, float64(n)) > 0 {
		return nil, matcherError()
	}
	return &matcherResult{value, groups, next, str, numeric.ClippedTrunc(start, 0, n), numeric.ClippedTrunc(end, 0, n)}, nil
}

func matcherLimit(args []any, index int, code string) (int, error) {
	if len(args) <= index || args[index] == nil {
		return -1, nil
	}
	if !evaluator.IsNumeric(args[index]) {
		return 0, &evaluator.JSONataError{Code: "T0410", Message: "matcher limit must be a number"}
	}
	if numeric.Compare(args[index], 0.0) < 0 {
		return 0, &evaluator.JSONataError{Code: code, Message: "matcher limit must not be negative"}
	}
	return numeric.IterationLimit(args[index]), nil
}

func matcherString(args []any) (string, bool, error) {
	if len(args) == 0 {
		return "", false, &evaluator.JSONataError{Code: "T0410", Message: "string argument required"}
	}
	if args[0] == nil {
		return "", false, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return "", false, &evaluator.JSONataError{Code: "T0410", Message: "argument 1 must be a string"}
	}
	return s, true, nil
}

func matchResultSeq(result []any) any {
	if len(result) == 0 {
		return nil
	}
	return &evaluator.Sequence{Values: result}
}

func matcherIteration(count int) error {
	if count >= 100000 {
		return &evaluator.JSONataError{Code: "U_REGEX_LIMIT", Message: "matcher result budget exhausted"}
	}
	return nil
}

func makeFnMatch(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, _ any, env *evaluator.Environment) (any, error) {
		s, ok, err := matcherString(args)
		if err != nil || !ok {
			return nil, err
		}
		if len(args) < 2 || !callable(args[1]) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "argument 2 must be a matcher"}
		}
		limit, err := matcherLimit(args, 2, "D3040")
		if err != nil || limit == 0 {
			return nil, err
		}
		m, err := evaluateMatcher(args[1], []any{s}, s, evalFn, env)
		if err != nil {
			return nil, err
		}
		result := make([]any, 0)
		for m != nil && (limit < 0 || len(result) < limit) {
			if err := matcherIteration(len(result)); err != nil {
				return nil, err
			}
			obj := evaluator.NewOrderedMap()
			obj.Set("match", m.text)
			obj.Set("index", float64(m.start))
			obj.Set("groups", m.groups)
			result = append(result, obj)
			m, err = evaluateMatcher(m.next, nil, s, evalFn, env)
			if err != nil {
				return nil, err
			}
		}
		return matchResultSeq(result), nil
	}
}

func makeFnContains(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, focus any, env *evaluator.Environment) (any, error) {
		if len(args) == 1 && focus != nil {
			args = []any{focus, args[0]}
		}
		s, ok, err := matcherString(args)
		if err != nil || !ok {
			return nil, err
		}
		if len(args) < 2 || args[1] == nil {
			return nil, nil
		}
		if token, ok := args[1].(string); ok {
			i, err := jstring.IndexUnits(env.Context(), jstring.Units(s), jstring.Units(token), 0)
			return i >= 0, err
		}
		m, err := evaluateMatcher(args[1], []any{s}, s, evalFn, env)
		return m != nil, err
	}
}

func makeFnSplit(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, _ any, env *evaluator.Environment) (any, error) {
		s, ok, err := matcherString(args)
		if err != nil || !ok {
			return nil, err
		}
		if len(args) < 2 {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "separator required"}
		}
		if args[1] == nil {
			return nil, nil
		}
		limit, err := matcherLimit(args, 2, "D3020")
		if err != nil {
			return nil, err
		}
		result := make([]any, 0)
		if limit == 0 {
			return result, nil
		}
		units := jstring.Units(s)
		if token, ok := args[1].(string); ok {
			if len(args) > 2 && args[2] != nil {
				limit = numeric.ClippedTrunc(args[2], 0, len(units)+1)
			}
			if limit == 0 {
				return result, nil
			}
			sep := jstring.Units(token)
			if len(sep) == 0 {
				for _, u := range units {
					if limit >= 0 && len(result) >= limit {
						break
					}
					result = append(result, jstring.CodeUnit(u))
				}
				return result, nil
			}
			start := 0
			for limit < 0 || len(result) < limit {
				if err := matcherIteration(len(result)); err != nil {
					return nil, err
				}
				i, err := jstring.IndexUnits(env.Context(), units, sep, start)
				if err != nil {
					return nil, err
				}
				if i < 0 {
					result = append(result, jstring.FromUnits(units[start:]))
					break
				}
				result = append(result, jstring.FromUnits(units[start:i]))
				start = i + len(sep)
			}
			return result, nil
		}
		m, err := evaluateMatcher(args[1], []any{s}, s, evalFn, env)
		if err != nil {
			return nil, err
		}
		start := 0
		for m != nil && (limit < 0 || len(result) < limit) {
			if err := matcherIteration(len(result)); err != nil {
				return nil, err
			}
			if m.start < start {
				return nil, matcherError()
			}
			result = append(result, jstring.FromUnits(units[start:m.start]))
			start = m.end
			m, err = evaluateMatcher(m.next, nil, s, evalFn, env)
			if err != nil {
				return nil, err
			}
		}
		if limit < 0 || len(result) < limit {
			result = append(result, jstring.FromUnits(units[start:]))
		}
		return result, nil
	}
}

func makeFnReplace(evalFn EvalFn) evaluator.EnvAwareBuiltin {
	return func(args []any, focus any, env *evaluator.Environment) (any, error) {
		s, ok, err := matcherString(args)
		if err != nil || !ok {
			return nil, err
		}
		if len(args) < 3 {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "pattern and replacement required"}
		}
		pattern := args[1]
		if pattern == nil {
			return nil, nil
		}
		if pattern == "" {
			return nil, &evaluator.JSONataError{Code: "D3010", Message: "empty replacement pattern"}
		}
		limit, err := matcherLimit(args, 3, "D3011")
		if err != nil {
			return nil, err
		}
		if limit == 0 {
			return s, nil
		}
		if literal, ok := pattern.(string); ok {
			if replacement, ok := args[2].(string); ok {
				return replaceLiteralUnits(s, literal, replacement, limit, env)
			}
			pattern = &evaluator.RegexValue{Pattern: ecmaregex.Quote(literal)}
		}
		if !callable(pattern) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "pattern must be a string or matcher"}
		}
		replacement, isTemplate := args[2].(string)
		if !isTemplate && !callable(args[2]) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "replacement must be string or function"}
		}
		m, err := evaluateMatcher(pattern, []any{s}, s, evalFn, env)
		if err != nil {
			return nil, err
		}
		units := jstring.Units(s)
		parts := make([]string, 0)
		position, count := 0, 0
		for m != nil && (limit < 0 || count < limit) {
			if err := matcherIteration(count); err != nil {
				return nil, err
			}
			if m.start < position {
				return nil, matcherError()
			}
			parts = append(parts, jstring.FromUnits(units[position:m.start]))
			var text string
			if isTemplate {
				groups, _ := evaluator.ArrayValues(m.groups)
				stringsOnly := make([]string, len(groups))
				for i, g := range groups {
					stringsOnly[i], _ = g.(string)
				}
				text = expandJSONataReplacement(replacement, m.text, stringsOnly)
			} else {
				v, err := evalFn(args[2], []any{m.value}, focus, env)
				if err != nil {
					return nil, err
				}
				var ok bool
				text, ok = v.(string)
				if !ok {
					return nil, &evaluator.JSONataError{Code: "D3012", Message: "replacement function must return a string"}
				}
			}
			parts = append(parts, text)
			position = m.start + len(jstring.Units(m.text))
			if position > len(units) {
				return nil, matcherError()
			}
			count++
			m, err = evaluateMatcher(m.next, nil, s, evalFn, env)
			if err != nil {
				return nil, err
			}
		}
		parts = append(parts, jstring.FromUnits(units[position:]))
		return jstring.Join(parts, ""), nil
	}
}

func replaceLiteralUnits(s, pattern, replacement string, limit int, env *evaluator.Environment) (any, error) {
	units, token := jstring.Units(s), jstring.Units(pattern)
	parts := make([]string, 0)
	start, count := 0, 0
	for limit < 0 || count < limit {
		if err := env.Context().Err(); err != nil {
			return nil, err
		}
		i, err := jstring.IndexUnits(env.Context(), units, token, start)
		if err != nil {
			return nil, err
		}
		if i < 0 {
			break
		}
		if err := matcherIteration(count); err != nil {
			return nil, err
		}
		parts = append(parts, jstring.FromUnits(units[start:i]), replacement)
		start = i + len(token)
		count++
	}
	parts = append(parts, jstring.FromUnits(units[start:]))
	return jstring.Join(parts, ""), nil
}

// expandJSONataReplacement scans only the possible group-number width. It
// cannot overflow through a caller-supplied run of replacement digits.
func expandJSONataReplacement(repl, full string, groups []string) string {
	var b strings.Builder
	width := len(strconv.Itoa(len(groups)))
	for i := 0; i < len(repl); {
		if repl[i] != '$' {
			b.WriteByte(repl[i])
			i++
			continue
		}
		i++
		if i >= len(repl) {
			b.WriteByte('$')
			break
		}
		if repl[i] == '$' {
			b.WriteByte('$')
			i++
			continue
		}
		if repl[i] == '0' {
			b.WriteString(full)
			i++
			continue
		}
		n, digits := 0, 0
		for digits < width && i+digits < len(repl) && repl[i+digits] >= '0' && repl[i+digits] <= '9' {
			n = n*10 + int(repl[i+digits]-'0')
			digits++
		}
		if digits == 0 {
			b.WriteByte('$')
			continue
		}
		if width > 1 && n > len(groups) && digits > 1 {
			n /= 10
			digits--
		}
		if n > 0 && n <= len(groups) {
			b.WriteString(groups[n-1])
		}
		i += digits
	}
	return b.String()
}
