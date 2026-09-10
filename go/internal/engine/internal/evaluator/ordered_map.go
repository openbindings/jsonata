package evaluator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/openbindings/jsonata/go/internal/engine/internal/jstring"
)

// OrderedMap is a map that preserves insertion order for JSON serialization.
// Go has no stdlib ordered map; this is the minimal implementation needed so
// that objects created by JSONata expressions serialize keys in definition order.
// Input data from json.Unmarshal remains plain map[string]any — the helper
// functions (MapGet, MapKeys, etc.) bridge both types at call sites.
type OrderedMap struct {
	external bool // boundary snapshot; native nested lists require readmission
	keys     []string
	data     map[string]any
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{data: make(map[string]any)}
}

func NewOrderedMapWithCapacity(n int) *OrderedMap {
	return &OrderedMap{
		keys: make([]string, 0, n),
		data: make(map[string]any, n),
	}
}

func (m *OrderedMap) Set(key string, val any) {
	val = InternalizeValue(val)
	if _, exists := m.data[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.data[key] = val
}

func (m *OrderedMap) Get(key string) (any, bool) {
	v, ok := m.data[key]
	return v, ok
}

func (m *OrderedMap) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *OrderedMap) Delete(key string) {
	if _, ok := m.data[key]; !ok {
		return
	}
	delete(m.data, key)
	m.keys = slices.DeleteFunc(m.keys, func(k string) bool { return k == key })
}

func (m *OrderedMap) Keys() []string        { return m.keys }
func (m *OrderedMap) Len() int              { return len(m.keys) }
func (m *OrderedMap) ToMap() map[string]any { return m.data }

// Range calls fn for each entry in insertion order.
func (m *OrderedMap) Range(fn func(key string, val any) bool) {
	for _, k := range m.keys {
		if !fn(k, m.data[k]) {
			break
		}
	}
}

// MarshalJSON implements json.Marshaler, preserving insertion order.
// json.MarshalIndent calls this then re-indents, so no separate indent method needed.
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := marshalNoHTMLEscape(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := marshalNoHTMLEscape(m.data[k])
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// marshalNoHTMLEscape serializes v to JSON without escaping &, <, >.
func marshalNoHTMLEscape(v any) ([]byte, error) {
	v, err := PrepareJSONStrings(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// PrepareJSONStrings supplies encoding/json with explicit JSON escapes for
// admitted isolated code units, including object keys. It leaves language
// value normalization (functions, sequences, null) to its existing callers.
func PrepareJSONStrings(v any) (any, error) {
	if seq, ok := v.(*Sequence); ok {
		return PrepareJSONStrings(ExternalizeLists(seq))
	}
	switch value := v.(type) {
	case string:
		if utf8.ValidString(value) {
			return value, nil
		}
		b, err := jstring.Quote(value)
		return json.RawMessage(b), err
	case []any:
		out := make([]any, len(value))
		for i, child := range value {
			var err error
			out[i], err = PrepareJSONStrings(child)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case map[string]any:
		out := NewOrderedMapWithCapacity(len(value))
		for _, key := range MapKeys(value) {
			out.Set(key, value[key])
		}
		return out, nil
	}
	return v, nil
}

// UnmarshalJSON implements json.Unmarshaler, preserving key order.
func (m *OrderedMap) UnmarshalJSON(b []byte) error {
	v, err := DecodeJSON(b)
	if err != nil {
		return err
	}
	object, ok := v.(*OrderedMap)
	if !ok {
		return fmt.Errorf("expected JSON object")
	}
	*m = *object
	return nil
}

// DecodeJSON decodes a JSON value, using *OrderedMap for objects to preserve
// key insertion order. Arrays, strings, numbers, booleans, and null are
// decoded normally. This should be used instead of json.Unmarshal when key
// order matters (which is always the case for JSONata evaluation).
func DecodeJSON(b json.RawMessage) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	v, err := decodeValue(dec, b)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("JSON contains more than one value")
	}
	return v, nil
}

// DecodeRawMap converts a map of field names to raw JSON values into an
// *OrderedMap by decoding each value individually via DecodeJSON. Objects
// in the values preserve key insertion order, consistent with DecodeJSON.
func DecodeRawMap(m map[string]json.RawMessage) (*OrderedMap, error) {
	om := NewOrderedMapWithCapacity(len(m))
	for _, key := range slices.Sorted(maps.Keys(m)) {
		val, err := DecodeJSON(m[key])
		if err != nil {
			return nil, fmt.Errorf("decode key %q: %w", key, err)
		}
		om.Set(key, val)
	}
	return om, nil
}

func decodeValue(dec *json.Decoder, source []byte) (any, error) {
	tok, err := jstring.Token(dec, source)
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return decodeObject(dec, source)
		case '[':
			return decodeArray(dec, source)
		}
	case json.Number:
		return t, nil
	case string:
		return t, nil
	case bool:
		return t, nil
	case nil:
		return Null, nil
	}
	return tok, nil
}

func decodeObject(dec *json.Decoder, source []byte) (*OrderedMap, error) {
	m := NewOrderedMap()
	for dec.More() {
		keyTok, err := jstring.Token(dec, source)
		if err != nil {
			return nil, err
		}
		key := keyTok.(string)
		val, err := decodeValue(dec, source)
		if err != nil {
			return nil, err
		}
		m.Set(key, val)
	}
	// Consume closing '}'
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return m, nil
}

func decodeArray(dec *json.Decoder, source []byte) (*Sequence, error) {
	var arr []any
	for dec.More() {
		val, err := decodeValue(dec, source)
		if err != nil {
			return nil, err
		}
		arr = append(arr, val)
	}
	// Consume closing ']'
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if arr == nil {
		arr = []any{}
	}
	return NewArray(arr), nil
}

// ── Helpers for dual map[string]any / *OrderedMap handling ───────────────────

func MapGet(obj any, key string) (any, bool) {
	switch m := obj.(type) {
	case *OrderedMap:
		return m.Get(key)
	case map[string]any:
		v, ok := m[key]
		return v, ok
	}
	return nil, false
}

func MapKeys(obj any) []string {
	switch m := obj.(type) {
	case *OrderedMap:
		return m.Keys()
	case map[string]any:
		return slices.Sorted(maps.Keys(m))
	}
	return nil
}

func MapLen(obj any) int {
	switch m := obj.(type) {
	case *OrderedMap:
		return m.Len()
	case map[string]any:
		return len(m)
	}
	return 0
}

func IsMap(obj any) bool {
	switch obj.(type) {
	case *OrderedMap, map[string]any:
		return true
	}
	return false
}

func MapRange(obj any, fn func(key string, val any) bool) {
	switch m := obj.(type) {
	case *OrderedMap:
		m.Range(fn)
	case map[string]any:
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !fn(k, m[k]) {
				break
			}
		}
	}
}
