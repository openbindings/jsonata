package gnata_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestStringSemantics(t *testing.T) {
	runValueCases(t, "testdata/official-string-cases.json", 72)
}

func TestStringOrdering(t *testing.T) {
	runValueCases(t, "testdata/official-string-order-cases.json", 20)
}

func TestObjectQuerySemantics(t *testing.T) {
	runValueCases(t, "testdata/official-object-cases.json", 18)
}

func runValueCases(t *testing.T, path string, count int) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []struct{ ID, Expr, InputJSON, ResultJSON string }
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) != count {
		t.Fatalf("inventory %d", len(f.Cases))
	}
	for _, c := range f.Cases {
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
						t.Fatalf("got %#v, want %s; %v", got, c.ResultJSON, err)
					}
				})
			}
		})
	}
}
