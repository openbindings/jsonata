# JSONata runtime for JavaScript and TypeScript

Private candidate: `@openbindings/jsonata`. No registry release is
claimed. This package needs no OpenBindings documents, SDKs or services.

```typescript
import { createJSONExecutor } from "@openbindings/jsonata";

const executor = createJSONExecutor();
const outputJSON = await executor.evaluate(
  '{"id":id,"total":0.1+0.2}',
  '{"id":9007199254740993}',
);
// JSON contains id 9007199254740993 and total 0.3.
```

Use JSON text to retain values that native JSON.parse would round. Input,
optional `bindingsJSON`, and successful output remain ordinary JSON. Variable
names in bindings omit the dollar sign. No host functions or callback objects
are admitted. Undefined, invalid nested results and insufficient budgets reject
the promise. `null` is a successful JSON result, not undefined.

```typescript
import { createNodeExecutor } from "@openbindings/jsonata/node";

const executor = createNodeExecutor({ workers: 2, timeout: 1000 });
const abortController = new AbortController();
try {
  await executor.evaluate("$limit", "null", {
    bindingsJSON: '{"limit":9007199254740993}',
    signal: abortController.signal,
  });
} finally {
  await executor.close();
}
```

The in-process executor supports cooperative cancellation. The optional Node
18+ pool isolates evaluator execution in bounded workers; importing its entry
does not initialize the evaluator in the parent. Owners must await `close()`.
Worker heap limits are not process RSS limits. Neither executor is a complete
security sandbox for arbitrary hostile workloads.

Options control resource budgets, not precision modes. Defaults are a
one-second timeout, 64 cached expressions, 262144 expression and 8388608
combined input/bindings/output text limits (each text limit counts UTF-16 code
units), stack depth 100, sequence bound 10000000, and numeric work 4096
digits/exponent magnitude. See [IMPLEMENTATION.md](IMPLEMENTATION.md).

Exports are the closed executor at the package root and the optional `/node`
executor. Private engine helpers, host-function registration and raw upstream
APIs are not supported package exports. Browser bundles expose
`jsonataExecutor.createJSONExecutor`; modern, minified and ES5-syntax variants
are included. ES5 syntax alone is not proof of support for every legacy host:
required built-ins, including Promise, must be available.

The backend derives from [jsonata-js](https://github.com/jsonata-js/jsonata).
This is a distinct implementation artifact, not an upstream release. Original
licenses and bundled dependency notices are retained. Exact carriage and
documented numerical choices are this implementation's policy, not new
OpenBindings or JSONata language requirements.

Build with `npm ci && npm test`; the native suite uses a local HTTP fixture.
`npm pack` produces an archive but does not publish. Public publication is
disabled in this development checkout.
