package gnata_test

import (
	"context"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestWildcardValueBoundaries(t *testing.T) {
	for _, c := range []struct{ expression, input, expected string }{
		{`$exists([$sum,$count].*)`, `{}`, `false`},
		{`$exists([1,2].*)`, `{}`, `false`},
		{`$exists([null,false,1,"x"].*)`, `{}`, `false`},
		{`$exists(([1,2]).*)`, `{}`, `false`},
		{`[[1,2],[3]].*`, `{}`, `[1,2,3]`},
		{`[{"a":1},2].*`, `{}`, `1`},
		{`[[{"a":1}],[{"b":2}]].*`, `{}`, `[{"a":1},{"b":2}]`},
		{`$type({"f":$sum}.*)`, `{}`, `"function"`},
		{`*`, `[1,2]`, `[1,2]`},
		{`*`, `[{"a":1},{"a":2}]`, `[{"a":1},{"a":2}]`},
		{`*.a`, `[{"a":1},{"a":2}]`, `[1,2]`},
		{`$.*`, `[{"a":1},{"a":2}]`, `[1,2]`},
		{`(*).a`, `[{"a":1},{"a":2}]`, `[1,2]`},
		{`*#$i.{"value":$,"index":$i}`, `[{"a":1},{"a":2}]`, `[{"value":{"a":1},"index":0},{"value":{"a":2},"index":1}]`},
	} {
		t.Run(c.expression, func(t *testing.T) {
			e, err := gnata.Compile(c.expression)
			if err != nil {
				t.Fatal(err)
			}
			input, err := gnata.DecodeJSON([]byte(c.input))
			if err != nil {
				t.Fatal(err)
			}
			want, err := gnata.DecodeJSON([]byte(c.expected))
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"full", "bytes", "vars"} {
				var got any
				switch mode {
				case "full":
					got, err = e.Eval(context.Background(), input)
				case "bytes":
					got, err = e.EvalBytes(context.Background(), []byte(c.input))
				case "vars":
					got, err = e.EvalBytesWithVars(context.Background(), []byte(c.input), nil)
				}
				if err != nil || !gnata.DeepEqual(got, want) {
					t.Fatalf("%s: got %#v / %v; want %s", mode, got, err, c.expected)
				}
			}
		})
	}
}
