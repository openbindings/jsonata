package gnata_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

// Ordinary-value witnesses against JSONata 2.1.1. Exact numerical operations
// are deliberately not assigned new semantics by these compatibility repairs.
func TestNativeCompatibility(t *testing.T) {
	cases := []struct{ name, expr, input, want, code string }{
		{"arithmetic-right-error", `2 + ""`, `{}`, "", "T2002"},
		{"arithmetic-left-null", `null + 2`, `{}`, "", "T2001"},
		{"decode-url-error", `$decodeUrl("%E0%A4%A")`, `{}`, "", "D3140"},
		{"decode-component-error", `$decodeUrlComponent("%E0%A4%A")`, `{}`, "", "D3140"},
		{"decode-invalid-utf8", `$decodeUrlComponent("%ED%A0%80")`, `{}`, "", "D3140"},
		{"decode-reserved", `$decodeUrl("a%2fb%3Ac%3fd%23e")`, `{}`, `"a%2fb%3Ac%3fd%23e"`, ""},
		{"decode-component", `$decodeUrlComponent("a%2fb%3Ac%3fd%23e")`, `{}`, `"a/b:c?d#e"`, ""},
		{"encode-replacement-char", `$encodeUrlComponent("�")`, `{}`, `"%EF%BF%BD"`, ""},
		{"format-no-placeholder", `$formatNumber(42,"---")`, `{}`, "", "D3086"},
		{"filter-singleton", `$filter([1,2],function($x){$x=1})`, `{}`, `1`, ""},
		{"filter-object-member", `{"v":$filter([1,2],function($x){$x=1})}`, `{}`, `{"v":1}`, ""},
		{"filter-empty", `$filter([1,2],function($x){false})`, `{}`, "", ""},
		{"filter-multiple", `$filter([1,2],function($x){true})`, `{}`, `[1,2]`, ""},
		{"wildcard", `items.*`, `{"items":[{"a":1,"b":2},{"c":3}]}`, `[1,2,3]`, ""},
		{"wildcard-nested", `items.*`, `{"items":[{"a":[[1,2],[3]]},{"b":[4]}]}`, `[1,2,3,4]`, ""},
		{"match-single", `$match("a12",/[0-9]+/)`, `{}`, `{"match":"12","index":1,"groups":[]}`, ""},
		{"match-multiple", `$match("a1b2",/[0-9]/)`, `{}`, `[{"match":"1","index":1,"groups":[]},{"match":"2","index":3,"groups":[]}]`, ""},
		{"match-utf16-index", `$match("😀a1",/[0-9]/)`, `{}`, `{"match":"1","index":3,"groups":[]}`, ""},
		{"match-none", `$match("abc",/[0-9]/)`, `{}`, "", ""},
		{"clone-object", `$clone($)`, `{"n":0.10000000000000002,"a":[],"b":null}`, `{"n":0.10000000000000002,"a":[],"b":null}`, ""},
		{"clone-focus", `$clone()`, `{"a":1}`, `{"a":1}`, ""},
		{"clone-array", `$clone([[],[1],null])`, `{}`, `[[],[1],null]`, ""},
		{"clone-function", `$clone({"f":function($x){$x}})`, `{}`, `{"f":""}`, ""},
		{"clone-missing", `$clone(missing)`, `{}`, "", ""},
		{"clone-null", `$clone(null)`, `{}`, "", "T0410"},
		{"clone-number", `$clone(1)`, `{}`, "", "T0410"},
		{"clone-override", `($clone:=function($x){{"kept":true}}; $ ~> |$|{"flag":true}|)`, `{}`, `{"kept":true,"flag":true}`, ""},
		{"clone-invalid-override", `($clone:=42; $ ~> |$|{"flag":true}|)`, `{}`, "", "T2013"},
	}
	for _, c := range cases {
		for _, mode := range []string{"full", "bytes"} {
			t.Run(c.name+"/"+mode, func(t *testing.T) {
				e, err := gnata.Compile(c.expr)
				if err != nil {
					t.Fatal(err)
				}
				var got any
				if mode == "bytes" {
					got, err = e.EvalBytes(context.Background(), json.RawMessage(c.input))
				} else {
					data, decodeErr := gnata.DecodeJSON(json.RawMessage(c.input))
					if decodeErr != nil {
						t.Fatal(decodeErr)
					}
					got, err = e.Eval(context.Background(), data)
				}
				if c.code != "" {
					if err == nil || !strings.Contains(err.Error(), c.code) {
						t.Fatalf("want error %s; got %v (%v)", c.code, got, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if c.want == "" {
					if got != nil {
						t.Fatalf("want undefined; got %#v", got)
					}
					return
				}
				// Compare JSON shapes without gnata.DeepEqual (the subject under test).
				encoded, err := json.Marshal(gnata.NormalizeValue(got))
				if err != nil {
					t.Fatal(err)
				}
				var actual, want any
				da := json.NewDecoder(strings.NewReader(string(encoded)))
				da.UseNumber()
				_ = da.Decode(&actual)
				dw := json.NewDecoder(strings.NewReader(c.want))
				dw.UseNumber()
				_ = dw.Decode(&want)
				if !reflect.DeepEqual(actual, want) {
					t.Fatalf("want %s; got %s", c.want, encoded)
				}
			})
		}
	}
}
