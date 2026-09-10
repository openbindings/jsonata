package evaluator

import (
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/parser"
)

func deepClone(v any) any {
	switch val := v.(type) {
	case *Sequence:
		values := make([]any, len(val.Values))
		for i, item := range val.Values {
			values[i] = deepClone(item)
		}
		return NewArray(values)
	case *OrderedMap:
		m := NewOrderedMapWithCapacity(val.Len())
		val.Range(func(k string, vv any) bool {
			m.Set(k, deepClone(vv))
			return true
		})
		return m
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, vv := range val {
			m[k] = deepClone(vv)
		}
		return m
	case []any:
		s := make([]any, len(val))
		for i, vv := range val {
			s[i] = deepClone(vv)
		}
		return s
	case BuiltinFunction, EnvAwareBuiltin, *Lambda, *SignedBuiltin, *NativeFunction, *RegexValue:
		// Match the compatibility clone helper's intermediate function formatting,
		// without passing numeric values through display stringification.
		return ""
	default:
		return v
	}
}

// CloneArgument implements the reference $clone compatibility helper. No numeric conversion
// is needed to detach JSON objects and arrays.
func CloneArgument(args []any, focus any) (any, error) {
	value := focus
	if len(args) > 1 {
		return nil, &JSONataError{Code: "T0410", Message: "$clone: expected one argument"}
	}
	if len(args) == 1 {
		value = args[0]
	}
	if value == nil {
		return nil, nil
	}
	_, isArray := ArrayValues(value)
	if !IsMap(value) && !isArray {
		code := "T0410"
		if len(args) == 0 {
			code = "T0411"
		}
		return nil, &JSONataError{Code: code, Message: "$clone: expected an object or array"}
	}
	return deepClone(value), nil
}

func evalTransform(node *parser.Node, _ any, env *Environment) (any, error) {
	return BuiltinFunction(func(args []any, focus any) (any, error) {
		var doc any
		if len(args) > 0 {
			doc = args[0]
		} else {
			doc = focus
		}
		return applyTransform(node, doc, env)
	}), nil
}

var (
	errTransformUpdate = JSONataError{
		Code: "T2011", Message: "the insert/update clause of the transform expression must evaluate to an object",
	}
	errTransformDelete = JSONataError{
		Code: "T2012", Message: "the delete clause of the transform expression must evaluate to an array of strings",
	}
)

func applyTransform(node *parser.Node, input any, env *Environment) (any, error) {
	if input == nil {
		return nil, nil
	}
	clone, _ := env.Lookup("clone")
	if env.StandardLibraryOnly() {
		clone = BuiltinFunction(CloneArgument)
	}
	switch clone.(type) {
	case BuiltinFunction, EnvAwareBuiltin, *Lambda, *SignedBuiltin, *NativeFunction, *RegexValue:
	default:
		return nil, &JSONataError{Code: "T2013", Message: "the transform expression requires $clone to be a function"}
	}
	cloned, err := callFunction(clone, []any{input}, nil, env)
	if err != nil {
		return nil, err
	}

	matched, err := Eval(node.Pattern, cloned, env)
	if err != nil {
		return nil, err
	}

	var targets []any
	if values, ok := ArrayValues(matched); ok {
		matched = values
	}
	switch m := matched.(type) {
	case *OrderedMap:
		targets = []any{m}
	case map[string]any:
		targets = []any{m}
	case []any:
		for _, item := range m {
			if IsMap(item) {
				targets = append(targets, item)
			}
		}
	}

	if len(targets) == 0 && matched != nil {
		// Pattern matched a non-object value: validate update/delete types
		// but don't mutate (original jsonata-js behavior for non-object patterns).
		return cloned, validateTransformClauses(node, cloned, env)
	}

	for _, target := range targets {
		if err := applyTransformTarget(node, target, env); err != nil {
			return nil, err
		}
	}
	return cloned, nil
}

// validateTransformClauses evaluates update/delete clauses for type-checking only,
// without mutating the target. Used when the pattern matches a non-object value.
func validateTransformClauses(node *parser.Node, target any, env *Environment) error {
	if node.Update != nil {
		if updateVal, err := Eval(node.Update, target, env); err != nil {
			return err
		} else if updateVal != nil && !IsNull(updateVal) && !IsMap(updateVal) {
			return &errTransformUpdate
		}
	}
	if node.Delete != nil {
		if deleteVal, err := Eval(node.Delete, target, env); err != nil {
			return err
		} else if deleteVal != nil && !IsNull(deleteVal) {
			switch deleteVal.(type) {
			case []any, *Sequence, string:
			default:
				return &errTransformDelete
			}
		}
	}
	return nil
}

func applyTransformTarget(node *parser.Node, target any, env *Environment) error {
	if node.Update != nil {
		if updateVal, err := Eval(node.Update, target, env); err != nil {
			return err
		} else if updateVal != nil && !IsNull(updateVal) {
			if !IsMap(updateVal) {
				return &errTransformUpdate
			}
			transformMerge(target, updateVal)
		}
	}
	if node.Delete != nil {
		if deleteVal, err := Eval(node.Delete, target, env); err != nil {
			return err
		} else if deleteVal != nil && !IsNull(deleteVal) {
			if values, ok := ArrayValues(deleteVal); ok {
				deleteVal = values
			}
			switch dv := deleteVal.(type) {
			case []any:
				for _, f := range dv {
					if s, ok := f.(string); ok {
						transformDelete(target, s)
					}
				}
			case string:
				transformDelete(target, dv)
			default:
				return &errTransformDelete
			}
		}
	}
	return nil
}

func transformMerge(target, update any) {
	MapRange(update, func(k string, v any) bool {
		switch t := target.(type) {
		case *OrderedMap:
			t.Set(k, v)
		case map[string]any:
			t[k] = v
		}
		return true
	})
}

func transformDelete(target any, key string) {
	switch t := target.(type) {
	case *OrderedMap:
		t.Delete(key)
	case map[string]any:
		delete(t, key)
	}
}
