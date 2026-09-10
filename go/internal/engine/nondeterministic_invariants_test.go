package gnata_test

import (
	"context"
	"sync"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

// Check language invariants, not equality between random samples, engines or
// runs. This deliberately makes no statistical or cryptographic claim.
func TestRandomAndShuffleInvariants(t *testing.T) {
	const expression = `(
		$r := $random();
		$values := [9007199254740993, 0, 0, 9007199254740992];
		$shuffled := $shuffle($values);
		$type($r) = "number" and $r >= 0 and $r < 1 and
		$eval("$r") = $r and $number($string($r)) = $r and
		$count($shuffled) = 4 and
		$sort($shuffled) = [0, 0, 9007199254740992, 9007199254740993] and
		$values = [9007199254740993, 0, 0, 9007199254740992]
	)`
	compiled, err := gnata.Compile(expression)
	if err != nil {
		t.Fatal(err)
	}
	var pending sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		pending.Add(1)
		go func() {
			defer pending.Done()
			for sample := 0; sample < 64; sample++ {
				var value any
				var evalErr error
				if sample%2 == 0 {
					value, evalErr = compiled.Eval(context.Background(), nil)
				} else {
					value, evalErr = compiled.EvalBytes(context.Background(), []byte(`{}`))
				}
				if evalErr != nil || value != true {
					t.Errorf("random/shuffle invariant: %v / %v", value, evalErr)
					return
				}
			}
		}()
	}
	pending.Wait()
}
