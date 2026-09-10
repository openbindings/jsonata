package numeric_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
)

func rational(t *testing.T, s string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatal(s)
	}
	return r
}

func TestDecimalValueOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(74661))
	for i := 0; i < 3000; i++ {
		a := fmt.Sprintf("%de%d", rng.Int63n(1<<61)-(1<<60), rng.Intn(80)-40)
		b := fmt.Sprintf("%de%d", rng.Int63n(1<<61)-(1<<60), rng.Intn(80)-40)
		x, y := rational(t, a), rational(t, b)
		if got := numeric.CompareText(a, b); got != x.Cmp(y) {
			t.Fatalf("compare %s %s: %d", a, b, got)
		}
		for _, s := range []string{a, b} {
			text, ok := numeric.Text(numeric.FromText(s))
			if !ok || rational(t, text).Cmp(rational(t, s)) != 0 {
				t.Fatalf("carriage %s => %s", s, text)
			}
		}
		for _, op := range []string{"+", "-", "*"} {
			want := new(big.Rat)
			switch op {
			case "+":
				want.Add(x, y)
			case "-":
				want.Sub(x, y)
			case "*":
				want.Mul(x, y)
			}
			got, err := numeric.DefaultLimits().Calculate(op, json.Number(a), json.Number(b), 0)
			if err != nil {
				t.Fatal(err)
			}
			s, ok := numeric.Text(got)
			if !ok || rational(t, s).Cmp(want) != 0 {
				t.Fatalf("%s %s %s => %v want %s", a, op, b, got, want.RatString())
			}
		}
	}
}

func TestHugeExponentValueComparison(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want int
	}{
		{"1e999999999999999999999999", "10e999999999999999999999998", 0},
		{"-1e-999999999999999999999999", "0", -1},
		{"-0e999999999999999999999999", "0", 0},
		{"1.00000000000000000000000000000001e1000000", "1e1000000", 1},
	} {
		if got := numeric.CompareText(tt.a, tt.b); got != tt.want {
			t.Fatalf("%s vs %s: %d", tt.a, tt.b, got)
		}
	}
}

func TestAssignedRoundedDivision(t *testing.T) {
	for _, tt := range []struct{ a, b, want string }{
		{"1", "3", "0.3333333333333333333333333333333333"},
		{"2", "3", "0.6666666666666666666666666666666667"},
		{"-1", "6", "-0.1666666666666666666666666666666667"},
		{"1", "1267650600228229401496703205376", "0.0000000000000000000000000000007888609052210118054117285652827862296732064351090230047702789306640625"},
	} {
		v, err := numeric.DefaultLimits().Calculate("/", json.Number(tt.a), json.Number(tt.b), 0)
		if err != nil {
			t.Fatal(err)
		}
		s, _ := numeric.Text(v)
		if rational(t, s).Cmp(rational(t, tt.want)) != 0 {
			t.Fatalf("%s/%s=%s want %s", tt.a, tt.b, s, tt.want)
		}
	}
	third, err := numeric.DefaultLimits().Calculate("/", float64(1), float64(3), 0)
	if err != nil {
		t.Fatal(err)
	}
	v, err := numeric.DefaultLimits().Calculate("*", third, float64(3), 0)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := numeric.Text(v)
	if numeric.CompareText(s, "0.9999999999999999999999999999999999") != 0 {
		t.Fatal(v)
	}
}
