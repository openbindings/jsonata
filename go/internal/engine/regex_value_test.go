package gnata_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestRegexValueContract(t *testing.T) {
	raw, err := os.ReadFile("testdata/official-regex-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []struct{ ID, Expr, InputJSON, ResultJSON, Error string }
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) != 59 {
		t.Fatalf("inventory %d", len(f.Cases))
	}
	for _, c := range f.Cases {
		t.Run(c.ID, func(t *testing.T) {
			e, err := gnata.Compile(c.Expr)
			if c.Error == "syntax" {
				if err == nil {
					t.Fatal("invalid syntax accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := gnata.DecodeJSON([]byte(c.InputJSON))
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
					if c.Error != "" {
						if err == nil || !strings.Contains(err.Error(), c.Error) {
							t.Fatalf("want %s, got %v/%v", c.Error, got, err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					want, err := gnata.DecodeJSON([]byte(c.ResultJSON))
					if err != nil {
						t.Fatal(err)
					}
					if !gnata.DeepEqual(got, want) {
						t.Fatalf("got %#v; want %s", got, c.ResultJSON)
					}
				})
			}
		})
	}
}
