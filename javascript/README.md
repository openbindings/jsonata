# JSONata for JavaScript and TypeScript

Embed JSONata using `@openbindings/jsonata`. JSON text goes in and comes out;
large integers and decimal values need not pass through JavaScript's native
number representation. No OpenBindings documents, SDKs or services are required.

## Installation

This is pre-release source, not a published npm release. Build a local archive
with Node.js 22.23.2:

```sh
git clone https://github.com/openbindings/jsonata.git
cd jsonata/javascript
npm ci --ignore-scripts --no-audit --no-fund
npm pack
```

From your application directory, install the archive (substitute your path):

```sh
npm install /absolute/path/to/jsonata/javascript/openbindings-jsonata-0.0.0-dev.tgz
```

The archive includes the built executors, TypeScript declarations, browser
bundles and notices. No sibling repository or build tool is required by the
installed package. `npm pack` does not publish; registry publication remains
disabled in the development manifest.

## Evaluate a transform

Save this as `example.mjs` and run `node example.mjs`:

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

The same import works in TypeScript. CommonJS applications can use
`const { createJSONataExecutor } = require("@openbindings/jsonata")` inside an
async function. Reuse the executor across calls.

## Bindings and results

Use JSON text to retain values that native JSON.parse would round. Input,
optional `bindingsJSON`, and successful output remain ordinary JSON. Variable
names in bindings omit the dollar sign. No host functions or callback objects
are admitted. Undefined, invalid nested results and insufficient budgets reject
the promise. `null` is a successful JSON result, not undefined.

Input and bindings must contain no duplicate decoded object-member names, even
equal-valued duplicates. This pre-release admission tightening replaces inconsistent
duplicate handling. Dollar-prefixed external binding names are rejected; dollar-named
data fields and language-local variable shadowing remain available.

Value preservation does not mean preserving whitespace or number spelling, and
does not make every arithmetic operation exact. See the numerical assignments in
[IMPLEMENTATION.md](IMPLEMENTATION.md). Parsing the output with native
`JSON.parse` may lose precision again; keep JSON text or use a suitable parser.

## Node worker pool

For Node.js 18+ applications that need evaluator work off the main thread:

```javascript
import { createNodeExecutor } from "@openbindings/jsonata/node";

const executor = createNodeExecutor({ workers: 2, timeout: 1000 });
const abortController = new AbortController();
try {
  const outputJSON = await executor.evaluate("$limit", "null", {
    bindingsJSON: '{"limit":9007199254740993}',
    signal: abortController.signal,
  });
  console.log(outputJSON); // 9007199254740993
} finally {
  await executor.close();
}
```

The in-process executor supports cooperative cancellation. The optional Node
18+ pool isolates evaluator execution in bounded workers; importing its entry
does not initialize the evaluator in the parent. Owners must await `close()`.
Worker heap limits are not process RSS limits. Neither executor is a complete
security sandbox for arbitrary hostile workloads.

## Resource options

Options control resource budgets, not precision modes. Defaults are a
one-second timeout, 64 cached expressions, 262144 expression and 8388608
combined input/bindings/output text limits (each text limit counts UTF-16 code
units), stack depth 100, sequence bound 10000000, and numeric work 4096
digits/exponent magnitude. See [IMPLEMENTATION.md](IMPLEMENTATION.md).

The Node worker pool accepts timeouts from 1 through 2147483647 milliseconds.
Larger values are rejected at construction, before workers or timers are created.

## Browser use and public exports

Exports are the closed executor at the package root and the optional `/node`
executor. Private engine helpers, host-function registration and raw upstream
APIs are not supported package exports. Browser bundles expose
`jsonataExecutor.createJSONataExecutor`; modern, minified and ES5-syntax variants
are included. ES5 syntax alone is not proof of support for every legacy host:
required built-ins, including Promise, must be available.

For a script-tag application, serve the installed package's `json-executor.js`
or `json-executor.min.js` as a static asset and use
`jsonataExecutor.createJSONataExecutor()` in your browser code. The matching
`json-executor-es5.js` and `json-executor-es5.min.js` variants use ES5 syntax.
Do not load the Node worker entry in a browser. Use maintained hosts in
production; legacy compatibility is not a security support promise.

## Development and attribution

The backend derives from [jsonata-js](https://github.com/jsonata-js/jsonata).
This is a distinct implementation artifact, not an upstream release. Original
licenses and bundled dependency notices are retained. Exact carriage and
documented numerical choices are this implementation's policy, not new
OpenBindings or JSONata language requirements.

After installing dependencies, run `npm test` for the native suite, full
coverage checks and a build. The suite uses a local HTTP fixture. The
[family maintenance guide](https://github.com/openbindings/jsonata/blob/main/maintenance/README.md)
covers cross-language qualification and fresh archive consumers.

MIT-licensed; see [LICENSE](LICENSE) and the bundled `THIRD_PARTY_NOTICES.md`.
