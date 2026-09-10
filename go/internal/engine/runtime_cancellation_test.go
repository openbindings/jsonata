package gnata_test

import (
	"context"
	"errors"
	"testing"
	"time"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

func TestRuntimeCancellationNestedEval(t *testing.T) {
	expr, err := gnata.Compile(`$eval(expression)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err = expr.EvalBytes(ctx, []byte(`{"expression":"$sum($map([1..1000000], function($v){$v*$v}))"}`))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("nested eval lost deadline classification: %v", err)
	}
	v, err := expr.EvalBytes(context.Background(), []byte(`{"expression":"1+1"}`))
	if err != nil || v != float64(2) {
		t.Fatalf("cancelled call poisoned reuse: %v %v", v, err)
	}
}

func BenchmarkRuntimeCancellation(b *testing.B) {
	expr, err := gnata.Compile(`{"parameters":{"path":{"id":id}},"body":body}`)
	if err != nil {
		b.Fatal(err)
	}
	input := []byte(`{"id":"7","body":{"name":"example"}}`)
	for _, cancellable := range []bool{false, true} {
		b.Run(map[bool]string{false: "background", true: "context"}[cancellable], func(b *testing.B) {
			ctx := context.Background()
			if cancellable {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := expr.EvalBytes(ctx, input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
