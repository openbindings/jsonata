package evaluator

import (
	"encoding/json"
	"math"
	"slices"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/parser"
)

// Null is the singleton JSONata null value.
var Null any = jsonNullType{}

// JSONNull is a sentinel type that represents JSON null explicitly,
// distinguishing it from Go nil (which represents JSONata undefined).
type jsonNullType struct{}

func (jsonNullType) MarshalJSON() ([]byte, error) { return []byte(parser.NullJSON), nil }

// IsNull reports whether v is the JSON null sentinel.
func IsNull(v any) bool {
	_, ok := v.(jsonNullType)
	return ok
}

// Sequence is the evaluator's identity-bearing list container. ConsArray
// distinguishes a JSON array from a query sequence. Both use the same storage,
// but only sequences participate in singleton/empty sequence collapse.
type Sequence struct {
	Values        []any
	KeepSingleton bool // do NOT unwrap single-element sequences
	ConsArray     bool // JSON array kind (input or constructed), not a query sequence
	OuterWrapper  bool // input was a JSON array; treated as a single document
	TupleStream   bool // contains tuple objects {"@": value, varName: value}
}

// CreateSequence creates a Sequence optionally pre-populated with one value.
func CreateSequence(items ...any) *Sequence {
	s := &Sequence{Values: make([]any, 0, len(items)+4)}
	s.Values = append(s.Values, items...)
	return s
}

// IsSequence reports whether v is a *Sequence.
func IsSequence(v any) bool {
	s, ok := v.(*Sequence)
	return ok && !s.ConsArray
}

// CollapseSequence applies JSONata singleton-collapsing rules internally:
//   - len 0 → nil (undefined)
//   - len 1 → elem[0] unless KeepSingleton is set
//   - len > 1 → same sequence, preserving its kind through variables/functions
//   - JSON arrays always retain their container and identity
func CollapseSequence(s *Sequence) any {
	if s.ConsArray {
		return s
	}
	switch len(s.Values) {
	case 0:
		return nil
	case 1:
		if s.KeepSingleton {
			return s
		}
		return s.Values[0]
	default:
		return s
	}
}

// CollapseAndKeep normalizes a function call result for callers that need
// KeepArray (the [] suffix) support. Builtins returning *Sequence rely on
// CollapseSequence; when keepArray is true, singletons are preserved as
// one-element arrays instead of being unwrapped.
func CollapseAndKeep(result any, keepArray bool) any {
	if seq, ok := result.(*Sequence); ok {
		if keepArray {
			s := *seq
			s.KeepSingleton = true
			seq = &s
		}
		result = CollapseSequence(seq)
	}
	if keepArray {
		switch result.(type) {
		case []any, *Sequence:
			return result
		case nil:
			return nil
		default:
			return []any{result}
		}
	}
	return result
}

// IsArray distinguishes the JSON array kind from a query sequence.
func IsArray(v any) bool {
	if seq, ok := v.(*Sequence); ok {
		return seq.ConsArray
	}
	_, ok := v.([]any)
	return ok
}

// CollapseToSlice returns the sequence values as a plain []any slice.
// Ownership transfer; callers must not mutate the returned slice.
func CollapseToSlice(s *Sequence) []any {
	return slices.Clip(s.Values)
}

// IsNumeric returns true for finite numeric values (float64 or json.Number).
func IsNumeric(v any) bool {
	_, ok := numeric.Text(v)
	return ok
}

// CheckNumeric validates that v is a finite numeric value. Returns a D1001 error for Inf/NaN.
func CheckNumeric(v any) error {
	switch n := v.(type) {
	case float64:
		if math.IsInf(n, 0) || math.IsNaN(n) {
			return &JSONataError{Code: "D1001", Value: v}
		}
	case json.Number:
		if !numeric.Valid(string(n)) {
			return &JSONataError{Code: "D1001", Value: v}
		}
	}
	return nil
}

// ToBoolean implements JSONata boolean casting rules.
func ToBoolean(v any) bool {
	if v == nil || IsNull(v) {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case float64:
		return val != 0
	case json.Number:
		return IsNumeric(val) && !numeric.Zero(val)
	case *OrderedMap:
		return val.Len() > 0
	case map[string]any:
		return len(val) > 0
	case []any:
		switch len(val) {
		case 0:
			return false
		case 1:
			return ToBoolean(val[0])
		default:
			return slices.ContainsFunc(val, ToBoolean)
		}
	case *Sequence:
		return ToBoolean(val.Values)
	}
	return false
}

// DeepEqual implements JSONata structural equality.
func DeepEqual(a, b any) bool { //nolint:gocyclo // type-switch fast path adds branches but not real complexity
	switch v := a.(type) {
	case *NativeFunction:
		other, ok := b.(*NativeFunction)
		return ok && v == other
	case *SignedBuiltin:
		other, ok := b.(*SignedBuiltin)
		return ok && v == other
	case *Lambda:
		other, ok := b.(*Lambda)
		return ok && v == other
	case *RegexValue:
		other, ok := b.(*RegexValue)
		return ok && v == other
	}
	if av, ok := ArrayValues(a); ok {
		bv, ok := ArrayValues(b)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !DeepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	}
	if IsNumeric(a) {
		return IsNumeric(b) && numeric.Compare(a, b) == 0
	}
	switch av := a.(type) {
	case float64:
		if bv, ok := b.(float64); ok {
			return av == bv
		}
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	}

	if a == nil || b == nil || IsNull(a) || IsNull(b) {
		return a == nil && b == nil || IsNull(a) && IsNull(b)
	}
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []any:
		bv, ok := ArrayValues(b)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !DeepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		if !IsMap(b) || MapLen(b) != len(av) {
			return false
		}
		for k, va := range av {
			vb, exists := MapGet(b, k)
			if !exists || !DeepEqual(va, vb) {
				return false
			}
		}
		return true
	case *OrderedMap:
		if !IsMap(b) || MapLen(b) != av.Len() {
			return false
		}
		equal := true
		av.Range(func(k string, va any) bool {
			vb, exists := MapGet(b, k)
			if !exists || !DeepEqual(va, vb) {
				equal = false
				return false
			}
			return true
		})
		return equal
	}
	return false
}

// JSONataError is the structured error type used throughout evaluation.
// Code matches the JSONata spec error codes (S0xxx, T0xxx, T1xxx, T2xxx, D1xxx, D2xxx, D3xxx).
type JSONataError struct {
	Code    string
	Token   string
	Value   any
	Message string
}

func (e *JSONataError) Error() string {
	if e.Message != "" && e.Code != "" {
		return e.Code + ": " + e.Message
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}
