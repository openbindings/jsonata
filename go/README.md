# JSONata runtime for Go

Private candidate module: `github.com/openbindings/jsonata/go`.
No release is published by this checkout. The module depends on no OpenBindings
SDK and accepts/returns JSON text rather than imposing a host number carrier.

```go
import (
    "context"
    jsonata "github.com/openbindings/jsonata/go"
)

executor, err := jsonata.New(jsonata.Options{})
if err != nil { return err }
output, err := executor.Evaluate(context.Background(),
    `{"id":id,"total":0.1+0.2}`,
    []byte(`{"id":9007199254740993}`), nil)
// output is JSON containing id 9007199254740993 and total 0.3.
```

Reuse an executor concurrently. Pass a context for cancellation or a caller
deadline. Optional bindings are a JSON object, e.g. `[]byte("{\"limit\":3}")`;
the expression sees `$limit`. Nil bindings mean none. Functions and host objects
cannot be bound. `errors.Is(err, jsonata.ErrUndefined)` distinguishes absent
results from successful JSON null; context cancellation remains identifiable
with `errors.Is`. Other diagnostic strings are not a stable machine interface.

`jsonata.Options` controls expression/input/output byte budgets, cache capacity,
timeout and numeric work. Zero fields select documented defaults. There is no
lossy mode. The timeout is cooperative, not a hard CPU/memory sandbox. See
[IMPLEMENTATION.md](IMPLEMENTATION.md) for exact arithmetic, approximation,
carriage and resource boundaries. These guarantees require the original JSON
text; digits lost before this call cannot be reconstructed.

Import the `/syntax` package when only checking expression grammar. It does not
initialize the evaluator. Everything below `/internal` is unsupported backend
or qualification machinery, not part of the application's API.

Run `go test -race ./...` from this module. Upstream-derived native tests and
project policy tests remain distinct. The family repository also runs a shared
public-boundary corpus against both languages and real archive consumers.
