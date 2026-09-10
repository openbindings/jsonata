# JSONata for Go

Embed JSONata through a concurrent, value-preserving JSON-text executor.
Module: `github.com/openbindings/jsonata/go`. No OpenBindings SDK is required.

## Installation

This is pre-release source; no tagged Go release is available yet. Requires Go
1.25.6 or later. Clone the standalone repository:

```sh
git clone https://github.com/openbindings/jsonata.git
```

In your application's Go module, use a local replacement for the checkout
(substitute the absolute path). If starting a new application, first run
`go mod init example.com/myapp`.

```sh
go mod edit -replace github.com/openbindings/jsonata/go=/absolute/path/to/jsonata/go
go get github.com/openbindings/jsonata/go@v0.0.0-dev
```

Here `v0.0.0-dev` selects the locally replaced module; it does **not** name a
published release. This setup needs no sibling repositories or Go workspace.

## Evaluate a transform

Save as `main.go` in your application and run `go run .`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    jsonata "github.com/openbindings/jsonata/go"
)

func main() {
    executor, err := jsonata.New(jsonata.Options{})
    if err != nil {
        log.Fatal(err)
    }
    output, err := executor.Evaluate(context.Background(),
        `{"id":id,"total":price * quantity}`,
        []byte(`{"id":9007199254740993,"price":0.1,"quantity":3}`), nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(output))
    // JSON containing id 9007199254740993 and total 0.3.
}
```

Reuse an executor concurrently. Input and output are JSON bytes; returned output
is caller-owned. Preserve the original input text: digits lost by a caller's
parser cannot be reconstructed.

## Bindings, errors and cancellation

The fourth argument to `Evaluate` is optional JSON-object variable bindings.
For example, the JSON bytes `{"limit":3}` bind `$limit`; nil means no bindings.
Names omit the dollar sign. Functions and host objects cannot be bound.

`errors.Is(err, jsonata.ErrUndefined)` distinguishes absent results from
successful JSON `null`. Invalid result values and exceeded budgets also fail.
Context cancellation remains identifiable with `errors.Is(err, context.Canceled)`.
Other diagnostic strings are not a stable machine interface.

Pass a context for caller cancellation or deadlines. `jsonata.Options` controls
expression/input/output byte budgets, cache capacity, timeout and numeric work;
zero fields select defaults. Defaults include a one-second cooperative timeout,
256 KiB expression, 8 MiB combined input/bindings, 8 MiB output and 64 cached
expressions. This is not a hard CPU/memory sandbox.

There is no lossy mode. Exact carriage does not mean every arithmetic operation
is exact: see [IMPLEMENTATION.md](IMPLEMENTATION.md) for assigned arithmetic,
approximation, resource and value-preservation boundaries.

## Syntax validation

Import `github.com/openbindings/jsonata/go/syntax` and call
`syntax.Validate(expression)` for grammar checking without initializing the
evaluator. A successful syntax check does not guarantee successful evaluation.
Everything below `/internal` is private implementation or qualification
machinery, not an application API.

## Development and license

Run `go test -race ./...` from this module. Upstream-derived native tests and
project policy tests remain distinct. The [family maintenance guide](https://github.com/openbindings/jsonata/blob/main/maintenance/README.md)
covers shared public-boundary tests and fresh module-archive consumers.

MIT-licensed; see [LICENSE](LICENSE). The backend derives from
[gnata](https://github.com/recolabs/gnata), with original and third-party notices
retained. This is a distinct package, not a gnata release or a drop-in replacement
for its host-object API.
