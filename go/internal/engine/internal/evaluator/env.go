package evaluator

import (
	"context"
	"maps"
	"time"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/parser"
)

// defaultMaxCallDepth is the maximum recursive call depth before U1001 is returned.
// Matches the JSONata reference implementation's default.
const defaultMaxCallDepth = 100

// callCounter tracks the current recursive call depth across all child environments.
// A pointer is shared so all nested envs increment/decrement the same counter.
type callCounter struct {
	depth     int
	evalDepth int
	max       int
}

// Environment holds variable bindings for an evaluation context.
// It forms a linked chain for lexical scoping.
type Environment struct {
	standardLibraryOnly bool
	parent              *Environment
	bindings            map[string]any
	calls               *callCounter // shared call-depth counter; nil inherits from parent
	ctx                 context.Context
	numbers             *numeric.Limits
	timestamp           time.Time
}

// UseStandardLibraryOnly omits the upstream compatibility hook. The transform
// operator still copies internally, including inside nested scopes and $eval.
func (e *Environment) UseStandardLibraryOnly() {
	e.standardLibraryOnly = true
	e.Bind("clone", nil)
}

func (e *Environment) StandardLibraryOnly() bool {
	return e != nil && (e.standardLibraryOnly || e.parent.StandardLibraryOnly())
}

// ResetTimestamp begins an independent evaluation. Nested scopes and $eval
// inherit this instant; the shared builtin environment is never mutated.
func (e *Environment) ResetTimestamp() { e.timestamp = time.Now().UTC().Truncate(time.Millisecond) }

func (e *Environment) Timestamp() time.Time {
	if !e.timestamp.IsZero() {
		return e.timestamp
	}
	if e.parent != nil {
		return e.parent.Timestamp()
	}
	return time.Time{}
}

func (e *Environment) NumberLimits() numeric.Limits {
	if e == nil {
		return numeric.DefaultLimits()
	}
	if e.numbers != nil {
		return *e.numbers
	}
	return e.parent.NumberLimits()
}

// Each evaluation owns a copy; closures inherit it without process globals.
func (e *Environment) SetNumberLimits(l numeric.Limits) { e.numbers = &l }

// NewEnvironment creates a root environment with no bindings.
func NewEnvironment() *Environment {
	return &Environment{
		bindings:  make(map[string]any),
		calls:     &callCounter{max: defaultMaxCallDepth},
		timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}
}

// NewChildEnvironment creates a child scope inheriting from parent.
func NewChildEnvironment(parent *Environment) *Environment {
	env := &Environment{parent: parent, bindings: make(map[string]any)}
	if parent != nil {
		env.calls = parent.callCounter()
	}
	return env
}

// callCounter returns the shared call counter, walking up the chain if needed.
func (e *Environment) callCounter() *callCounter {
	if e == nil {
		return &callCounter{max: defaultMaxCallDepth}
	}
	if e.calls != nil {
		return e.calls
	}
	return e.parent.callCounter()
}

// Bind sets a variable in this environment.
func (e *Environment) Bind(name string, value any) {
	e.bindings[name] = InternalizeValue(value)
}

// Parent returns the parent environment (nil for root environments).
func (e *Environment) Parent() *Environment {
	return e.parent
}

// Lookup looks up a variable, walking the parent chain.
// Returns (nil, false) if not found.
func (e *Environment) Lookup(name string) (any, bool) {
	if v, ok := e.bindings[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.Lookup(name)
	}
	return nil, false
}

// LookupWithEnv looks up a variable and returns both the value and the
// specific environment in which the binding was found. This is used by the
// parent operator (%) so that chained %.% navigations correctly use the
// parent of the binding's environment, not the parent of the starting env.
// Returns (nil, nil, false) if not found.
func (e *Environment) LookupWithEnv(name string) (any, *Environment, bool) {
	if v, ok := e.bindings[name]; ok {
		return v, e, true
	}
	if e.parent != nil {
		return e.parent.LookupWithEnv(name)
	}
	return nil, nil, false
}

// ResetCallCounter installs a fresh call-depth counter on this environment,
// decoupling it from any inherited parent counter. Use this when creating a
// per-eval child environment from a shared parent to avoid cross-eval interference.
func (e *Environment) ResetCallCounter() {
	e.calls = &callCounter{max: defaultMaxCallDepth}
}

// IncrEvalDepth increments the $eval nesting counter and returns an error if
// the maximum depth is exceeded. Must be paired with DecrEvalDepth via defer.
func (e *Environment) IncrEvalDepth(maxDepth int) error {
	c := e.callCounter()
	c.evalDepth++
	if c.evalDepth > maxDepth {
		c.evalDepth--
		return &JSONataError{Code: "D3121", Message: "$eval: maximum nesting depth exceeded"}
	}
	return nil
}

// DecrEvalDepth decrements the $eval nesting counter.
func (e *Environment) DecrEvalDepth() {
	c := e.callCounter()
	c.evalDepth--
}

// Clone creates a shallow copy of the environment, duplicating the bindings map
// but sharing the same parent and call counter references.
func (e *Environment) Clone() *Environment {
	child := &Environment{
		standardLibraryOnly: e.standardLibraryOnly,
		bindings:            make(map[string]any, len(e.bindings)),
		parent:              e.parent,
		calls:               e.calls,
		ctx:                 e.ctx,
		numbers:             e.numbers,
		timestamp:           e.timestamp,
	}
	maps.Copy(child.bindings, e.bindings)
	return child
}

// Context returns the context.Context associated with this environment.
// It walks the parent chain if the local ctx is nil, falling back to
// context.Background() for root environments without an explicit context.
func (e *Environment) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	if e.parent != nil {
		return e.parent.Context()
	}
	return context.Background()
}

// SetContext sets a context.Context on this environment, making it
// available to Eval and all child environments via Context().
func (e *Environment) SetContext(ctx context.Context) {
	e.ctx = ctx
}

// Range iterates over the bindings in this environment (not parents).
func (e *Environment) Range(fn func(name string, val any)) {
	for k, v := range e.bindings {
		fn(k, v)
	}
}

// LookupDirect checks only the direct bindings (no parent chain walk).
func (e *Environment) LookupDirect(name string) (any, bool) {
	v, ok := e.bindings[name]
	return v, ok
}

// BuiltinFunction is a native Go function implementing a JSONata built-in.
// args are the evaluated arguments; focus is the current context value.
type BuiltinFunction func(args []any, focus any) (any, error)

// EnvAwareBuiltin is a built-in function that receives the current evaluation
// environment. This is required for functions that need to dispatch callbacks
// with the correct per-evaluation call counter (HOFs like $map, $filter) or
// that create child scopes ($eval).
type EnvAwareBuiltin func(args []any, focus any, env *Environment) (any, error)

// NativeFunction owns callable identity independently of a Go function's code
// address. Distinct closures can share code while having different captures.
type NativeFunction struct {
	implementation any
	arity          int
}

// NewNativeFunction retains callable identity and callback arity independently
// of the Go slice-based calling convention. Metadata travels with aliases,
// stored functions and partial application, never with the callee's spelling.
func NewNativeFunction(implementation any, arity int) *NativeFunction {
	return &NativeFunction{implementation: implementation, arity: arity}
}

// FunctionArity is the language callback arity, not reflect.Type.NumIn (all Go
// builtins receive an argument slice). Unannotated host functions retain the
// historical value-only callback convention.
func FunctionArity(fn any) int {
	switch f := fn.(type) {
	case *NativeFunction:
		return f.arity
	case *Lambda:
		return len(f.Params)
	case *SignedBuiltin:
		return len(f.ParsedSig)
	default:
		return 1
	}
}

// SignedBuiltin wraps a BuiltinFunction with a type signature for arity and
// type validation at the direct call site. HOF callbacks that invoke the
// function via ApplyFunction bypass signature validation, allowing extra
// arguments (key, index, array) to be passed silently.
type SignedBuiltin struct {
	Fn        BuiltinFunction
	Sig       string
	ParsedSig []parser.ParamSpec // pre-parsed signature; avoids re-parsing on every call
}

// Lambda represents a user-defined function (lambda expression).
type Lambda struct {
	Params        []string           // parameter names
	Body          *parser.Node       // function body AST node
	Closure       *Environment       // lexical scope at definition site
	Thunk         bool               // for tail-call optimization
	Sig           string             // type signature (Wave 5)
	ParsedSig     []parser.ParamSpec // pre-parsed signature; avoids re-parsing per call
	CapturedFocus any                // focus ($) captured at definition time for zero-param closures
}
