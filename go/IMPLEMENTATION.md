# Runtime implementation contract

This governs this runtime family, not OpenBindings Core, an OBI format, a
binding specification or upstream JSONata. An application can use the runtime
without OpenBindings. Passing the stronger policy vectors does not establish
general language conformance; failing them alone does not prove a third-party
implementation nonconformant.

## Language and closed boundary

Retain the documentation-based JSONata 2.1 language, including individually
incorporated function authorities. Do not add expression syntax, functions,
host callbacks or a precision selector. The upstream-derived private engines
may contain compatibility APIs not exposed by the public executors.

Each call admits one JSON input and optional JSON-object variable bindings,
whose names omit `$`. It returns one JSON value, or fails. Undefined is distinct
from null. Functions, regex/matcher values and invalid nested values cannot be
disguised as objects/null or silently omitted by serialization. Language-defined
sequence flattening and deliberate field omission remain language operations.

Admission consumes the complete JSON text and rejects duplicate decoded member
names within an object, even when their values are equal, recursively in input
and bindings. External binding names beginning with `$` are rejected; dollar-named
data fields and language-local variable shadowing remain available. These are
this runtime family's admission rules, not additional Core or JSONata requirements.

Preserve numerical value, admitted string code units, type, presence and array
order when selecting/copying/rearranging values. This is not preservation of
JSON spelling, whitespace, object-member order, discarded fields, or digits
already lost by a caller's native parser. Valid member names, including names
resembling implementation markers or prototype properties, remain ordinary data.

## Assigned numerical values

Finite JSON decimal tokens and expression literals denote their exact decimal
values. Comparisons, truth, numeric membership, deep-equality leaves, sorting
and serialization share those values, irrespective of their origin.

Addition, subtraction, multiplication, negation, absolute value, accumulation
and positive integral powers are exact within available work budgets. Remainder
is `x - trunc(x/y) * y`. Explicit floor, ceiling and rounding retain their
documented meaning, including ties-to-even for `$round`.

Terminating division is exact. Nonterminating division assigns 34 significant
decimal digits, nearest/ties-to-even, once. Average uses exact sum and this
division; negative integral powers use this division. The assigned rounded
decimal then compares and serializes as itself, not a hidden rational.

Exactly terminating square roots are exact. Other square roots assign 34
significant digits, nearest/ties-to-even. Powers with exponent `0.5` use that
root operation. Other nonintegral powers compare computations at fixed 80 and
120 guard digits after assignment to 34 digits; disagreement fails. This is a
stability check, **not proof of correctly rounded powers for every input**.
There is no binary64 fallback. Resource exhaustion is an error, not zero,
saturation, null, a string, or a successful approximate fallback.

Formatting, indexes, counts, dates, regex and string operations obey their own
contracts. Deliberate truncation/rounding of a control argument is not lossless
carriage. Unicode casing is pinned to Unicode 16.0.0 without locale tailoring.
String ordering uses codepoints where required. Composite membership retains
the reference identity convention, distinct from deep equality.

These are implementation choices where upstream leaves room. They do not claim
that raw upstream JS reference results are universally identical, or that all
transcendental arithmetic is exact. The retained function decision record and
source-pinned expectation overlays explain intentional differences.

## Resource and execution contract

Budgets belong to reusable executor instances; inputs, bindings and cancellation
belong to calls. Compile caches retain expressions, not request values. Sufficient
budgets do not change successful assigned values. Numeric work defaults to
4096 digits/exponent magnitude; configurable work limits are not precision modes.
Other fixed internal allocation/depth guards remain documented limitations.

Go defaults: 256 KiB expression, 8 MiB combined input/bindings, 8 MiB output,
64 cached expressions, one-second cooperative timeout. Sizes count bytes.
JavaScript text limits count UTF-16 code units; see `JSONExecutorOptions` and
`execution-options.js` for defaults. These host-specific budget units do not
change language values and need not admit identical workloads.

Go context and in-process JS cancellation are cooperative: compilation,
serialization and individual synchronous builtins may finish current work
before observing cancellation. These are not hard CPU/memory sandboxes. The
optional Node executor uses bounded workers and queues, terminates a worker on
deadline/cancellation, and must be closed by its owner. No cancellation token,
host policy or resource option is inserted into the expression's variable space.

Random/time-dependent functions are not required to return equal values across
separate invocations or implementations. Their domain and per-evaluation clock
properties are tested separately from deterministic result parity.

## Distribution boundary

The standalone artifacts contain their licenses, private implementation and
required dependencies; they must build without SDK sources. Public executors
do not expose native host-object admission. SDK adapters own native value
conversion and protocol/operation error mapping. A later publicly distributed
SDK cannot depend on an inaccessible private runtime release: either publish
the prerequisite first or retain the entire candidate privately until ready.
