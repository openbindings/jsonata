package gnata_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestCheckpointPrecancelledEntryPoints(t *testing.T) {
	for _, src := range []string{"n", "n = 1", "$exists(n)", "($x:=n; $x)"} {
		expression, err := gnata.Compile(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, lane := range []string{"full", "bytes", "bytes-vars", "map"} {
			t.Run(src+"/"+lane, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				var err error
				switch lane {
				case "full":
					_, err = expression.Eval(ctx, map[string]any{"n": 1})
				case "bytes":
					_, err = expression.EvalBytes(ctx, json.RawMessage(`{"n":1}`))
				case "bytes-vars":
					_, err = expression.EvalBytesWithVars(ctx, json.RawMessage(`{"n":1}`), map[string]any{"x": 1})
				case "map":
					_, err = expression.EvalMap(ctx, map[string]json.RawMessage{"n": json.RawMessage(`1`)})
				}
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("pre-cancelled %s returned %v", lane, err)
				}
			})
		}
	}
}
