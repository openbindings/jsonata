'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');

describe('Per-evaluation cooperative cancellation', function() {
    this.timeout(3000);
    it('refuses pre-cancelled evaluation and preserves arbitrary abort reasons as causes', async function() {
        const c = new AbortController(); c.abort('stop');
        await assert.rejects(jsonata('$').evaluate({}, undefined, {signal:c.signal}), e =>
            e.name === 'AbortError' && e.code === 'ABORT_ERR' && e.cause === 'stop');
    });
    it('stops tail recursion after a real event-loop timer fires', async function() {
        const c = new AbortController();
        const expr = jsonata('($f := function($n){$f($n+1)}; $f(0))', {timeout:1000});
        const timer = setTimeout(() => c.abort(), 5);
        try { await assert.rejects(expr.evaluate({}, undefined, {signal:c.signal}), e => e.name === 'AbortError'); }
        finally { clearTimeout(timer); }
    });
    it('cancelling one call does not poison concurrent or later reuse', async function() {
        const expr = jsonata('slow ? ($f := function($n){$f($n+1)}; $f(0)) : id', {timeout:1000});
        const c = new AbortController();
        const slow = assert.rejects(expr.evaluate({slow:true}, undefined, {signal:c.signal}), e => e.name === 'AbortError');
        const timer = setTimeout(() => c.abort(), 5);
        try {
            assert.strictEqual(await expr.evaluate({slow:false,id:'other'}, undefined, {signal:new AbortController().signal}), 'other');
            await slow;
            assert.strictEqual(await expr.evaluate({slow:false,id:'later'}), 'later');
        } finally { clearTimeout(timer); }
    });
    it('discards a result when host code aborts during evaluation', async function() {
        const c = new AbortController();
        const expr = jsonata('$host()');
        expr.registerFunction('host', () => { c.abort(); return 42; });
        await assert.rejects(expr.evaluate({}, undefined, {signal:c.signal}), e => e.name === 'AbortError');
    });
    it('leaves ordinary values and callback evaluation unchanged', async function() {
        const expr = jsonata('{"id":id,"n":n+1}');
        const plain = await expr.evaluate({id:'7',n:2});
        const controlled = await expr.evaluate({id:'7',n:2}, undefined, {signal:new AbortController().signal});
        assert.deepStrictEqual(controlled, plain);
        assert.deepStrictEqual(await expr.evaluate({id:'7',n:2}, undefined, {}), plain);
        let callbackValue;
        await expr.evaluate({id:'7',n:2}, undefined, (_err, value) => { callbackValue = value; });
        assert.deepStrictEqual(callbackValue, plain);
    });
    it('resource exhaustion remains distinct from caller cancellation', async function() {
        const signal = new AbortController().signal;
        await assert.rejects(jsonata('($f := function($n){$f($n+1)}; $f(0))', {timeout:15})
            .evaluate({}, undefined, {signal}), e => e.code === 'D1012');
        await assert.rejects(jsonata('[1..100]', {sequence:10}).evaluate({}, undefined, {signal}), e => e.code === 'D2015');
        assert.strictEqual(signal.aborted, false);
    });
    it('nested $eval preserves cancellation classification and ordinary expression errors', async function() {
        const c = new AbortController();
        const expr = jsonata('$eval(expression)', {timeout:1000});
        const timer = setTimeout(() => c.abort(), 5);
        try {
            await assert.rejects(expr.evaluate({expression:'($f := function($n){$f($n+1)}; $f(0))'}, undefined, {signal:c.signal}), e => e.name === 'AbortError');
        } finally { clearTimeout(timer); }
        await assert.rejects(expr.evaluate({expression:'$error("ordinary")'}, undefined, {signal:new AbortController().signal}), e => e.code === 'D3121');
    });
});
