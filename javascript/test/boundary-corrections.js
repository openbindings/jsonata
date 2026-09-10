'use strict';
const assert = require('assert');
const {createJSONataExecutor} = require('../src/json-executor');
const {createNodeExecutor} = require('../src/node-executor');

describe('Boundary corrections', function() {
    this.timeout(15000);
    const executor = createJSONataExecutor({timeout:5000});
    for (const input of ['{"id":7', '{"id":7} trailing', '{"id":7,"id":7}', '{"id":7,"id":8}', '{"id":7,"\\u0069d":7}', '{"id":7,"x":{"a":{},"a":[]}}']) {
        for (const expression of ['id','id = 7','$exists(id)','{"value":id}','42']) {
            it('rejects complete input '+expression+' / '+input, async () => {
                await assert.rejects(executor.evaluate(expression,input));
            });
        }
    }
    for (const input of ['{"$":{"fake":2}}','{"$x":1}','{"x":1,"x":1}','{"x":{"n":1,"n":2}}']) {
        it('rejects invalid bindings '+input, async () => {
            await assert.rejects(executor.evaluate('$$','{"real":1}',{bindingsJSON:input}));
        });
    }
    for (const [expression,input,output] of [
        ['a = b','{"a":{"length":0},"b":[]}','false'],
        ['b = a','{"a":{"length":0},"b":[]}','false'],
        ['a != b','{"a":[{"length":0}],"b":[[]]}','true'],
        ['$distinct(items)','{"items":[{"length":0},[]]}','[{"length":0},[]]'],
        ['$','{"__proto__":{"x":7},"constructor":2}','{"__proto__":{"x":7},"constructor":2}'],
    ]) {
        it('preserves types '+expression+' / '+input, async () => assert.strictEqual(await executor.evaluate(expression,input),output));
    }
    it('retains the root and dollar-named data', async () => {
        assert.strictEqual(await executor.evaluate('$$','{"real":1}',{bindingsJSON:'{"":42}'}),'{"real":1}');
        assert.strictEqual(await executor.evaluate('$x','null',{bindingsJSON:'{"x":{"$":7}}'}),'{"$":7}');
    });
    it('rejects overflowed timers before allocating a worker', async () => {
        assert.throws(() => createNodeExecutor({timeout:2147483648}),/timeout|timer/i);
        const pool=createNodeExecutor({timeout:2147483647,workers:1});
        try {assert.strictEqual(await pool.evaluate('1+1','null'),'2');} finally {await pool.close();}
        assert(createJSONataExecutor({timeout:2147483648}));
    });
});
