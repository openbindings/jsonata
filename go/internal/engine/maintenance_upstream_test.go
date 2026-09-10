package gnata_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

func TestMaintenanceUpstreamWitnesses(t *testing.T) {
	raw, err := os.ReadFile("testdata/maintenance-upstream.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Expr, Dataset, Error string
		Undefined                bool
		Result                   json.RawMessage
	}
	if err = json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			input := []byte(`{}`)
			if c.Dataset != "" {
				input, err = os.ReadFile("testdata/datasets/" + c.Dataset + ".json")
				if err != nil {
					t.Fatal(err)
				}
			}
			e, err := gnata.Compile(c.Expr)
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.EvalBytes(context.Background(), input)
			if c.Error != "" {
				if err == nil || !strings.Contains(err.Error(), c.Error) {
					t.Fatalf("expected %s; got %#v / %v", c.Error, got, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.Undefined {
				if got != nil {
					t.Fatalf("expected undefined; got %#v", got)
				}
				return
			}
			want, err := gnata.DecodeJSON(c.Result)
			if err != nil {
				t.Fatal(err)
			}
			if !gnata.DeepEqual(got, want) {
				t.Fatalf("got %#v; expected %s", got, c.Result)
			}
		})
	}
}
