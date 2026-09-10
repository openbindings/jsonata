# Isolated evaluator candidate

This checkout contains an unadopted, upstream-derived qualification candidate.
It is not a published Gnata release or a claim that the upstream project owns
these changes. Publication, permanent identity and default activation are held.

Use a context with every evaluation; compile once and reuse the expression.
`EvalBytes` preserves admitted JSON number values. Native values cannot recover
digits already lost by a caller. Exact decimals are carried coherently through
the evaluator; numeric work limits reject excessive work without reducing the
precision of successful results. The official SDK contract, not Core, supplies
the stronger fidelity policy. General fractional powers use a qualified
approximation, not exact transcendental arithmetic.

`CompileOptions.StandardLibraryOnly` excludes the reference `$clone` embedding
compatibility helper. The documented transform operator copies internally in
that environment, including in `$eval`; ordinary locally declared lambdas are
still permitted. This flag does not sandbox host custom-function APIs. The
official SDK adapter always selects the documented library and only admits JSON
variables; callers use the generic SDK injection seam for another evaluator.
No document-level language selector or numerical mode is added.

Regex evaluation uses a privately relocated and instrumented Regonaut engine,
not RE2. It supports the qualified ECMAScript regex lane, including lookarounds,
backreferences and UTF-16 indexing. Backtracking consumes bounded engine work;
context cancellation is cooperative. This is not a general hard-CPU or process
memory sandbox. Source, license and replay metadata are in
`internal/thirdparty/regonaut`; `scripts/vendor-regex.mjs` performs relocation.

Default Unicode casing and integer/date decimal-digit families use pinned
Unicode 16 data. Sort and order-by use Unicode codepoint order, without locale
tailoring or normalization. Reproduce the tables without
a JS checkout using `go run scripts/generate-casing.go INPUT_DIRECTORY`; input
identities and their authoritative URLs are in `internal/unicodecase/SOURCE.json`.
The generator verifies all hashes before writing. An optional second argument
selects a separate output directory for byte-for-byte replay verification.

Date pictures use the existing millisecond calendar domain. Integer field
overflow is rejected before host calendar arithmetic; Roman, alphabetic,
word and Unicode decimal forms share the supported integer representations.
AM/PM case selection ignores width as in the reference. For duplicate month
or weekday abbreviations the last calendar entry wins, also as in the reference;
full/unambiguous names are required for a reversible textual date. These narrow
evaluator conventions add no Core or binding requirements.

The full native race suite and cross-language numeric/string/structural tests
are qualification evidence, not a proof for every expression. Complete
date-picture, resource, deployment and end-to-end acceptance remains in progress.
