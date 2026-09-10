package gnata_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

// Read values directly: encoding/json's replacement policy is not an oracle
// for the evaluator's assigned string contents.
func TestAuthoredStringCodeUnits(t *testing.T) {
	for unit := 0xd800; unit <= 0xdfff; unit++ {
		expr, err := gnata.Compile(fmt.Sprintf(`"\u%04x"`, unit))
		if err != nil {
			t.Fatal(err)
		}
		got, err := expr.Eval(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		want := string([]byte{0xe0 | byte(unit>>12), 0x80 | byte((unit>>6)&0x3f), 0x80 | byte(unit&0x3f)})
		if got != want {
			t.Fatalf("code unit %04x replaced: want bytes %x, got %x", unit, want, got)
		}
	}
}

func TestStringBoundaryAndBuiltins(t *testing.T) {
	ctx := context.Background()
	for _, source := range []string{`s`, `$substring(s,0)`, `$uppercase(s)`, `$lowercase(s)`, `$trim(s)`, `$pad(s,1)`, `$join($split(s,""),"")`, `$clone({"s":s}).s`, `$eval($string({"s":s})).s`} {
		t.Run(source, func(t *testing.T) {
			expr, err := gnata.Compile(source)
			if err != nil {
				t.Fatal(err)
			}
			for unit := 0xd800; unit <= 0xdfff; unit++ {
				want := string([]byte{0xe0 | byte(unit>>12), 0x80 | byte((unit>>6)&0x3f), 0x80 | byte(unit&0x3f)})
				raw := json.RawMessage(fmt.Sprintf(`{"s":"\u%04x"}`, unit))
				input, err := gnata.DecodeJSON(raw)
				if err != nil {
					t.Fatal(err)
				}
				full, fullErr := expr.Eval(ctx, input)
				fast, fastErr := expr.EvalBytes(ctx, raw)
				withVars, varsErr := expr.EvalBytesWithVars(ctx, raw, nil)
				mapped, mapErr := expr.EvalMap(ctx, map[string]json.RawMessage{"s": json.RawMessage(fmt.Sprintf(`"\u%04x"`, unit))})
				if fullErr != nil || fastErr != nil || varsErr != nil || mapErr != nil || full != want || fast != want || withVars != want || mapped != want {
					t.Fatalf("unit %x: full=%#v/%v bytes=%#v/%v vars=%#v/%v map=%#v/%v", unit, full, fullErr, fast, fastErr, withVars, varsErr, mapped, mapErr)
				}
			}
		})
	}
	for _, source := range []string{`$keys($)`, `$clone($)`, `$eval($string($))`} {
		expr, err := gnata.Compile(source)
		if err != nil {
			t.Fatal(err)
		}
		raw := json.RawMessage(`{"\ud800":1,"\udc00":2,"�":3}`)
		input, err := gnata.DecodeJSON(raw)
		if err != nil {
			t.Fatal(err)
		}
		full, fullErr := expr.Eval(ctx, input)
		fast, fastErr := expr.EvalBytes(ctx, raw)
		if fullErr != nil || fastErr != nil || !reflect.DeepEqual(gnata.NormalizeValue(full), gnata.NormalizeValue(fast)) {
			t.Fatalf("key boundary %s: %#v %#v %v %v", source, full, fast, fullErr, fastErr)
		}
	}
}

func TestAuthoredStringComposition(t *testing.T) {
	cases := []struct {
		expr string
		want any
	}{
		{`"\ud83d" & "\ude00"`, "😀"},
		{`("\ud83d" & "\ude00") = "😀"`, true},
		{`"\ud800" = "�"`, false},
		{`$count($keys({"\ud800":1,"\udc00":2,"�":3}))`, float64(3)},
		{`($x:="\ud83d"; $f:=function($s){$s & "\ude00"}; $f($x))`, "😀"},
		{`$eval('"\\ud83d" & "\\ude00"')`, "😀"},
		{`"a" & "😀" & "é"`, "a😀é"},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			expr, err := gnata.Compile(c.expr)
			if err != nil {
				t.Fatal(err)
			}
			got, err := expr.Eval(context.Background(), nil)
			if err != nil || got != c.want {
				t.Fatalf("want %#v, got %#v (%v)", c.want, got, err)
			}
		})
	}
}
