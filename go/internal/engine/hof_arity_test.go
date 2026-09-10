package gnata_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
)

func TestHigherOrderArity(t *testing.T) {
	raw, err := os.ReadFile("testdata/official-hof-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct{ ID, Expr, InputJSON, ResultJSON, ErrorCode string }
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 26 {
		t.Fatalf("inventory: %d", len(fixture.Cases))
	}
	for _, c := range fixture.Cases {
		t.Run(c.ID, func(t *testing.T) {
			e, err := gnata.Compile(c.Expr)
			if err != nil {
				t.Fatal(err)
			}
			input, err := gnata.DecodeJSON([]byte(c.InputJSON))
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"full", "bytes"} {
				t.Run(mode, func(t *testing.T) {
					var got any
					var err error
					if mode == "full" {
						got, err = e.Eval(context.Background(), input)
					} else {
						got, err = e.EvalBytes(context.Background(), []byte(c.InputJSON))
					}
					if c.ErrorCode != "" {
						var je *evaluator.JSONataError
						if !errors.As(err, &je) || je.Code != c.ErrorCode {
							t.Fatalf("got %v / %v; want %s", got, err, c.ErrorCode)
						}
						return
					}
					want, decodeErr := gnata.DecodeJSON([]byte(c.ResultJSON))
					if decodeErr != nil || err != nil || !gnata.DeepEqual(got, want) {
						t.Fatalf("got %v / %v; want %s; decode %v", got, err, c.ResultJSON, decodeErr)
					}
				})
			}
		})
	}
}
