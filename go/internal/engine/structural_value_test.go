package gnata_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestStructuralValueModel(t *testing.T) {
	raw, err := os.ReadFile("testdata/official-value-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct{ ID, Expr, InputJSON, ResultJSON string }
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 53 {
		t.Fatalf("case inventory changed: %d", len(fixture.Cases))
	}
	for _, c := range fixture.Cases {
		t.Run(c.ID, func(t *testing.T) {
			e, err := gnata.Compile(c.Expr)
			if err != nil {
				t.Fatal(err)
			}
			data, err := gnata.DecodeJSON([]byte(c.InputJSON))
			if err != nil {
				t.Fatal(err)
			}
			want, err := gnata.DecodeJSON([]byte(c.ResultJSON))
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"full", "bytes", "vars"} {
				t.Run(mode, func(t *testing.T) {
					var got any
					var err error
					switch mode {
					case "full":
						got, err = e.Eval(context.Background(), data)
					case "bytes":
						got, err = e.EvalBytes(context.Background(), []byte(c.InputJSON))
					case "vars":
						got, err = e.EvalBytesWithVars(context.Background(), []byte(c.InputJSON), nil)
					}
					if err != nil || !gnata.DeepEqual(got, want) {
						t.Errorf("%s: got %#v; want %s; error %v", c.Expr, got, c.ResultJSON, err)
					}
				})
			}
		})
	}
}

func TestNativeAndReusedResultIdentity(t *testing.T) {
	identity, err := gnata.Compile(`$`)
	if err != nil {
		t.Fatal(err)
	}
	expr, err := gnata.Compile(`{"selected":arr[0] in arr,"copy":$clone(arr)[0] in arr,"types":$map(values,$type)}`)
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"arr": []any{[]any{}, []any{}}, "values": []any{nil, false, float64(0), "", []any{}, map[string]any{}}}
	want, err := gnata.DecodeJSON([]byte(`{"selected":true,"copy":false,"types":["null","boolean","number","string","array","object"]}`))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		var current any = input
		if i > 0 {
			current, err = identity.Eval(context.Background(), current)
			if err != nil {
				t.Fatal(err)
			}
		}
		if i > 1 {
			current = gnata.NormalizeValue(current)
		}
		got, err := expr.Eval(context.Background(), current)
		if err != nil || !gnata.DeepEqual(got, want) {
			t.Fatalf("boundary %d got %v; error %v", i, got, err)
		}
	}
}
