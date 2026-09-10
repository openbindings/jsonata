'use strict';
const assert = require('assert');
const {spawnSync} = require('child_process');
const {performance} = require('perf_hooks');
const {createJSONataExecutor} = require('../src/json-executor');
const {createNodeExecutor} = require('../src/node-executor');

for (const [name, factory] of [['cooperative',createJSONataExecutor], ['worker',createNodeExecutor]]) {
    describe('Closed JSON executor: '+name, function() {
        this.timeout(10000);
        let executor;
        beforeEach(() => { executor = factory({timeout:3000}); });
        afterEach(async () => { if (executor.close) await executor.close(); });
        for (const raw of ['9007199254740993','1e+400','1e-400','0.12345678901234567890123456789','null','[[],[1],null]','"\\ud800"','{"\\ud800":"\\udfff"}','{"isLosslessNumber":true,"value":"7","_jsonata_function":true}']) {
            it('carries raw JSON '+raw, async () => {
                const identity = await executor.evaluate('$',raw);
                // Path projection deliberately flattens nested arrays; lookup
                // selects the member as one value for this carriage witness.
                assert.strictEqual(await executor.evaluate('$lookup($eval($string({"value":$})),"value")',raw),identity);
                assert.strictEqual(await executor.evaluate('$',identity),identity);
                if (!raw.includes('e+') && raw !== '1e-400') assert.strictEqual(identity,raw);
            });
        }
        it('computes and selects in the same assigned domain', async () => {
            assert.strictEqual(await executor.evaluate('id = 9007199254740993 ? 0.1+0.2 : 0','{"id":9007199254740993}'),'0.3');
            assert.strictEqual(await executor.evaluate('$x','{}',{bindingsJSON:'{"x":null}'}),'null');
            assert.strictEqual(await executor.evaluate('$x','{}',{bindingsJSON:'{"x":9007199254740993}'}),'9007199254740993');
        });
        for (const expression of ['missing','function(){1}','{"nested":[function(){1}]}','/x/','{"groups":/a(b)?/("a").groups}']) {
            it('rejects a non-JSON result before serialization: '+expression, async () => {
                await assert.rejects(executor.evaluate(expression,'{}'), e => e.code === 'U_JSON_VALUE');
                assert.strictEqual(await executor.evaluate('42','{}'),'42');
            });
        }
        it('keeps the standard environment closed, including dynamic evaluation', async () => {
            for (const expression of ['$flatten([1])','$values({"a":1})','$eval("$flatten([1])")','$clone({})','$eval("$clone({})")','$process()','{"_jsonata_function":true}()']) await assert.rejects(executor.evaluate(expression,'{}'));
            await assert.rejects(executor.evaluate('$x()','{}',{bindingsJSON:'{"x":{"_jsonata_lambda":true}}'}));
            assert.strictEqual(await executor.evaluate('($flatten := function($x){$x};$flatten(7))','{}'),'7');
            assert.strictEqual(await executor.evaluate('($clone := function($x){$x};$eval("$clone(7)"))','{}'),'7');
            for (const expression of ['$ ~> |$|{"flag":true}|','($clone:=5;$ ~> |$|{"flag":true}|)','$eval("$ ~> |$|{\\"flag\\":true}|")']) {
                assert.strictEqual(await executor.evaluate(expression,'{"id":9007199254740993}'),'{"id":9007199254740993,"flag":true}');
            }
        });
        it('applies immutable budgets without changing successful values', async () => {
            const policy = {maxDigits:32,maxExponent:32};
            const small = factory({timeout:3000, numericWork:policy,cacheSize:2});
            const large = factory({timeout:3000,numericWork:{maxDigits:4096,maxExponent:4096},cacheSize:3});
            policy.maxDigits = 1;
            try {
                const pairs = await Promise.all(Array.from({length:12},async (_,i) => {
                    const expression = 'n+0.1'; const input = '{"n":'+i+'}';
                    return Promise.all([small.evaluate(expression,input),large.evaluate(expression,input)]);
                }));
                for (const pair of pairs) assert.strictEqual(pair[0],pair[1]);
                await assert.rejects(small.evaluate('$power(10,100)','{}'), e => e.code === 'U_NUMERIC_LIMIT');
                assert.strictEqual(await large.evaluate('$power(10,100)','{}'),'1e+100');
                assert.strictEqual(await small.evaluate('0.1+0.2','{}'),'0.3');
            } finally { if (small.close) await small.close(); if (large.close) await large.close(); }
        });
        it('enforces exact expression/input/output length edges and bindings cost', async () => {
            const limited = factory({timeout:3000,maxExpressionLength:5,maxInputLength:5,maxOutputLength:5});
            try {
                assert.strictEqual(await limited.evaluate('$','12345'),'12345');
                await assert.rejects(limited.evaluate('$','123456'),RangeError);
                await assert.rejects(limited.evaluate('123456','{}'),RangeError);
                await assert.rejects(limited.evaluate('$','123',{bindingsJSON:'{"x":1}'}),RangeError);
                await assert.rejects(limited.evaluate('1e10','{}'), e => e.name === 'RangeError');
            } finally { if (limited.close) await limited.close(); }
        });
        it('does not admit work with an already-aborted signal', async () => {
            const controller = new AbortController(); const reason = new Error('caller cancelled'); controller.abort(reason);
            await assert.rejects(executor.evaluate('$','{}',{signal:controller.signal}), e => e === reason);
        });
    });
}

describe('Node worker lifecycle and hard containment', function() {
    this.timeout(10000);
    it('imports composition without loading evaluator or number libraries', function() {
        const source = 'require('+JSON.stringify(require.resolve('../src/node-executor'))+'); console.log(JSON.stringify(Object.keys(require.cache)))';
        const result = spawnSync(process.execPath,['-e',source],{encoding:'utf8',timeout:3000});
        assert.strictEqual(result.status,0,result.stderr);
        for (const file of JSON.parse(result.stdout)) assert(!/src\/(?:jsonata|numeric|parser)\.js$|node_modules\/(?:decimal|big|lossless-json)/.test(file),file);
    });
    it('terminates a warmed worker in a synchronous pathological regex, then recovers', async function() {
        const executor = createNodeExecutor({workers:1,timeout:4000});
        try {
            await executor.evaluate('1','{}'); // worker startup is not the cancelled work
            const controller = new AbortController(); const reason = new Error('stop running regex');
            const start = performance.now();
            const result = executor.evaluate('$contains($,/^(a+)+$/)',JSON.stringify('a'.repeat(34)+'!'),{signal:controller.signal});
            const timer = setTimeout(() => controller.abort(reason),100);
            try { await assert.rejects(result,e => e === reason); } finally { clearTimeout(timer); }
            // The public Promise settles after worker.terminate() confirms exit.
            assert(performance.now()-start < 1500,'cancellation failed to terminate promptly');
            assert.strictEqual(await executor.evaluate('0.1+0.2','{}'),'0.3');
        } finally { await executor.close(); }
    });
    it('bounds active+queued jobs, cancels queued work, and closes active work', async function() {
        const executor = createNodeExecutor({workers:1,maxPending:2,timeout:4000});
        await executor.evaluate('1','{}');
        const active = assert.rejects(executor.evaluate('$contains($,/^(a+)+$/)',JSON.stringify('a'.repeat(34)+'!')),/closed/);
        const controller = new AbortController();
        const queued = assert.rejects(executor.evaluate('2','{}',{signal:controller.signal}),/queued cancelled/);
        await assert.rejects(executor.evaluate('3','{}'),/pending work budget/);
        controller.abort(new Error('queued cancelled')); await queued;
        await executor.close(); await active;
        await assert.rejects(executor.evaluate('4','{}'),/closed/);
        await executor.close();
    });
    it('deadline terminates a blocked worker without a caller signal', async function() {
        const executor = createNodeExecutor({workers:1,timeout:500});
        try {
            await executor.evaluate('1','{}');
            await assert.rejects(executor.evaluate('$contains($,/^(a+)+$/)',JSON.stringify('a'.repeat(34)+'!')),e => e.code === 'U_TIMEOUT');
            assert.strictEqual(await executor.evaluate('4','{}'),'4');
        } finally { await executor.close(); }
    });
});
