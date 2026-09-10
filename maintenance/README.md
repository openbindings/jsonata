# Maintenance ownership

## Routine qualification

Use Node 22.23.2 and Go 1.25.13 for maintenance. The linter is
golangci-lint 2.12.2 (upstream binary built with Go 1.26.2). These are build/test
tools, not changed consumer minimums: the root JS artifacts retain the tested
Node 8 surface, and optional Node workers require Node 18+. Obsolete runtimes
belong only in isolated compatibility tests without secrets.

From the repository root, install the locked development graph with
`npm --prefix javascript ci --ignore-scripts --no-audit --no-fund`, build with
`npm --prefix javascript run build`, then use the one entry point:

```
node maintenance/run.mjs go
node maintenance/run.mjs javascript
node maintenance/run.mjs shared
node maintenance/run.mjs resource
node maintenance/run.mjs packages
node maintenance/run.mjs generated
node maintenance/run.mjs dependency
node maintenance/run.mjs static
```

Each invocation writes its own `.maintenance-output/<lane>-…` directory, with
commands, stdout, stderr and exit status. `JSONATA_MAINTENANCE_OUTPUT` may name
a fresh output directory. No audit-folder path or SDK checkout is required.
Set `GOLANGCI_LINT` to the pinned executable if it is not on PATH. Generated
checks fetch checksum-verified upstream inputs, or accept both
`JSONATA_UNICODE_INPUT` and `JSONATA_REGEX_INPUT` cached directories.

The package lane installs real npm/module archives in fresh consumers. It does
not prove a registry release exists. Cross-SDK, browser and CLI integration is
separately owned by the project/consumer repositories; it must accompany adoption.

The root workflow runs non-publishing lanes with read-only repository access.
Raw static analysis intentionally remains nonzero for retained findings; the
separate reviewed-finding reports are not clean lint. Choosing which reports
block shared branches is an explicit maintainer/CI-policy decision, not a waiver
implemented here. No required-check settings have been changed.

## Current dependency and diagnostic dispositions

The test-only `request` chain was replaced with Node HTTP fixtures. The selected
maintenance tool updates are Mocha 12.0.0, nyc 18.0.0, JSDoc 4.0.5 and Browserify
17.0.1. Numerical/runtime dependencies are unchanged. Documentation reads source
files, not duplicate generated bundles; its malformed array-type comment was
corrected. Native tests and all four coverage categories remain mandatory.

The full npm audit is **not clean**: four low-severity package entries trace to
[GHSA-848j-6mx2-7j84](https://github.com/advisories/GHSA-848j-6mx2-7j84), for which
the reviewed upstream advisory has no patched version. They belong to
Browserify's optional cryptography shim. `audit-exposure.mjs` retains both raw
audits and proves that the maintained build does not load those packages and
neither generated runtime graph contains them. No cryptographic key/signature
operation uses them in this project. This is a current-graph nonreachability
finding, not approval to use them elsewhere. Maintainers must revisit on any
dependency, advisory, entry-point or build change; the guard fails on changed
findings or newly reached affected modules. A scanner/network failure is not
reported as zero vulnerabilities. Production audit must remain clean.

`STATIC_FINDINGS.json` records 302 raw lint findings and 25 unsuppressed security
findings, each with source identity and rationale. The added lint item is the
repeated `S0104` literal in new regression fixtures, not an ignored correctness
warning. Raw G101/G404/G115 exclusions in the inherited configuration are not
used for the independent security pass. New, changed, increased or removed
findings, and changed source files, require review. The check cannot silently
refresh its own baseline.

The security review found and repaired malformed signed Unicode escape admission
in the Go lexer. `ParseUint(...,16,16)` rejects signed forms before narrowing;
native and shared regressions preserve JavaScript's existing S0104 rejection.
No valid expression, numeric assignment policy or public API was changed.

## Ownership and upstream updates

This repository owns the runtime façade, upstream-derived private backends,
their integration patches, shared implementation contract and qualification.
SDKs own adapters, not copied evaluator forks. One source change still needs
two native implementations; packaging does not eliminate that semantic cost.

`INPUTS.json` records exact upstream base commits and dirty-candidate source
hashes. Its audit paths are historical provenance, not build dependencies.
`patches/` records three selected maintenance corrections: JS #831 guards,
their Go mirror, and Go #22 empty grouped-join handling. The two associated join
expressions and aggregate registrations are included. The stronger existing
padding allocation guard remains; no blind upstream merge was performed.

Upstream update procedure:

1. Pin candidate upstream commits. Export a new checkout; never overwrite the
   accepted source or update floating branches as part of qualification.
2. Classify each diff as required language correction, compatible implementation
   choice, resource/security fix, mechanical change, or irrelevant upstream API.
   Keep patches mapped to original paths before the mechanical Go import relocation.
3. Add failing-before witnesses and register every referenced expression/dataset.
   Run `inventory.mjs`; missing or unregistered expressions are errors.
4. Regenerate Unicode, regex and syntax copies with their pinned generators and
   source manifests. Never stamp the runtime-family commit as an upstream commit.
5. Run upstream-derived suites plus reviewed expectation overlays, shared public
   boundary tests, independent numeric oracles, race/resource tests and SDK/CLI
   consumers. Retain failures and compare cost/security/static-analysis changes.
6. Build actual archives and install them into fresh consumers without sibling
   links. Go's `pack-go.mjs` creates a file module proxy for this proof; it does
   not publish or supply a source `replace`. Record exact source/artifact hashes.

The native suites preserve language tests separately from project numerical
policy. Existing reviewed Go lint debt and JS development-tool advisories are
not erased by relocation. A green behavior suite is not a green security,
performance, publication or release-activation gate.

The current candidate has no remote repository and npm publication is disabled.
Activation requires reachable dependency artifacts and deliberate release
coordination; no shared specification or release branch is changed here.

`node maintenance/verify-generated.mjs UNICODE_INPUT_DIRECTORY REGONAUT_MODULE_DIRECTORY`
reproduces the JS and Go Unicode assets and instrumented regex source in fresh
temporary directories, checks every output byte, and leaves the candidate intact.
The Unicode generator records source URLs and SHA-256 checksums; the regex input
must be the checksum-verified `github.com/auvred/regonaut@v0.0.1` module directory.
The Go generator can run using Go alone; JS is not a Go runtime dependency.
The SDK's syntax-copy regeneration remains an SDK integration responsibility.
