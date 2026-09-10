package gnata_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
	"github.com/openbindings/jsonata/go/internal/engine/internal/numeric"
)

func TestNumericBudgetInvariance(t *testing.T) {
	for _, source := range []string{`1/3`, `$eval("1/3")`, `$average([1,0,0])`, `$power(2,-100)`, `$round(9007199254740993.25,1)`, `($f:=function($x){$sum([$x,0.1,0.2])};$f(9007199254740993))`} {
		var values []any
		for _, size := range []uint32{512, 4096} {
			limits := gnata.NumericWorkLimits{MaxDigits: size, MaxExponent: int32(size)}
			e, err := gnata.CompileWithOptions(source, gnata.CompileOptions{NumericWork: &limits})
			if err != nil {
				t.Fatal(err)
			}
			limits.MaxDigits = 1 // compiled expression must own its snapshot
			v, err := e.Eval(context.Background(), nil)
			if err != nil {
				t.Fatalf("%s: %v", source, err)
			}
			values = append(values, v)
		}
		if numeric.Compare(values[0], values[1]) != 0 {
			t.Fatalf("budget changed %s: %v", source, values)
		}
	}
}

func TestNumericBudgetIsolation(t *testing.T) {
	low, err := gnata.CompileWithOptions(`1/3`, gnata.CompileOptions{NumericWork: &gnata.NumericWorkLimits{MaxDigits: 16, MaxExponent: 512}})
	if err != nil {
		t.Fatal(err)
	}
	high, err := gnata.Compile(`1/3`)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, err := low.Eval(context.Background(), nil); err == nil {
				t.Errorf("low budget silently succeeded: %v", v)
			}
			v, err := high.Eval(context.Background(), nil)
			if err != nil || numeric.Compare(v, json.Number("0.3333333333333333333333333333333333")) != 0 {
				t.Errorf("high budget affected: %v %v", v, err)
			}
		}()
	}
	wg.Wait()
}
