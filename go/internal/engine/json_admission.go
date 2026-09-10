package gnata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/jstring"
	"github.com/tidwall/gjson"
)

func decodeJSONBindings(input []byte) (map[string]any, error) {
	if input == nil {
		return nil, nil
	}
	value, err := DecodeJSON(input)
	if err != nil {
		return nil, fmt.Errorf("JSONata bindings: %w", err)
	}
	object, ok := value.(*evaluator.OrderedMap)
	if !ok {
		return nil, fmt.Errorf("JSONata bindings must be a JSON object")
	}
	bound := object.ToMap()
	for name := range bound {
		if strings.HasPrefix(name, "$") {
			return nil, fmt.Errorf("JSONata binding names must omit the dollar prefix")
		}
	}
	return bound, nil
}

// validateJSONSelection uses the standard library for complete grammar validation
// and the existing selector for container traversal only. Numeric conversions in
// gjson.Result are never used as values; selected numbers retain their raw tokens.
// Names are scoped to each object and decoded with the same string policy as the
// full evaluator. Neither this walk nor its input escapes into the compile cache.
func validateJSONSelection(ctx context.Context, input []byte) error {
	if !json.Valid(input) {
		return fmt.Errorf("JSONata input must contain one complete JSON value")
	}
	var visit func(gjson.Result) error
	visit = func(value gjson.Result) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		object := value.IsObject()
		if !object && !value.IsArray() {
			return nil
		}
		var seen map[string]struct{}
		if object {
			seen = make(map[string]struct{})
		}
		var failure error
		value.ForEach(func(key, child gjson.Result) bool {
			if object {
				name := key.Str
				if strings.Contains(key.Raw, `\`) || !utf8.ValidString(name) {
					name, failure = jstring.Unquote([]byte(key.Raw))
					if failure != nil {
						return false
					}
				}
				if _, exists := seen[name]; exists {
					failure = fmt.Errorf("JSONata input contains a duplicate object member")
					return false
				}
				seen[name] = struct{}{}
			}
			failure = visit(child)
			return failure == nil
		})
		return failure
	}
	return visit(gjson.ParseBytes(input))
}

// evalJSONWithVars is the closed executor's admission boundary. General
// expressions decode once. Selectors can avoid materializing unrelated values,
// but never avoid validating their syntax and object-member uniqueness.
func (e *Expression) evalJSONWithVars(ctx context.Context, input []byte, vars map[string]any) (any, error) {
	if (e.fastPath || e.cmpFast != nil || e.funcFast != nil) && !needsStringDecode(input, nil) {
		if err := validateJSONSelection(ctx, input); err != nil {
			return nil, err
		}
		return e.EvalBytesWithVars(ctx, input, vars)
	}
	value, err := DecodeJSON(input)
	if err != nil {
		return nil, err
	}
	return e.EvalWithVars(ctx, value, vars)
}
