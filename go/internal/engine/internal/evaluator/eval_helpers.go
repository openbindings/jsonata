package evaluator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/parser"
)

func appendToSequence(seq *Sequence, v any) {
	if v == nil {
		return
	}
	switch val := v.(type) {
	case *Sequence:
		if val.ConsArray {
			seq.Values = append(seq.Values, val)
			return
		}
		for _, item := range val.Values {
			appendToSequence(seq, item)
		}
	default:
		seq.Values = append(seq.Values, v)
	}
}

func stringifyValue(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch val := v.(type) {
	case string:
		return val, nil
	case json.Number:
		return FormatNumber(val), nil
	case float64:
		return FormatFloat(val), nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	default:
		b, err := marshalNoHTMLEscape(v)
		if err != nil {
			return "", fmt.Errorf("cannot stringify value: %w", err)
		}
		return string(b), nil
	}
}

// FormatNumber formats the assigned decimal value without a binary64 detour.
func FormatNumber(n json.Number) string { return numeric.Format(string(n)) }

// FormatFloat uses the native value's JSON decimal image.
func FormatFloat(n float64) string {
	s, ok := numeric.Text(n)
	if !ok {
		return "null"
	}
	return numeric.Format(s)
}

func compareValues(left, right any, op string) (any, error) {
	leftNumber := IsNumeric(left)
	_, leftString := left.(string)
	if left != nil && !leftNumber && !leftString {
		return nil, &JSONataError{Code: "T2010", Message: "comparison requires numbers or strings"}
	}
	if left == nil || right == nil {
		return nil, nil
	}
	order, err := compareOrder(left, right)
	if err != nil {
		var e *JSONataError
		if errors.As(err, &e) && e.Code == "T2007" {
			return nil, &JSONataError{Code: "T2009", Message: e.Message}
		}
		return nil, &JSONataError{Code: "T2010", Message: err.Error()}
	}
	switch op {
	case "<":
		return order < 0, nil
	case "<=":
		return order <= 0, nil
	case ">":
		return order > 0, nil
	case ">=":
		return order >= 0, nil
	}
	return nil, fmt.Errorf("unknown comparison %q", op)
}

func compareOrder(a, b any) (int, error) {
	if a == nil && b == nil {
		return 0, nil
	}
	if a == nil {
		return 1, nil
	}
	if b == nil {
		return -1, nil
	}
	an, bn := IsNumeric(a), IsNumeric(b)
	if an && bn {
		return numeric.Compare(a, b), nil
	}
	as, astr := a.(string)
	bs, bstr := b.(string)
	if astr && bstr {
		return strings.Compare(as, bs), nil
	}
	if (an && bstr) || (astr && bn) {
		return 0, &JSONataError{Code: "T2007", Message: "cannot compare string and number values"}
	}
	return 0, &JSONataError{Code: "T2008", Message: fmt.Sprintf("cannot compare values of type %T and %T", a, b)}
}

func containsValue(arr, elem any) bool {
	if arr == nil {
		return false
	}
	switch v := arr.(type) {
	case []any:
		for _, item := range v {
			if sameValue(item, elem) {
				return true
			}
		}
	case *Sequence:
		for _, item := range v.Values {
			if sameValue(item, elem) {
				return true
			}
		}
	default:
		return sameValue(arr, elem)
	}
	return false
}

func evalValue(node *parser.Node) (any, error) {
	switch node.Value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case parser.NullJSON:
		return Null, nil
	default:
		return nil, nil
	}
}

func evalVariable(node *parser.Node, input any, env *Environment) (any, error) {
	if node.Value == "" {
		return input, nil
	}
	val, found := env.Lookup(node.Value)
	if !found {
		return nil, nil
	}
	return val, nil
}

func evalName(node *parser.Node, input any, _ *Environment) (any, error) {
	switch v := input.(type) {
	case *OrderedMap:
		val, ok := v.Get(node.Value)
		if !ok {
			return nil, nil
		}
		if val == nil {
			return Null, nil
		}
		return val, nil
	case map[string]any:
		val, ok := v[node.Value]
		if !ok {
			return nil, nil
		}
		if val == nil {
			return Null, nil
		}
		return val, nil
	case []any:
		// JSONata maps field lookups across arrays.
		// Per the JSONata spec, array results from each field lookup are
		// flattened into the result sequence (not nested).
		seq := CreateSequence()
		fieldFound := false
		for _, item := range v {
			val, err := evalName(node, item, nil)
			if err != nil {
				return nil, err
			}
			if val == nil {
				continue
			}
			fieldFound = true
			// Flatten plain []any results from navigating through arrays.
			// This matches JSONata's automatic flattening semantics.
			switch inner := val.(type) {
			case *Sequence:
				seq.Values = append(seq.Values, inner.Values...)
			case []any:
				for _, sv := range inner {
					if sv == nil {
						sv = Null
					}
					seq.Values = append(seq.Values, sv)
				}
			default:
				appendToSequence(seq, val)
			}
		}
		if len(seq.Values) == 0 {
			if fieldFound {
				// At least one element had this field defined (e.g. as an
				// empty array []). Return empty array rather than nil so
				// downstream $exists sees the field as present.
				return []any{}, nil
			}
			return nil, nil
		}
		if len(seq.Values) == 1 {
			return seq.Values[0], nil
		}
		return CollapseSequence(seq), nil
	case *Sequence:
		return evalName(node, v.Values, nil)
	default:
		return nil, nil
	}
}

func evalWildcard(_ *parser.Node, input any, _ *Environment) (any, error) {
	if IsMap(input) {
		if MapLen(input) == 0 {
			return nil, nil
		}
		seq := CreateSequence()
		MapRange(input, func(_ string, val any) bool {
			appendWildcardValue(seq, val)
			return true
		})
		if len(seq.Values) == 0 {
			return nil, nil
		}
		if len(seq.Values) == 1 {
			return seq.Values[0], nil
		}
		return CollapseSequence(seq), nil
	}
	switch v := input.(type) {
	case *Sequence:
		return evalWildcard(nil, v.Values, nil)
	case []any:
		seq := CreateSequence()
		for _, item := range v {
			// A directly selected array exposes its elements, not their fields.
			// The path mapper is responsible for applying .* to each context.
			appendWildcardValue(seq, item)
		}
		return CollapseSequence(seq), nil
	default:
		return nil, nil
	}
}

// Wildcard expansion flattens array-valued properties recursively. It must not
// turn each object's collapsed result sequence into a nested JSON array.
func appendWildcardValue(seq *Sequence, value any) {
	if arr, ok := ArrayValues(value); ok {
		for _, item := range arr {
			appendWildcardValue(seq, item)
		}
		return
	}
	seq.Values = append(seq.Values, value)
}
