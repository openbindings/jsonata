# JSONata

Embeddable JSONata execution for Go and JavaScript, maintained by the
OpenBindings Project. One repository, independently consumable language packages:

- Repository identity: `openbindings/jsonata`.
- Go module: `github.com/openbindings/jsonata/go`.
- JavaScript/TypeScript package: `@openbindings/jsonata`.

Private development candidate: two independently consumable artifacts, one
implementation contract. Nothing here requires OpenBindings documents, SDKs or
services. The package namespace identifies the publisher, not a dependency.
The selected names and `0.0.0-dev` versions do not imply a repository or registry
release has been published.

- `go/`: a concurrent, closed JSON-text executor and syntax-only validator.
- `javascript/`: an in-process executor for JS/TS and browsers, plus an optional
  Node worker pool with an explicit lifetime.
- `contract/`: implementation policy and shared acceptance vectors.
- `maintenance/`: source provenance, bounded upstream corrections, inventory
  checking and reproducible local artifact production.

Input/output is JSON text so an application need not adopt a custom in-memory
number type. Bindings are optional JSON objects. Success is one JSON value;
undefined, functions, invalid nested values and exceeded budgets are errors.
Hosts choose work budgets, not a second precision mode. See
[the implementation contract](contract/IMPLEMENTATION.md) for exact carriage,
arithmetic assignment, approximation limits and what fidelity does not mean.

The upstream-derived engines are private implementation details. Existing raw
engine APIs, streaming helpers, host-function registration, browser playgrounds
and upstream branding are not automatically supported APIs of these artifacts.
The retained tests and notices belong to their credited upstream projects.

The OpenBindings SDK adapters sit above this family: they convert SDK values,
enforce allowed transform placement/bindings, and translate runtime failures to
invocation errors. They may also accept other evaluators. Stronger project
implementation tests are not a test of general Core conformance. Graph is not
migrated by this package extraction.

## Qualification and maintenance

Run `node maintenance/inventory.mjs`, `go test -race ./...` from `go/`, and
`npm ci && npm test` from `javascript/`. The JS suite includes a localhost HTTP
fixture. Run `node maintenance/qualify.mjs` for both public executor boundaries.
See [maintenance](maintenance/README.md) for reproducibility and adoption gates.
No package or repository is published by these commands.
