package jsonata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	jsonata "github.com/openbindings/jsonata-runtime/go"
)

const nullJSON = `null`

func newExecutor(t *testing.T, options jsonata.Options) *jsonata.Executor {
	t.Helper()
	e, err := jsonata.New(options)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func equivalent(t *testing.T, a, b []byte) bool {
	t.Helper()
	decode := func(raw []byte) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	return reflect.DeepEqual(decode(a), decode(b))
}

func TestClosedBoundary(t *testing.T) {
	e := newExecutor(t, jsonata.Options{})
	for _, raw := range []string{
		nullJSON, `false`, `0`, `""`, `[]`, `{}`, `9223372036854775807`, `1e400`, `1e-400`,
		`0.12345678901234567890123456789`, `{"items":[null,false,[],{}],"__proto__":{"constructor":7}}`,
	} {
		for _, expression := range []string{`$`, `($f:=function($x){$x};$f($))`, `$eval("$")`} {
			out, err := e.Evaluate(context.Background(), expression, []byte(raw), nil)
			if err != nil {
				t.Fatalf("%s / %s: %v", expression, raw, err)
			}
			if !equivalent(t, out, []byte(raw)) {
				t.Fatalf("changed %s to %s", raw, out)
			}
		}
	}
	// encoding/json would erase the distinction, so check escaped code units
	// directly and then feed the result through another runtime evaluation.
	for _, raw := range []string{`"\ud800"`, `{"\ud800":"\udc00"}`, `["\ud800",{"\udc00":[null]}]`} {
		out, err := e.Evaluate(context.Background(), `$`, []byte(raw), nil)
		if err != nil || !strings.EqualFold(string(out), raw) {
			t.Fatalf("code units: %s / %v", out, err)
		}
		again, err := e.Evaluate(context.Background(), `$`, out, nil)
		if err != nil || !bytes.Equal(out, again) {
			t.Fatalf("reload: %s / %v", again, err)
		}
	}
	for _, expr := range []string{
		`function(){1}`, `{"fn":$string}`, `[/a/]`, `$match("b",/(a)?b/).groups`,
		`$ ~> |$|{"self":$}|`, `$flatten([])`, `$clone({})`, `2 ** 3`,
	} {
		if out, err := e.Evaluate(context.Background(), expr, []byte(`{}`), nil); err == nil {
			t.Fatalf("invalid result/language escaped %s: %s", expr, out)
		}
	}
	if _, err := e.Evaluate(context.Background(), `missing`, []byte(nullJSON), nil); !errors.Is(err, jsonata.ErrUndefined) {
		t.Fatal(err)
	}
	out, err := e.Evaluate(context.Background(), `[$exists($present),$type($present),$wide]`,
		[]byte(nullJSON), []byte(`{"present":null,"wide":9007199254740993}`))
	if err != nil || !equivalent(t, out, []byte(`[true,"null",9007199254740993]`)) {
		t.Fatalf("bindings: %s / %v", out, err)
	}
	for _, raw := range []string{nullJSON, `[]`, `1`, `true`, `"x"`, ``} {
		if _, err := e.Evaluate(context.Background(), `$`, []byte(nullJSON), []byte(raw)); err == nil {
			t.Fatalf("bindings admitted %q", raw)
		}
	}
	for _, raw := range []string{``, `{} {}`, `{"a":NaN}`, `[1,]`} {
		if _, err := e.Evaluate(context.Background(), `$`, []byte(raw), nil); err == nil {
			t.Fatalf("input admitted %q", raw)
		}
	}
}

func TestBudgetsCancellationAndReuse(t *testing.T) {
	limits := jsonata.NumericWorkLimits{MaxDigits: 8, MaxExponent: 8}
	e := newExecutor(t, jsonata.Options{NumericWork: &limits, MaxCompiledExpressions: 2})
	limits.MaxDigits = 100
	if _, err := e.Evaluate(context.Background(), `$power(2,100)`, []byte(nullJSON), nil); err == nil {
		t.Fatal("arithmetic budget ignored")
	}
	var group sync.WaitGroup
	for range 24 {
		group.Go(func() {
			out, err := e.Evaluate(context.Background(), `0.1+0.2`, []byte(nullJSON), nil)
			if err != nil || string(out) != "0.3" {
				t.Errorf("reuse: %s / %v", out, err)
			}
		})
	}
	group.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Evaluate(ctx, `$`, []byte(nullJSON), nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	if _, err := e.Evaluate(ctx, `($f:=function($n){$f($n+1)};$f(0))`, []byte(nullJSON), nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	out, err := e.Evaluate(context.Background(), `1+1`, []byte(nullJSON), nil)
	if err != nil || string(out) != "2" {
		t.Fatalf("recovery: %s / %v", out, err)
	}
	for _, options := range []jsonata.Options{
		{MaxExpressionBytes: -1}, {MaxInputBytes: -1}, {MaxOutputBytes: -1}, {MaxCompiledExpressions: -1}, {Timeout: -1},
	} {
		if _, err := jsonata.New(options); err == nil {
			t.Fatal("invalid budget admitted")
		}
	}
	for _, c := range []struct {
		options               jsonata.Options
		expr, input, bindings string
	}{
		{jsonata.Options{MaxExpressionBytes: 2}, `1+1`, nullJSON, ``},
		{jsonata.Options{MaxInputBytes: 4}, `$`, nullJSON, `{}`},
		{jsonata.Options{MaxOutputBytes: 3}, `'abcd'`, nullJSON, ``},
	} {
		var bindings []byte
		if c.bindings != "" {
			bindings = []byte(c.bindings)
		}
		if _, err := newExecutor(t, c.options).Evaluate(context.Background(), c.expr, []byte(c.input), bindings); err == nil {
			t.Fatal("budget ignored")
		}
	}
}
