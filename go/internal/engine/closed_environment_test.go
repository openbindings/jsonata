package gnata_test

import (
	"context"
	"encoding/json"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/syntax"
)

func TestClosedStandardEnvironment(t *testing.T) {
	for _, name := range []string{"flatten", "values", "readFile", "env", "process", "require", "fetch", "exec"} {
		for _, expr := range []string{"$" + name + "()", "$eval(" + string(mustJSON("$"+name+"()")) + ")"} {
			if err := syntax.Validate(expr); err != nil {
				t.Fatal(err)
			}
			compiled, err := gnata.Compile(expr)
			if err != nil {
				t.Fatal(err)
			}
			if value, err := compiled.EvalBytes(context.Background(), json.RawMessage(`{}`)); err == nil {
				t.Fatalf("extension %s executed: %#v", expr, value)
			}
		}
	}
	// User-declared language functions remain valid; environment closure is not
	// a blacklist of variable names or an instruction to remove lambda support.
	expr, err := gnata.Compile(`($flatten := function($v){$v}; $eval("$flatten([1])"))`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := expr.EvalBytes(context.Background(), json.RawMessage(`{}`))
	if err != nil || !gnata.DeepEqual(got, []any{float64(1)}) {
		t.Fatalf("declared function: %#v %v", got, err)
	}
}

func mustJSON(v string) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestRejectUndocumentedTransformPrefix(t *testing.T) {
	for _, source := range []string{`~$|{"x":1}|`, `{} ~> ~$|{"x":1}|`} {
		t.Run(source, func(t *testing.T) {
			if err := syntax.Validate(source); err == nil {
				t.Error("syntax validation admitted the undocumented ~ transform prefix")
			}
			if _, err := gnata.Compile(source); err == nil {
				t.Error("compiler admitted the undocumented ~ transform prefix")
			}
			nested, err := gnata.Compile("$eval(" + string(mustJSON(source)) + ")")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := nested.EvalBytes(context.Background(), json.RawMessage(`{}`)); err == nil {
				t.Error("dynamic evaluation admitted the undocumented ~ transform prefix")
			}
		})
	}
}
