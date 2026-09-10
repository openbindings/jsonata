# JSONata

Embed JSONata transformations in Go, JavaScript, TypeScript and browser
applications, with value-preserving JSON input and output.

Maintained by the **OpenBindings Project**, but usable on its own: no
OpenBindings documents, SDKs or services are required. This repository contains
two native implementations with a shared implementation contract and test corpus.

| Package | Use it for |
| --- | --- |
| [`github.com/openbindings/jsonata/go`](go/README.md) | Go applications, with concurrent executor reuse and context cancellation |
| [`@openbindings/jsonata`](javascript/README.md) | JavaScript, TypeScript and browser applications |
| [`@openbindings/jsonata/node`](javascript/README.md#node-worker-pool) | An optional Node.js worker pool with deadlines and an explicit lifetime |

## Status and installation

This is a **pre-release source repository**. There are no tagged package releases
or npm registry releases yet. The `0.0.0-dev` package version is a development
placeholder, not a published release. You can build and use the packages locally
now; you do not need the rest of the OpenBindings repositories.

```sh
git clone https://github.com/openbindings/jsonata.git
```

For JavaScript/TypeScript, build a local package using Node.js 22.23.2:

```sh
cd jsonata/javascript
npm ci --ignore-scripts --no-audit --no-fund
npm pack
```

`npm pack` builds the bundles and creates `openbindings-jsonata-0.0.0-dev.tgz`;
it does not publish anything. From your application's directory, install that
archive (substitute your checkout path):

```sh
npm install /absolute/path/to/jsonata/javascript/openbindings-jsonata-0.0.0-dev.tgz
```

For Go, see the [local module setup](go/README.md#installation). It uses a source
checkout until tagged releases are available. The module requires Go 1.25.6 or
later; maintenance qualification uses Go 1.25.13.

## JavaScript and TypeScript

Save this as `example.mjs` in your application and run `node example.mjs`:

```javascript
import { createJSONataExecutor } from "@openbindings/jsonata";

const executor = createJSONataExecutor();
const outputJSON = await executor.evaluate(
  '{"id":id,"total":price * quantity}',
  '{"id":9007199254740993,"price":0.1,"quantity":3}',
);

console.log(outputJSON);
// {"id":9007199254740993,"total":0.3}
```

The same import has TypeScript declarations. CommonJS consumers can use
`require("@openbindings/jsonata")`. See the [package guide](javascript/README.md)
for variable bindings, browser bundles and the optional Node worker pool.

## Go

After the [local module setup](go/README.md#installation), save this as `main.go`
in your application and run `go run .`:

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

Reuse the executor across calls, including concurrent Go calls. The separate
[`/syntax` package](go/README.md#syntax-validation) checks expression grammar
without loading the evaluator.

## What the boundary guarantees

Each evaluation accepts an expression, one JSON value as text, and optional
JSON-object variable bindings. A successful result is one JSON value as text.
This keeps the public API independent of any application's in-memory number type.

Input and bindings must be complete JSON texts without duplicate object-member
names, including equal-valued duplicates. Binding names omit the `$` prefix.
This pre-release admission tightening replaces formerly inconsistent duplicate
handling; it is a runtime policy, not an OpenBindings Core requirement.

- Selecting, copying and rearranging values preserves their numerical values,
  admitted string code units, types, presence and array order.
- Large integers and decimal inputs are not silently converted through binary64.
  Exact decimal operations include addition, subtraction and multiplication.
- `null` is a valid result. Undefined results, functions and invalid nested
  values are errors, not values silently dropped or replaced with null.
- Resource limits are configurable. Exceeding them fails the evaluation rather
  than choosing a lower-precision result.

**Value preservation is not byte-for-byte preservation or unlimited exact
arithmetic.** Whitespace, number spelling and object-member order can change.
Transforms may deliberately discard or round values. Nonterminating division and
some other operations have documented numerical assignments and limits. See the
[implementation contract](contract/IMPLEMENTATION.md) before relying on a
particular arithmetic guarantee.

Pass the original JSON text when fidelity matters. Digits already lost by a
caller's parser cannot be recovered, and subsequently parsing the output with
native JavaScript `JSON.parse` can round large numbers again.

## Compatibility and execution limits

These are distinct implementation packages derived from
[jsonata-js](https://github.com/jsonata-js/jsonata) and
[gnata](https://github.com/recolabs/gnata), not upstream releases or drop-in
replacements for their host-object APIs. They retain JSONata expression syntax
and the documented language; the [contract](contract/IMPLEMENTATION.md) records
this family's numerical and execution choices. Identical numeric results with
the upstream JavaScript implementation are not promised.

Only the documented executors and Go syntax validator are public interfaces.
Host-function registration, raw engine helpers and streaming APIs are not exposed.
No OpenBindings-specific variables or protocol concepts are introduced.

The default evaluation timeout is one second, with bounded input, output, cache
and numeric work. Go and in-process JavaScript cancellation are cooperative;
the optional Node 18+ pool can terminate workers. Neither is a complete security
sandbox. See the language guides for lifecycle and resource options. Use maintained
host runtimes in production; older-host compatibility tests are not a security
support promise.

## Development and maintenance

`go/` and `javascript/` are independently consumable packages. `contract/` holds
their shared policy and acceptance vectors; `maintenance/` owns provenance,
upstream corrections, artifact checks and qualification.

See the [maintenance guide](maintenance/README.md) for the pinned toolchain and
commands. Local qualification includes native suites, cross-language vectors,
resource tests and fresh consumers of actual package archives. This is not a
claim that hosted CI or a release gate has passed. Known static-analysis findings
and development-dependency dispositions are recorded in that guide.

## License and attribution

The language packages are MIT-licensed: [Go license](go/LICENSE),
[JavaScript license](javascript/LICENSE). Retained upstream code, tests and
third-party components keep their original notices. JavaScript archives include
generated bundled-dependency notices; Go retains dependency and vendored notices.
See [source provenance](maintenance/INPUTS.json) for the extraction records.
