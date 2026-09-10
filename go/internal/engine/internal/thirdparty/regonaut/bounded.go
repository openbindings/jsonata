// OpenBindings candidate resource adapter. Upstream matching semantics remain
// in regonaut.go; this file is not part of regonaut v0.0.1.
package regonaut

import (
	"context"
	"fmt"
)

// Limits affect completion, never the value of a successful match. Each call
// owns its counters; there are no workers or global timeout settings.
type Limits struct {
	Steps                 uint64
	LiveCells, InputUnits int
}

func DefaultLimits() Limits { return Limits{Steps: 1_000_000, LiveCells: 262144, InputUnits: 1 << 20} }

type LimitError struct{ Resource string }

func (e *LimitError) Error() string { return "U_REGEX_LIMIT: " + e.Resource + " budget exhausted" }

type boundedAbort struct{ err error }

func abort(err error) { panic(boundedAbort{err}) }
func recoverBounded(err *error) {
	if p := recover(); p != nil {
		if a, ok := p.(boundedAbort); ok {
			*err = a.err
		} else {
			panic(p)
		}
	}
}

func (c *compiler) checkCompileBudget() {
	if c.boundedDepth > 128 {
		abort(&LimitError{"pattern nesting"})
	}
	if len(c.byteCode) > 16384 {
		abort(&LimitError{"program instructions"})
	}
	if c.capturesCount > 256 {
		abort(&LimitError{"captures"})
	}
}

// CompileBoundedUTF16 is the only production compile entry point of the private
// adapter. Its small host bounds also constrain parser work before execution.
func CompileBoundedUTF16(pattern []uint16, flags Flag) (result *RegExpUtf16, err error) {
	defer recoverBounded(&err)
	if len(pattern) > 16384 {
		return nil, &LimitError{"pattern units"}
	}
	// JSONata exposes only i/m. Annex B is the non-Unicode JavaScript grammar.
	if flags & ^(FlagIgnoreCase|FlagMultiline|FlagAnnexB) != 0 {
		return nil, fmt.Errorf("unsupported JSONata regex flags")
	}
	return CompileUtf16(append([]uint16(nil), pattern...), flags)
}

type executionBudget struct {
	ctx    context.Context
	limits Limits
	steps  uint64
}

func (vm *machine) checkExecutionBudget() {
	b := vm.budget
	if b == nil {
		return
	}
	if b.steps >= b.limits.Steps {
		abort(&LimitError{"steps"})
	}
	if b.steps%128 == 0 {
		if err := b.ctx.Err(); err != nil {
			abort(err)
		}
	}
	b.steps++
	if vm.liveCells() > b.limits.LiveCells {
		abort(&LimitError{"live stack cells"})
	}
}

func (vm *machine) liveCells() int {
	return len(vm.stack) + 2*len(vm.captures) + 2*len(vm.capturesStack) + len(vm.stacksStack) + 10*len(vm.backtrackingStack)
}

func (vm *machine) checkFrameBudget() {
	if vm.budget != nil && vm.liveCells()+2*len(vm.captures)+len(vm.stack)+10 > vm.budget.limits.LiveCells {
		abort(&LimitError{"live stack cells"})
	}
}

// FindBounded runs synchronously. Input and results stay as exact UTF-16 units.
// It never converts a cancelled or exhausted match into an ordinary no-match.
func (r *RegExpUtf16) FindBounded(ctx context.Context, source []uint16, pos int, limits Limits) (result *MatchUtf16, err error) {
	defer recoverBounded(&err)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limits.Steps == 0 || limits.LiveCells <= 0 || limits.InputUnits < 0 {
		return nil, &LimitError{"zero allowance"}
	}
	if len(source) > limits.InputUnits {
		return nil, &LimitError{"input units"}
	}
	if pos < 0 || pos > len(source) {
		return nil, nil
	}
	if source == nil {
		source = []uint16{}
	}
	vm := newMachine(r.c, stringSource{utf16: source, isUtf16: true, pos: pos}, false)
	vm.budget = &executionBudget{ctx: ctx, limits: limits}
	vm.eval()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if vm.notMatched {
		return nil, nil
	}
	m := &MatchUtf16{Groups: make([]GroupUtf16, len(vm.captures))}
	for i, c := range vm.captures {
		m.Groups[i] = GroupUtf16{src: source, Start: c.start, End: c.end}
	}
	return m, nil
}
