package gnata_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

// Host barriers deliberately separate wall-clock reads. They are test
// instrumentation, not additional functions in the closed SDK environment.
func TestEvaluationClockStableAndConcurrent(t *testing.T) {
	for _, clock := range []string{`$millis()`, `$now()`, `$now("[Y0001]-[M01]-[D01]T[H01]:[m01]:[s01].[f001]")`, `$eval("$millis()")`, `$eval("$now()")`} {
		t.Run(clock, func(t *testing.T) {
			expr, err := gnata.Compile(`($before := ` + clock + `; $barrier(); [$before, ` + clock + `])`)
			if err != nil {
				t.Fatal(err)
			}
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			defer once.Do(func() { close(release) })
			env := gnata.NewCustomEnv(map[string]gnata.CustomFunc{"barrier": func([]any, any) (any, error) { close(entered); <-release; return nil, nil }})
			type answer struct {
				value any
				err   error
			}
			answers := make(chan answer, 1)
			go func() { v, e := expr.EvalWithCustomFuncs(context.Background(), nil, env); answers <- answer{v, e} }()
			<-entered
			time.Sleep(15 * time.Millisecond)
			otherEnv := gnata.NewCustomEnv(map[string]gnata.CustomFunc{"barrier": func([]any, any) (any, error) { return nil, nil }})
			other, err := expr.EvalWithCustomFuncs(context.Background(), nil, otherEnv)
			once.Do(func() { close(release) })
			first := <-answers
			if err != nil || first.err != nil {
				t.Fatalf("%v / %v", first.err, err)
			}
			before := gnata.NormalizeValue(first.value).([]any)
			after := gnata.NormalizeValue(other).([]any)
			if !reflect.DeepEqual(before[0], before[1]) || !reflect.DeepEqual(after[0], after[1]) {
				t.Fatalf("unstable clocks: %#v / %#v", before, after)
			}
			if reflect.DeepEqual(before[0], after[0]) {
				t.Fatal("separate evaluations shared a timestamp")
			}
		})
	}
}

func TestEvaluationClockSameAcrossConstructors(t *testing.T) {
	e, err := gnata.Compile(`$now() = $fromMillis($millis())`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := e.Eval(context.Background(), nil)
	if err != nil || v != true {
		t.Fatalf("inconsistent clocks: %v / %v", v, err)
	}
}
