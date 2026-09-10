# Maintenance ownership

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
