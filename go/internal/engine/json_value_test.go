package gnata_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

func TestJSONValueChecksBeforeSerialization(t *testing.T) {
	for _, expr := range []string{`function(){1}`, `[function(){1}]`, `{"fn": $string}`, `/a/`, `{"a":[/a/]}`, `missing`, `$match("b", /(a)?b/).groups`} {
		compiled, err := gnata.Compile(expr)
		if err != nil {
			t.Fatal(err)
		}
		got, err := compiled.EvalBytes(context.Background(), json.RawMessage(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		if value, err := gnata.JSONValue(got); err == nil {
			t.Errorf("non-JSON value escaped: %s: %#v", expr, value)
		}
	}
	for _, expr := range []string{`null`, `[null]`, `[]`, `{}`, `{"id":9223372036854775807,"s":"\ud800","items":[null,false,""]}`, `($f := function(){1}; $f())`} {
		compiled, err := gnata.Compile(expr)
		if err != nil {
			t.Fatal(err)
		}
		got, err := compiled.EvalBytes(context.Background(), json.RawMessage(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := gnata.JSONValue(got); err != nil {
			t.Errorf("JSON value rejected: %s: %v", expr, err)
		}
	}
}

func TestValueWalkBoundsCyclesAndResumes(t *testing.T) {
	expr, _ := gnata.Compile(`$`)
	cycle := map[string]any{}
	cycle["self"] = cycle
	if _, err := expr.Eval(context.Background(), cycle); err == nil || !strings.Contains(err.Error(), "U_VALUE_LIMIT") {
		t.Fatalf("native cycle: %v", err)
	}
	cyclic, err := gnata.Compile(`$ ~> |$|{"self": $}|`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cyclic.EvalBytes(context.Background(), json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "U_VALUE_LIMIT") {
		t.Fatalf("computed cycle: %v", err)
	}
	got, err := expr.EvalBytes(context.Background(), json.RawMessage(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gnata.JSONValue(got); err != nil {
		t.Fatal(err)
	}
}
