package jsonata_test

import (
	"context"
	"strings"
	"testing"

	jsonata "github.com/openbindings/jsonata/go"
)

func TestBoundaryAdmissionCorrections(t *testing.T) {
	e := newExecutor(t, jsonata.Options{})
	for _, raw := range []string{
		`{"id":7`, `{"id":7} trailing`, `{"id":7} {}`, `{"id":7,}`,
		`{"id":7,"bad":[1,]}`, `{"id":7,"bad":"\q"}`, `{"id":7,"bad":NaN}`,
		`{"id":7,"id":8}`, `{"id":7,"id":7}`, `{"id":7,"\u0069d":7}`, `{"id":7,"bad":{"a":[],"a":{}}}`,
	} {
		for _, expr := range []string{`id`, `id = 7`, `$exists(id)`, `{"value":id}`, `42`} {
			t.Run(expr+"/"+raw, func(t *testing.T) {
				if output, err := e.Evaluate(context.Background(), expr, []byte(raw), nil); err == nil || output != nil {
					t.Fatalf("invalid input admitted: %s / %v", output, err)
				}
			})
		}
	}
	for _, raw := range []string{`{"x":1,"x":1}`, `{"x":{"a":1,"a":2}}`, `{"$":{"fake":2}}`, `{"$x":1}`} {
		if output, err := e.Evaluate(context.Background(), `$$`, []byte(`{"real":1}`), []byte(raw)); err == nil || output != nil {
			t.Errorf("invalid bindings admitted: %s / %v", output, err)
		}
	}
	for _, c := range []struct{ expr, input, bindings, want string }{
		{`a = b`, `{"a":{"length":0},"b":[]}`, ``, `false`},
		{`$distinct(items)`, `{"items":[{"length":0},[]]}`, ``, `[{"length":0},[]]`},
		{`$$`, `{"real":1}`, `{"":42}`, `{"real":1}`},
		{`$x`, `null`, `{"x":{"$":7}}`, `{"$":7}`},
		{`id`, `{"id":9007199254740993,"bad":{"n":1e999999}}`, ``, `9007199254740993`},
		{`$`, `{"__proto__":{"x":7},"constructor":2}`, ``, `{"__proto__":{"x":7},"constructor":2}`},
	} {
		var bindings []byte
		if c.bindings != "" {
			bindings = []byte(c.bindings)
		}
		output, err := e.Evaluate(context.Background(), c.expr, []byte(c.input), bindings)
		if err != nil || string(output) != c.want {
			t.Errorf("%s: %s / %v; want %s", c.expr, output, err, c.want)
		}
	}
}

func BenchmarkBoundaryAdmission(b *testing.B) {
	for _, size := range []struct {
		name string
		n    int
	}{{"small", 8}, {"1000", 1000}, {"1MiB", 65536}} {
		input := []byte(`{"id":7,"items":[` + strings.Repeat(`{"v":123456789},`, size.n-1) + `{"v":123456789}]}`)
		for _, expr := range []string{`id`, `($x := id; $x)`} {
			b.Run(size.name+"/"+expr, func(b *testing.B) {
				e, err := jsonata.New(jsonata.Options{})
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				b.ResetTimer()
				for range b.N {
					out, err := e.Evaluate(context.Background(), expr, input, nil)
					if err != nil || string(out) != "7" {
						b.Fatalf("%s / %v", out, err)
					}
				}
			})
		}
	}
}
