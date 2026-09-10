# Candidate evaluator integration

Private implementation candidate, not a published release. The package's
existing `jsonata` name is used only in local consumer archives; it is not an
approved public fork identity. Full qualification remains in progress.

## Choose an execution boundary

`jsonata/json-executor` supplies a closed JSON-text boundary for Node and browser
composition. `jsonata/node-executor` adds an optional Node 18+ worker pool.
Both expose `evaluate(expression, inputJSON, {bindingsJSON, signal})` and return
JSON text. No OpenBindings dependency, OBI selector, host-function registration
or mutable global numerical context is introduced by either wrapper.

Both closed wrappers always select the documented standard library. The
upstream `$clone()` embedding helper is explicitly nonstandard and is absent
there, including inside `$eval`. The documented transform operator still makes
an internal deep copy; a local variable called `clone` cannot replace that
internal operation. Authors may declare ordinary lambdas with that name.
The general `jsonata()` API retains its upstream compatibility hook unless the
host selects `standardLibraryOnly: true`; this option is not a sandbox for that
API's separate host-function registration facility. It is not an OBI option or
a second numerical policy.

```js
const {createNodeExecutor} = require('jsonata/node-executor');
const executor = createNodeExecutor({workers: 2, maxPending: 32, timeout: 2000});
try {
    const outputJSON = await executor.evaluate(
        '{"id": id, "total": 0.1 + 0.2}',
        '{"id":9223372036854775807}'
    );
    // {"id":9223372036854775807,"total":0.3}
} finally {
    await executor.close();
}
```

The application owns one reusable executor and its teardown. Do not construct
and destroy a worker pool per operation. Idle workers do not keep Node alive.
Keep exact data as JSON text or use an exact JSON parser for the returned text:
native `JSON.parse` can lose digits independently of evaluator correctness.
Digits lost before input reaches this boundary cannot be recovered.

For SDK composition, pass the executor to the optional
`@openbindings/invoke/jsonata` `createJSONataEvaluator` adapter and supply that
evaluator through the ordinary invoker constructor. The generic SDK does not
import or select this runtime. Other evaluator adapters remain possible; their
numerical and resource guarantees are their own. An adapter must recognize its
engine's JSON values before serialization: some engines serialize functions
as empty strings, which is not a valid JSON-valued transform result.

## Values and failures

The official candidate preserves assigned finite decimal values on carriage
and on operations that do not intentionally change them. Exact supported
finite-decimal arithmetic stays exact; nonterminating division is assigned
once to 34 significant digits, nearest/ties-to-even. This is not a cap on
carried values, an exact-transcendental promise, or a Core conformance floor.
The separate implementation contract and function-decision record define the
function-specific arithmetic domain. There is no precision dial or lossy retry.

Null is a JSON result. An absent result, functions (including nested functions),
cycles, nonfinite numbers and host objects fail before serialization with
`U_JSON_VALUE`. Raw syntax and language errors retain their own categories;
resource rejection is not silently converted to null or a different value.
Bindings, when supplied, must be a JSON object containing JSON values only.
Member names such as `isLosslessNumber` and `_jsonata_function` remain data.

Default full upper/lower casing uses pinned Unicode 16.0.0 data in both native
implementations, independently of the host's Unicode version. It is locale-
independent and performs no normalization. In the source tree,
`unicode/SOURCE.json` identifies the data and license;
`scripts/generate-casing.cjs` reproduces the generated tables
from source-hash-verified inputs. This is implementation versioning, not a new
Core Unicode requirement. The same data supplies decimal-digit families for
integer and date pictures, including supplementary-plane digits. Default sort
and order-by use the documented Unicode codepoint order; string comparisons
use that same ordering. This differs from the raw JS reference for strings
whose UTF-16 and codepoint order disagree. Distributed packages include the Unicode license and
input fingerprints in `THIRD_PARTY_NOTICES.md`; regeneration uses the source
checkout, not undocumented files assumed present in a consumer archive.

Date functions use one millisecond timestamp per evaluation and the existing
Date-compatible range ±8640000000000000 ms. Default interchange uses Gregorian
ISO calendar dates, including signed six-digit expanded years; no year digits
are discarded. Parsing admits reduced calendar forms and compact numeric
offsets, defaults an absent timezone to UTC, truncates submillisecond fractions,
and rejects invalid calendar/offset fields. Picture formatting remains separate
from ISO interchange. The documented timezone argument is signed HHMM; the
reference's `0000` UTC spelling is retained. Ambient named timezone databases
are not consulted. Fractional pictures use F&O's fractional-digit placement and
explicit truncation rules, not ordinary integer padding. Negative years retain
their sign in ISO interchange; the picture `Y` component is the absolute year
as required by F&O. Picture fields reject host-integer overflow. For the
implementation-defined AM/PM representation, the candidate retains the
reference's case selection without width truncation. Ambiguous abbreviated
month/weekday names retain its last-calendar-entry lookup convention; use full
or unambiguous names when reversible date text is needed. These are evaluator
conventions, not Core requirements. Overall qualification remains in progress.

As documented by the incorporated language, base64 encoding takes byte-valued
characters (0–255), while decoding interprets bytes as UTF-8 text. These are
not inverse operations for arbitrary byte strings. Invalid base64, malformed
UTF-8, or an encode character outside the byte range fails explicitly instead
of truncating characters or silently replacing text.

## Immutable host budgets

These defaults are candidate policies, not universal application SLOs. Two
sufficient budgets must produce the same assigned deterministic result. A
smaller budget can reject; it cannot lower precision or select another engine.

| Option | Default | Unit and enforcement |
| --- | ---: | --- |
| `timeout` | 1000 | Milliseconds; cooperative in-process, wall deadline including queue/startup in the Node pool |
| `stack` | 100 | Evaluator stack budget, not a parser-depth limit |
| `sequence` | 10000000 | Evaluator sequence-growth budget |
| `cacheSize` | 64 | Compiled expressions per executor or worker; FIFO |
| `maxExpressionLength` | 262144 | UTF-16 code units in expression text, checked before compilation/worker copy |
| `maxInputLength` | 8388608 | Combined input and bindings JSON UTF-16 code units, checked before parsing/worker copy |
| `maxOutputLength` | 8388608 | Serialized output JSON UTF-16 code units, checked after serialization |
| `numericWork.maxDigits` | 4096 | Arithmetic coefficient work bound; integer 1–100000 |
| `numericWork.maxExponent` | 4096 | Arithmetic adjusted-exponent magnitude bound; integer 1–100000 |
| `workers` (Node only) | 2 | Worker count, integer 1–32 |
| `maxPending` (Node only) | 64 | Active plus queued evaluations, not bytes |
| `maxWorkerHeapMB` (Node only) | 128 | V8 old-generation heap per worker; minimum 16 MiB, not process RSS |

The closed result walk additionally limits depth to 512 and visited values to
one million. Padding and date-picture widths have a 10000-character allocation
ceiling.
Those internal ceilings are not numerical approximation controls. Output-size
checks occur after materialization and cannot by themselves prevent its memory
allocation. A worker heap budget is not a process memory or operating-system
sandbox guarantee; native allocations and parent queue strings also cost memory.

## Cancellation and containment

The underlying Promise API remains `compiled.evaluate(input, bindings, {signal})`;
the upstream callback overload remains available. A signal belongs to one call,
not an expression, binding or input value. Underlying cooperative cancellation
uses `AbortError`/`ABORT_ERR`; wrapper prechecks and Node cancellation preserve a
supplied caller reason. Do not depend on identical error prose across layers.

Signal-bearing evaluation checks before AST evaluation and delivery. Every 256
AST entries it checks whether 8 ms have elapsed since the last timer yield.
These scheduling constants are not a cancellation-latency guarantee. No-signal
in-process evaluation does not schedule such yields. It cannot preempt a
synchronous regex, codec, compilation step or arbitrary host callback.

The Node pool starts no evaluator in the parent. Running-job cancellation,
wall deadline (`U_TIMEOUT`) and close terminate the selected worker; the task
settles after the termination operation, rather than abandoning a background
Promise. Cancellation of queued work removes it without execution. Worker error
and exit handlers remain installed through termination. Rejected work does not
reuse an affected worker or reduce successful-value accuracy.

Use the Node pool for CPU isolation when synchronous expression work must not
block the application's event loop. The browser bundle is cooperative; hosts
requiring hard browser cancellation must provide a separately qualified Worker
boundary. Browser bundle execution alone is not such a qualification.

## Build and platform evidence

The single build pipeline transpiles complete dependency graphs and emits the
base evaluator and closed executor, modern/ES5 and minified/unminified variants,
plus separate optional Node worker entries. Artifact hashes and dependency
licenses accompany the archive. ES5 parsing evidence does not imply that every
ES5 host supplies required built-ins such as Promise. Actual supported-runtime,
type-consumer and security qualification is tracked separately; do not infer it
from a successful build on a newer Node version.
