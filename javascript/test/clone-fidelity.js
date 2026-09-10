"use strict";

const assert = require('assert');
const jsonata = require('../src/jsonata');

describe('Clone and explicit string preserve assigned JSON numbers', function () {
    for (const expression of ['$clone($)', '$ ~> |$|{"flag":true}|']) {
        it(expression + ' preserves unmodified nested values and detaches objects', async function () {
            const shared = {id: 0.10000000000000002};
            const input = {a: shared, b: shared, values: [1, 1.2345678901234567, null, false, 'é']};
            const result = await jsonata(expression).evaluate(input);
            assert.deepStrictEqual(result, expression === '$clone($)' ? input : {...input, flag: true});
            assert.notStrictEqual(result, input);
            assert.notStrictEqual(result.a, input.a);
            assert.notStrictEqual(result.a, result.b);
            result.a.id = 7;
            assert.strictEqual(input.a.id, 0.10000000000000002);
            assert.strictEqual(result.b.id, 0.10000000000000002);
        });
    }

    it('keeps existing intermediate function-to-string behavior', async function () {
        const result = await jsonata('$clone({"f":function($x){$x},"a":[function($x){$x}],"n":0.10000000000000002})').evaluate({});
        assert.deepStrictEqual(result, {f: '', a: [''], n: 0.10000000000000002});
    });

    it('renders the assigned value without the old 15-digit display rounding', async function () {
        assert.strictEqual(await jsonata('$string({"n":0.10000000000000002})').evaluate({}), '{"n":0.10000000000000002}');
        assert.strictEqual(await jsonata('$string(0.10000000000000002)').evaluate({}), '0.10000000000000002');
    });

    it('preserves empty arrays and null members', async function () {
        assert.deepStrictEqual(await jsonata('$clone({"empty":[],"null":null,"missing":missing})').evaluate({}), {empty: [], null: null});
        assert.deepStrictEqual(await jsonata('$clone([[],[1],null])').evaluate({}), [[], [1], null]);
    });

    it('retains undefined and invalid-argument outcomes', async function () {
        assert.strictEqual(await jsonata('$clone(missing)').evaluate({}), undefined);
        for (const expression of ['$clone(null)', '$clone(1)', '$clone("x")']) {
            await assert.rejects(() => jsonata(expression).evaluate({}), error => error.code === 'T0410');
        }
    });
});
