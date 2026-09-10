'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');
const numeric = require('../src/numeric');
const {cases} = require('./official-hof-cases.json');
describe('Higher-order callback arity and positions', function() {
    for (const c of cases) it(c.id, async function() {
        const run = () => jsonata(c.expr).evaluate(numeric.parse(c.inputJSON));
        if (c.errorCode) {
            await assert.rejects(run, error => error.code === c.errorCode);
        } else {
            assert.deepStrictEqual(JSON.parse(numeric.stringify(await run())), JSON.parse(c.resultJSON));
        }
    });
});
