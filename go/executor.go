// Package jsonata evaluates JSONata expressions through a closed, value-preserving
// JSON-text boundary. It has no OpenBindings or SDK dependency.
package jsonata

import (
	"context"

	engine "github.com/openbindings/jsonata/go/internal/engine"
)

// NumericWorkLimits bounds work, not precision. Insufficient budgets reject;
// they never change the assigned result of a successful evaluation.
type NumericWorkLimits = engine.NumericWorkLimits

// Options contains instance-owned resource budgets. Zero fields select defaults.
type Options = engine.JSONExecutorOptions

// ErrUndefined distinguishes an absent result from the JSON value null.
var ErrUndefined = engine.ErrJSONUndefined

// Executor owns a bounded compilation cache. It is safe for concurrent reuse.
// It retains neither request data nor request bindings between evaluations.
type Executor struct{ inner *engine.JSONExecutor }

// New selects 256 KiB expression, 8 MiB combined input/bindings and output,
// 64 compiled expressions, and a one-second cooperative timeout by default.
func New(options Options) (*Executor, error) {
	inner, err := engine.NewJSONExecutor(options)
	if err != nil {
		return nil, err
	}
	return &Executor{inner: inner}, nil
}

// Evaluate accepts one JSON value and returns one JSON value. Bindings are
// omitted with nil or supplied as a JSON object whose names omit the '$' prefix.
// No host functions or objects can cross this boundary. Cancellation is
// cooperative; parsing, serialization and individual library calls are checked
// at their boundaries, not forcibly preempted. Returned bytes are caller-owned.
func (e *Executor) Evaluate(ctx context.Context, expression string, inputJSON, bindingsJSON []byte) ([]byte, error) {
	if e == nil {
		return (*engine.JSONExecutor)(nil).Evaluate(ctx, expression, inputJSON, bindingsJSON)
	}
	return e.inner.Evaluate(ctx, expression, inputJSON, bindingsJSON)
}
