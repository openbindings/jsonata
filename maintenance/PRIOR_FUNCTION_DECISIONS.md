# Historical function-specific candidate decisions — implementation only

This is a preserved run-3 investigation record, not the current work queue.
Subsequent qualification and maintainer approval settled its cost, dependency
ownership and package-identity questions. Its bare evidence filenames refer to
that historical investigation, not files required to build this repository.
For the current maintained rules and executable gates, use the
[implementation contract](../contract/IMPLEMENTATION.md) and
[maintenance guide](README.md). Publication remains a separate procedure.

This is a mutable engineering record for the isolated run-3 candidates, not a
Core/binding change, an upstream language amendment, or release acceptance.
All public-source references below are evidence; executable reference behavior
does not override documented language meaning.

Current checkpoint: the historical open qualification items below are superseded
by FUNCTION_REVIEW.json, BOUNDARY_REVIEW.md and REPORT.md. Temporal/string/regex,
host-budget and independent root/power checks have now executed; final native,
SDK and CLI evidence is recorded in GATE.json. Cost acceptance and permanent
dependency ownership remain unresolved. Approximation limitations remain in
force; passing tests did not turn guard-digit agreement into a mathematical proof.

## Arithmetic and formatting

- `$sqrt`: an exactly terminating root is preserved exactly. Otherwise assign
  34 significant decimal digits, nearest/ties-to-even. The exponent `0.5` in
  `$power` uses the same root operation. The exact-root check squares the probe
  without rounding; a rounded square is not evidence of an exact root.
- Other nonintegral powers: existing apd/v3 and decimal.js computations at fixed
  80 and 120 guard digits must agree when assigned to 34 significant digits,
  nearest/ties-to-even. Instability or an unavailable work budget fails; there
  is no native floating-point fallback. Agreement is a stability check, NOT a
  mathematical proof of correctly rounded powers for every input. This remains
  a function-specific approximation, not a promise of exact transcendental
  arithmetic. Production cost and difficult-rounding boundaries still need
  final qualification. The fixed guard settings are not caller precision modes.
- `decimal.js` 10.6.0 is an isolated pinned candidate dependency for the latter
  operation. Permanent adoption, package topology and notices are still open.
  It does not replace the exact add/multiply/divide path or change global
  arithmetic-library settings.
- `$formatNumber` retains its existing picture/locale machinery and uses exact
  decimal scaling, half-even rounding and rendering. Exponential scaling occurs
  before rounding, without a new post-round normalization (W3C steps 5–6).
- `$formatBase` uses exact half-even integer rounding, including the radix;
  `$formatInteger` retains the reference's floor convention. Decimal, alphabetic
  and word formatting/parsing retain all integer digits. Roman output remains
  bounded by a host allocation budget; it does not return an approximate number.
- The old word parser's segmentation failed for a larger magnitude following a
  smaller phrase, e.g. "nine thousand and seven trillion". The candidate uses a
  magnitude stack, retaining the existing vocabulary and picture grammar.

## Numeric control arguments

- Numeric predicates floor exactly, then resolve negative indexes from the end.
- `$substring` truncates start and length separately, using the `substr`
  convention referenced by the JSONata string-function documentation. The
  reference's slice-end calculation differs for fractional positions; it is
  not copied into the candidate. Clipping is deliberate control behavior, not
  permission to clamp a language number during carriage or arithmetic.
- Padding deliberately truncates width. Both candidates retain the Go host's
  existing 10,000-character padding-width budget. Its final host configuration
  and general output budget still require resource qualification.
- Match/replace and matcher-based split retain the reference's `count < limit`
  iteration convention, including fractional limits. Literal split uses its
  explicit integer-count conversion. Both use exact comparisons/conversions;
  a huge limit is not wrapped through JavaScript's uint32 conversion.
- `$fromMillis` checks the exact value against the existing Date-compatible
  range ±8,640,000,000,000,000, then deliberately truncates milliseconds toward
  zero. Full timestamp/picture edge cases, especially expanded years and invalid
  timestamp fields, remain open; ordinary date tests alone do not settle them.
- Range stepping may use native arithmetic only after proving both endpoints
  and all intermediate integers are within the exact safe-integer interval.
  Wide integers still use decimal/big-integer stepping. This optimization changes
  cost, not successful results or public policy.

## Evidence and remaining work

`roots-power-independent-2.json`: 489 independent cases in both languages,
978 matches to a libmpdec oracle evaluated at 240 and 480 digits. This is a
bounded numeric test, not full evaluator, SDK or CLI acceptance. The first
oracle attempt incorrectly tested exactness using a rounded square; the
recorded failed run is retained, and the oracle was corrected before the pass.

`go-consumer-witnesses-2.json` and `js-consumer-witnesses-2.json` pass the new
wide-integer, word/alphabetic round-trip, control, order-by and power witnesses,
along with the relevant existing integer-format groups (199 JS tests).
`go-limit-consumers.json` and `js-limit-consumers.json` pass their then-current
selected regression groups; broader fractional/huge limit witnesses remain due.

The first complete JS fixture run recorded 1,664 passes and 22 failures out of
1,686, before the safe-range optimization. Two failures were range timeouts;
20 numerical-policy/error-timing differences now have source-pinned expectation
overlays. `expectation-oracle.py` computes their results independently using
Fraction/Decimal/factorial, never candidate evaluator output. Original fixture
and expression/dataset dependency hashes are checked before applying each delta.
Both full suites now pass with these specific overlays. The historical failed
runs and a reference-expectations observation switch remain available.

Separately, one Go-only regression expected `$distinct([1,1])` to unwrap to 1.
That assertion is corrected to [1], consistent with explicit-array processing
and the JS reference. It is a structural bug correction, not a numeric overlay.
The mirrored 53-case structural corpus includes variable/lambda paths and
distinct empty-container versus selected-container identity.

The tagged-list/callable identity repair passes the full Go race suite and
1,819 JS tests. This does not establish arbitrary native cyclic/DAG input
admission or all resource bounds. Regex/string semantics, clean dependency
packaging, SDK seams, Graph isolation and the final end-to-end gate remain open.

## 2026-09-10 continuation review

The current temporal corpus has 1,026 cases, including every Unicode 16 decimal
family. Numeric picture parsing now checks host bounds before calendar arithmetic.
The failing-before alphabetic witness represented 2^64 + 1970: Go wrapped it to
1970 and JS leaked a host type error. Both now reject it with D3110. Zero/zeroth
years, presentation-plus-width parsing, and Unicode ordinal date inputs also
have regressions. Temporal fields remain bounded controls, not a restriction on
ordinary exact JSON numbers. Go's duplicate numeric-field dispatch was consolidated.

F&O §9.8.4.7 explicitly leaves AM/PM rendering implementation-defined. The
candidate retains the reference's case handling without width truncation. For
ambiguous abbreviated names it retains the reference's last-entry lookup;
neither implementation advertises an ambiguous abbreviation as reversible.
These are implementation conventions, not new language/Core mandates.

Pinned JSONata array-functions and path-operators explicitly specify codepoint
ordering. The raw JS reference instead uses UTF-16 comparison. Eight of twenty
independent ordering witnesses failed before correction; Go passed unchanged.
The JS candidate now shares a codepoint comparator between default sort,
order-by and relational string comparison. The last choice keeps the existing
string-comparison extension coherent; comparison-operator prose itself focuses
on numbers. No locale collation or Unicode normalization was introduced.

Evidence: `temporal-representation-before/after`,
`go-temporal-representation-regressions`, `js-temporal-representation-regressions`,
`js-string-order-before/after`, `go-string-order-current`, and
`js-final-ordinal-and-order`. Full-suite evidence is recorded separately. This
review does not turn reference execution or fixture counts into a specification.

### Higher-order and object-query continuation

Higher-order review found and repaired lost Go native/partial callback arity:
12 of the initial 22 targeted cases failed (24 failures across full/byte entry
points). `$map` passed only values to `$power`/`$round`; `$reduce` accepted unary
callbacks when it should reject them, including empty and singleton inputs.
Callable arity now lives on the internal function value and survives aliases,
object storage, `$eval` and partial application. Registration records native
callback conventions; no function-name or expression-string dispatch is added.
`$each` now honors unary callbacks without an unwanted key argument. The expanded
26-case corpus includes three/four-position lambdas and unary/variadic rejection.
These are ordinary language corrections, not numerical-policy or Core changes.
`$string` and `$zip` retain the reference's callback arities 1 and 0 respectively;
these conventions are not inferred from Go's slice-based host signature.

Concurrent nondeterministic tests now assert `$random` type/range and assigned
value reload, plus `$shuffle` multiplicities, input immutability and wide-value
retention. They deliberately do not require equal random values across calls or
engines, statistical uniformity from a tiny sample, or cryptographic security.

Object-query review found seven Go collection/traversal differences across 18
cases (21 failing full/byte/variable observations). `$lookup` over arrays now
uses the established reference's collection convention, including empty results
and one-level flattening at each array traversal boundary. Direct object-property
lookup preserves its array value unchanged. `$keys` and `$spread` traverse nested
input arrays, without recursively rewriting selected property values. This is
alignment to existing tooling where the short function prose does not specify
all collection details, not a claim that every detail is normative JSONata.
Wide numeric values and legitimate `__proto__` keys are retained. One initially
authored test mistakenly expected `$spread` on a one-object array to unwrap;
both engines retained the array. That harness expectation was corrected under
the explicit-array rule; its original failed record is retained. No upstream
fixture expectation changed. No generic JSON normalization policy was changed.

### Sources

- [JSONata numeric functions](https://docs.jsonata.org/2.1.0/numeric-functions)
- [JSONata string functions](https://docs.jsonata.org/string-functions); the
  versioned 2.1 URL returned 404, so do not mislabel that page as a 2.1 fetch.
- [ECMAScript substr](https://tc39.es/ecma262/multipage/additional-ecmascript-features-for-web-browsers.html#sec-string.prototype.substr)
- [W3C format-number](https://www.w3.org/TR/xpath-functions-31/#func-format-number)
- [decimal.js power](https://mikemcl.github.io/decimal.js/#toPower): documents a
  possible one-ulp rounding error for powers; this is why no universal
  correctly-rounded claim follows merely from selecting the library.
- [apd Context](https://pkg.go.dev/github.com/cockroachdb/apd/v3#Context): pinned
  v3.2.1 source for root/power/context algorithms was also read locally.
