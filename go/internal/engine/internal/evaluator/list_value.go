package evaluator

// NewArray gives every language array its own container identity, even when
// empty. Sequence already owns list storage; ConsArray distinguishes the JSON
// array kind from query sequences. Identity never comes from a backing-slice
// address, capacity, contents, or the syntax of a function's argument.
func NewArray(values []any) *Sequence {
	if values == nil {
		values = []any{}
	}
	return &Sequence{Values: values, ConsArray: true}
}

// ArrayValues is a read-only view of array/sequence storage, not a conversion of
// a language value. Retain the original container when an operation selects it.
func ArrayValues(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		return v, true // host/constructor storage before admission
	case *Sequence:
		return v.Values, true
	default:
		return nil, false
	}
}

// InternalizeValue admits host JSON tree containers. Existing evaluator-owned
// containers retain identity; no caller-owned map or slice is modified.
func InternalizeValue(value any) any {
	// Retain the existing ten-million-element range domain plus container
	// ancestors. This is a materialization guard, not a range-language change.
	remaining := 10000000 + 513
	return internalizeValue(value, 0, &remaining)
}

func checkValueWalk(depth int, remaining *int) {
	*remaining--
	if depth > 512 || *remaining < 0 {
		panic(&JSONataError{Code: "U_VALUE_LIMIT", Message: "value nesting or materialization budget exceeded"})
	}
}

func internalizeValue(value any, depth int, remaining *int) any {
	checkValueWalk(depth, remaining)
	switch v := value.(type) {
	case BuiltinFunction, EnvAwareBuiltin:
		return NewNativeFunction(v, 1)
	case *OrderedMap:
		if !v.external {
			return v
		}
		out := NewOrderedMapWithCapacity(v.Len())
		v.Range(func(k string, item any) bool { out.Set(k, internalizeValue(item, depth+1, remaining)); return true })
		return out
	case []any:
		values := make([]any, len(v))
		for i, item := range v {
			if item == nil {
				item = Null
			}
			values[i] = internalizeValue(item, depth+1, remaining)
		}
		return NewArray(values)
	case map[string]any:
		out := NewOrderedMapWithCapacity(len(v))
		for _, key := range MapKeys(v) {
			item := v[key]
			if item == nil {
				item = Null
			}
			out.Set(key, internalizeValue(item, depth+1, remaining))
		}
		return out
	default:
		return value
	}
}

// ExternalizeLists keeps the existing Go result surface (plain slices plus
// ordered objects). It is used only at the evaluator boundary, never between
// language operations. Function and null values remain distinguishable.
func ExternalizeLists(value any) any {
	remaining := 10000000 + 513
	return externalizeLists(value, 0, &remaining)
}

func externalizeLists(value any, depth int, remaining *int) any {
	checkValueWalk(depth, remaining)
	switch v := value.(type) {
	case *Sequence:
		if !v.ConsArray && !v.KeepSingleton {
			if len(v.Values) == 0 {
				return nil
			}
			if len(v.Values) == 1 {
				return externalizeLists(v.Values[0], depth+1, remaining)
			}
		}
		out := make([]any, len(v.Values))
		for i, item := range v.Values {
			out[i] = externalizeLists(item, depth+1, remaining)
		}
		return out
	case *OrderedMap:
		out := NewOrderedMapWithCapacity(v.Len())
		out.external = true
		v.Range(func(k string, item any) bool {
			out.keys = append(out.keys, k)
			out.data[k] = externalizeLists(item, depth+1, remaining)
			return true
		})
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = externalizeLists(item, depth+1, remaining)
		}
		return out
	default:
		return value
	}
}

func (s *Sequence) MarshalJSON() ([]byte, error) {
	return marshalNoHTMLEscape(ExternalizeLists(s))
}

func sameValue(a, b any) bool {
	if a == nil || b == nil {
		return false
	}
	if aList, ok := a.(*Sequence); ok {
		bList, ok := b.(*Sequence)
		return ok && aList == bList
	}
	if aMap, ok := a.(*OrderedMap); ok {
		bMap, ok := b.(*OrderedMap)
		return ok && aMap == bMap
	}
	return DeepEqual(a, b)
}
