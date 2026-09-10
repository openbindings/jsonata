package gnata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
)

var ErrJSONUndefined = errors.New("JSONata expression returned undefined")

type JSONExecutorOptions struct {
	MaxExpressionBytes     int
	MaxInputBytes          int
	MaxOutputBytes         int
	MaxCompiledExpressions int
	Timeout                time.Duration
	NumericWork            *NumericWorkLimits
}

type JSONExecutor struct {
	options  JSONExecutorOptions
	mu       sync.Mutex
	compiled map[string]*Expression
	keys     []string
}

func NewJSONExecutor(options JSONExecutorOptions) (*JSONExecutor, error) {
	if options.MaxExpressionBytes == 0 {
		options.MaxExpressionBytes = 256 << 10
	}
	if options.MaxInputBytes == 0 {
		options.MaxInputBytes = 8 << 20
	}
	if options.MaxOutputBytes == 0 {
		options.MaxOutputBytes = 8 << 20
	}
	if options.MaxCompiledExpressions == 0 {
		options.MaxCompiledExpressions = 64
	}
	if options.Timeout == 0 {
		options.Timeout = time.Second
	}
	if options.MaxExpressionBytes < 1 || options.MaxInputBytes < 1 || options.MaxOutputBytes < 1 ||
		options.MaxCompiledExpressions < 1 || options.Timeout < 0 {
		return nil, fmt.Errorf("JSONata budgets must be positive")
	}
	if options.NumericWork != nil {
		numericLimits := *options.NumericWork
		options.NumericWork = &numericLimits
		if _, err := CompileWithOptions("$", CompileOptions{NumericWork: &numericLimits}); err != nil {
			return nil, err
		}
	}
	return &JSONExecutor{options: options, compiled: make(map[string]*Expression)}, nil
}

func (e *JSONExecutor) compile(expression string) (*Expression, error) {
	if len(expression) > e.options.MaxExpressionBytes {
		return nil, fmt.Errorf("JSONata expression exceeds byte budget")
	}
	e.mu.Lock()
	compiled := e.compiled[expression]
	e.mu.Unlock()
	if compiled != nil {
		return compiled, nil
	}
	compiled, err := CompileWithOptions(expression, CompileOptions{NumericWork: e.options.NumericWork, StandardLibraryOnly: true})
	if err != nil {
		return nil, fmt.Errorf("compile JSONata: %w", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if existing := e.compiled[expression]; existing != nil {
		return existing, nil
	}
	if len(e.keys) == e.options.MaxCompiledExpressions {
		delete(e.compiled, e.keys[0])
		e.keys = e.keys[1:]
	}
	e.keys = append(e.keys, expression)
	e.compiled[expression] = compiled
	return compiled, nil
}

func (e *JSONExecutor) Evaluate(ctx context.Context, expression string, inputJSON, bindingsJSON []byte) ([]byte, error) {
	if e == nil || e.compiled == nil {
		return nil, fmt.Errorf("construct JSONata executor with New")
	}
	if ctx == nil {
		return nil, fmt.Errorf("JSONata evaluation requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()
	if len(inputJSON) > e.options.MaxInputBytes || len(bindingsJSON) > e.options.MaxInputBytes-len(inputJSON) {
		return nil, fmt.Errorf("JSONata input and bindings exceed byte budget")
	}
	compiled, err := e.compile(expression)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var bound map[string]any
	if bindingsJSON != nil {
		value, err := DecodeJSON(bindingsJSON)
		if err != nil {
			return nil, fmt.Errorf("JSONata bindings: %w", err)
		}
		object, ok := value.(*evaluator.OrderedMap)
		if !ok {
			return nil, fmt.Errorf("JSONata bindings must be a JSON object")
		}
		bound = object.ToMap()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := compiled.EvalBytesWithVars(ctx, inputJSON, bound)
	if err != nil {
		return nil, fmt.Errorf("evaluate JSONata: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrJSONUndefined
	}
	// Validate the result domain before generic serialization can turn a
	// function/undefined member into apparently valid JSON.
	value, err := JSONValue(raw)
	if err != nil {
		return nil, err
	}
	value, err = evaluator.PrepareJSONStrings(value)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("JSONata result: %w", err)
	}
	encoded := bytes.TrimSuffix(output.Bytes(), []byte{'\n'})
	if len(encoded) > e.options.MaxOutputBytes {
		return nil, fmt.Errorf("JSONata output exceeds byte budget")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return encoded, nil
}
