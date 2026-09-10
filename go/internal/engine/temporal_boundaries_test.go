package gnata_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestTemporalBoundaries(t *testing.T) {
	b, err := os.ReadFile("testdata/official-temporal-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			ID, Expr, InputJSON, ResultJSON, Error string
			StandardOnly                           bool
		}
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Cases {
		t.Run(c.ID, func(t *testing.T) {
			e, err := gnata.CompileWithOptions(c.Expr, gnata.CompileOptions{StandardLibraryOnly: c.StandardOnly})
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"full", "bytes", "vars"} {
				t.Run(mode, func(t *testing.T) {
					var got any
					var err error
					switch mode {
					case "full":
						got, err = e.Eval(context.Background(), map[string]any{})
					case "bytes":
						got, err = e.EvalBytes(context.Background(), []byte(c.InputJSON))
					case "vars":
						got, err = e.EvalBytesWithVars(context.Background(), []byte(c.InputJSON), nil)
					}
					if c.Error != "" {
						if err == nil || !strings.Contains(err.Error(), c.Error) {
							t.Fatalf("expected %s, got %#v / %v", c.Error, got, err)
						}
						return
					}
					want, decodeErr := gnata.DecodeJSON([]byte(c.ResultJSON))
					if decodeErr != nil {
						t.Fatal(decodeErr)
					}
					if err != nil || !gnata.DeepEqual(got, want) {
						t.Fatalf("%s: %#v / %v; want %s", c.Expr, got, err, c.ResultJSON)
					}
				})
			}
		})
	}
}
