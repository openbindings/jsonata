"use strict";
const assert = require('assert');
const jsonata = require('../src/jsonata');

describe('Per-evaluation time under concurrent compiled-expression reuse', function () {
    for (const clock of ['$millis()', '$now()', '$now("[Y0001]-[M01]-[D01]T[H01]:[m01]:[s01].[f001]")']) {
        it(clock + ' stays stable while another evaluation starts', async function () {
            const expression = jsonata('($before := ' + clock + '; $barrier(); [$before, ' + clock + '])');
            let entered;
            const ready = new Promise(resolve => { entered = resolve; });
            let release;
            const blocked = new Promise(resolve => { release = resolve; });
            // Host instrumentation controls interleaving; it is not exposed by the OB embedding.
            const first = expression.evaluate({}, {barrier: async () => { entered(); await blocked; }});
            await ready;
            await new Promise(resolve => setTimeout(resolve, 15));
            let second;
            try { second = await expression.evaluate({}, {barrier: () => undefined}); }
            finally { release(); }
            const original = await first;
            assert.strictEqual(original[0], original[1], 'another evaluation changed the first clock');
            assert.strictEqual(second[0], second[1]);
            assert.notStrictEqual(original[0], second[0], 'separate evaluations need their own timestamp');
        });
    }
});
