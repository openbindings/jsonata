package gnata

import (
	"fmt"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/jstring"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

// JSONValue detaches an evaluator result into JSON-compatible Go containers,
// retaining exact numbers and string code units. Unlike generic serialization,
// it rejects functions, undefined members, host objects and cyclic/deep values
// before a serializer can disguise them as JSON. An undefined root is an error;
// JSONata null becomes Go nil. It does not invoke user serialization methods.
func JSONValue(value any) (out any, err error) {
	remaining := 1000000
	var walk func(any, int) (any, error)
	walk = func(value any, depth int) (any, error) {
		remaining--
		if depth > 512 || remaining < 0 {
			return nil, fmt.Errorf("JSONata result exceeds value work budget")
		}
		if evaluator.IsNull(value) {
			return nil, nil
		}
		if _, ok := numeric.Text(value); ok {
			return value, nil
		}
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			if !jstring.Valid(v) {
				return nil, fmt.Errorf("JSONata result contains malformed string encoding")
			}
			return v, nil
		case *evaluator.OrderedMap:
			out := make(map[string]any, v.Len())
			var err error
			v.Range(func(k string, item any) bool {
				if !jstring.Valid(k) {
					err = fmt.Errorf("JSONata result contains malformed member name")
					return false
				}
				out[k], err = walk(item, depth+1)
				return err == nil
			})
			return out, err
		case []any:
			out := make([]any, len(v))
			for i, item := range v {
				var err error
				out[i], err = walk(item, depth+1)
				if err != nil {
					return nil, err
				}
			}
			return out, nil
		default:
			return nil, fmt.Errorf("JSONata result is not a JSON value (%T)", value)
		}
	}
	return walk(value, 0)
}
