package gnata_test

import (
	"context"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/jstring"
)

func TestPinnedUnicodeCaseContexts(t *testing.T) {
	for _, c := range []struct{ input, upper, lower string }{
		{"", "", ""},
		{"aBc", "ABC", "abc"},
		{"Straße", "STRASSE", "straße"},
		{"İIıi", "İIII", "i\u0307iıi"},
		{"é e\u0301", "É E\u0301", "é e\u0301"},
		{"\U00010d50", "\U00010d50", "\U00010d70"},
		{"\U00010d70", "\U00010d50", "\U00010d70"},
		{"Σ", "Σ", "σ"},
		{"ΟΣ", "ΟΣ", "ος"},
		{"ΟΣΑ", "ΟΣΑ", "οσα"},
		{"\u0345Σ", "ΙΣ", "\u0345σ"},
		{"AΣ\u0345", "AΣΙ", "aς\u0345"},
		{"A\u0345Σ", "AΙΣ", "a\u0345ς"},
		{"AΣ\u0345B", "AΣΙB", "aσ\u0345b"},
		{"\U00010d50Σ", "\U00010d50Σ", "\U00010d70ς"},
		{"AΣ\U00010d50", "AΣ\U00010d50", "aσ\U00010d70"},
	} {
		for _, op := range []struct{ expr, want string }{{"$uppercase($)", c.upper}, {"$lowercase($)", c.lower}} {
			e, err := gnata.Compile(op.expr)
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.Eval(context.Background(), c.input)
			if err != nil || got != op.want {
				t.Fatalf("%s %q: %#v %v want %q", op.expr, c.input, got, err, op.want)
			}
		}
	}
	for unit := uint16(0xd800); unit <= 0xdfff; unit++ {
		s := jstring.CodeUnit(unit)
		for _, c := range []struct{ input, want string }{{"A" + s + "Σ", "a" + s + "σ"}, {"AΣ" + s + "B", "aς" + s + "b"}} {
			e, err := gnata.Compile("$lowercase($)")
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.Eval(context.Background(), c.input)
			if err != nil || got != c.want {
				t.Fatalf("unit %x: %#v %v", unit, got, err)
			}
		}
	}
}
