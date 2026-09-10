package gnata_test

import (
	"context"
	"encoding/json"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestNumericCarriageAllEntryPaths(t *testing.T) {
	for _, token := range []string{"9007199254740993.0", "0.10000000000000000001", "1e-1000", "1e1000", "-0", "1e999999999999999999999"} {
		for _, expression := range []string{"n", "nested.n", "$reverse(values)[0]"} {
			t.Run(token+"/"+expression, func(t *testing.T) {
				raw := []byte(`{"n":` + token + `,"nested":{"n":` + token + `},"values":[` + token + `]}`)
				expr, err := gnata.Compile(expression)
				if err != nil {
					t.Fatal(err)
				}
				data, err := gnata.DecodeJSON(raw)
				if err != nil {
					t.Fatal(err)
				}
				var mapped map[string]json.RawMessage
				if err := json.Unmarshal(raw, &mapped); err != nil {
					t.Fatal(err)
				}
				stream := gnata.NewStreamEvaluator([]*gnata.Expression{expr})
				paths := map[string]func() (any, error){
					"full":   func() (any, error) { return expr.Eval(context.Background(), data) },
					"bytes":  func() (any, error) { return expr.EvalBytes(context.Background(), raw) },
					"map":    func() (any, error) { return expr.EvalMap(context.Background(), mapped) },
					"vars":   func() (any, error) { return expr.EvalBytesWithVars(context.Background(), raw, nil) },
					"stream": func() (any, error) { return stream.EvalOne(context.Background(), raw, "exact-carriage", 0) },
				}
				for name, run := range paths {
					t.Run(name, func(t *testing.T) {
						got, err := run()
						if err != nil || !gnata.DeepEqual(got, json.Number(token)) {
							t.Fatalf("got %v (%T), want %s; error %v", got, got, token, err)
						}
					})
				}
			})
		}
	}
}
