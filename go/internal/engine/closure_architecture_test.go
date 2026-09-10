package gnata_test

import (
	"context"
	"encoding/json"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

// This is the stronger official implementation gate, not a claim that stock
// binary64 behavior violates Core. Keep failures visible until it is satisfied.
func TestClosureArchitectureObservation(t *testing.T) {
	for index, source := range []string{`9007199254740993 = 9007199254740992`, `$number("9007199254740993") = $number("9007199254740992")`, `n = 9007199254740992`, `0.1 + 0.2 = 0.3`} {
		expr, err := gnata.Compile(source)
		if err != nil {
			t.Fatal(err)
		}
		got, err := expr.Eval(context.Background(), map[string]any{"n": json.Number("9007199254740993")})
		want := index == 3
		if err != nil || got != want {
			t.Errorf("official numerical gate: %q got=%v want=%v error=%v", source, got, want, err)
		}
	}
}
