// Package functions implements the JSONata 2.x standard library.
package functions

import (
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/parser"
)

// EvalFn is a callback used by higher-order functions to invoke a lambda or
// builtin function value without creating an import cycle. The env parameter
// carries the per-evaluation call counter, ensuring concurrent Eval calls
// don't share stack-depth state.
type EvalFn func(fn any, args []any, focus any, env *evaluator.Environment) (any, error)

// builtinFuncs lists all plain BuiltinFunction registrations (name → func).
var builtinFuncs = []struct {
	name  string
	arity int
	fn    func([]any, any) (any, error)
}{
	// ── String ────────────────────────────────────────────────────────────────
	{"string", 1, fnString},
	{"length", 1, fnLength},
	{"substring", 3, fnSubstring},
	{"substringBefore", 2, fnSubstringBefore},
	{"substringAfter", 2, fnSubstringAfter},
	{"trim", 1, fnTrim},
	{"pad", 3, fnPad},
	{"join", 2, fnJoin},
	{"base64encode", 1, fnBase64Encode},
	{"base64decode", 1, fnBase64Decode},
	{"encodeUrl", 1, fnEncodeURL},
	{"encodeUrlComponent", 1, fnEncodeURLComponent},
	{"decodeUrl", 1, fnDecodeURL},
	{"decodeUrlComponent", 1, fnDecodeURLComponent},
	// ── Numeric ───────────────────────────────────────────────────────────────
	{"number", 1, fnNumber},
	{"random", 0, fnRandom},
	// ── Array ─────────────────────────────────────────────────────────────────
	{"count", 1, fnCount},
	{"append", 2, fnAppend},
	{"reverse", 1, fnReverse},
	{"shuffle", 1, fnShuffle},
	{"distinct", 1, fnDistinct},
	{"zip", 0, fnZip},
	// ── Object ────────────────────────────────────────────────────────────────
	{"keys", 1, fnKeys},
	{"spread", 1, fnSpread},
	{"merge", 1, fnMerge},
	{"error", 1, fnError},
	{"lookup", 2, fnLookup},
	{"clone", 1, evaluator.CloneArgument},
	// ── Boolean ───────────────────────────────────────────────────────────────
	{"boolean", 1, fnBoolean},
	{"not", 1, fnNot},
	{"exists", 1, fnExists},
	// ── Misc ──────────────────────────────────────────────────────────────────
	{"assert", 2, fnAssert},
	{"type", 1, fnTypeOf},
}

func newSignedBuiltin(fn func([]any, any) (any, error), sig string) *evaluator.SignedBuiltin {
	parsed, _ := parser.ParseSig(sig)
	return &evaluator.SignedBuiltin{Fn: fn, Sig: sig, ParsedSig: parsed}
}

// RegisterAll binds every JSONata built-in function into env.
// evalFn must call evaluator.ApplyFunction (supplied by gnata.go).
func RegisterAll(env *evaluator.Environment, evalFn EvalFn) {
	// Arity is explicit language metadata. In particular, $string's prettify
	// default and variadic $zip retain the reference callback conventions (1/0).
	bind := func(name string, arity int, fn evaluator.EnvAwareBuiltin) {
		env.Bind(name, evaluator.NewNativeFunction(fn, arity))
	}
	bind("fromMillis", 3, fnFromMillis)
	bind("now", 2, fnNow)
	bind("millis", 0, fnMillis)
	bind("toMillis", 2, fnToMillis)
	bind("formatInteger", 2, fnFormatInteger)
	bind("parseInteger", 2, fnParseInteger)
	bind("formatBase", 2, fnFormatBase)
	bind("formatNumber", 3, fnFormatNumber)
	bind("sqrt", 1, fnSqrt)
	bind("abs", 1, fnAbs)
	bind("floor", 1, fnFloor)
	bind("ceil", 1, fnCeil)
	bind("round", 2, fnRound)
	bind("power", 2, fnPower)
	bind("sum", 1, fnSum)
	bind("max", 1, fnMax)
	bind("min", 1, fnMin)
	bind("average", 1, fnAverage)
	for _, b := range builtinFuncs {
		env.Bind(b.name, evaluator.NewNativeFunction(evaluator.BuiltinFunction(b.fn), b.arity))
	}
	env.Bind("uppercase", newSignedBuiltin(fnUppercase, "s-:s"))
	env.Bind("lowercase", newSignedBuiltin(fnLowercase, "s-:s"))
	bind("match", 3, makeFnMatch(evalFn))
	bind("contains", 2, makeFnContains(evalFn))
	bind("split", 3, makeFnSplit(evalFn))
	bind("replace", 4, makeFnReplace(evalFn))
	bind("eval", 2, makeFnEval())
	bind("sort", 2, makeFnSort(evalFn))
	bind("sift", 2, makeFnSift(evalFn))
	bind("each", 2, makeFnEach(evalFn))
	bind("map", 2, makeFnMap(evalFn))
	bind("filter", 2, makeFnFilter(evalFn))
	bind("single", 2, makeFnSingle(evalFn))
	bind("reduce", 3, makeFnReduce(evalFn))
}
