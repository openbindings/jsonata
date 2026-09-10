package gnata_test

import (
	"context"
	"errors"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

func TestResourceControlErrorClassification(t *testing.T) {
	for _, source := range []string{`$round(1,9007199254740993)`, `$round(1,4096.0000000000000000001)`, `$round(1,-4096.0000000000000000001)`} {
		e, err := gnata.Compile(source)
		if err != nil {
			t.Fatal(err)
		}
		_, err = e.Eval(context.Background(), nil)
		var limit *numeric.Error
		if !errors.As(err, &limit) || limit.Kind != "work-limit" {
			t.Fatalf("%s: %v", source, err)
		}
	}
	for _, c := range []struct{ source, code string }{{`$round(1,0.5)`, "T0410"}, {`$pad("x",10001)`, "U_OUTPUT_LIMIT"}, {`$pad("x",-10001)`, "U_OUTPUT_LIMIT"}} {
		e, err := gnata.Compile(c.source)
		if err != nil {
			t.Fatal(err)
		}
		_, err = e.Eval(context.Background(), nil)
		var language *evaluator.JSONataError
		if !errors.As(err, &language) || language.Code != c.code {
			t.Fatalf("%s: %v", c.source, err)
		}
	}
}
