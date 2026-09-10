'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');
const numeric = require('../src/numeric');
const {cases} = require('./official-value-cases.json');
assert.strictEqual(cases.length, 53);
describe('Candidate structural value and control witnesses', function () {
    for (const c of cases) {
        it(c.id, async function () {
            const result = await jsonata(c.expr).evaluate(numeric.parse(c.inputJSON));
            // JSON result shape and assigned numbers, not internal sequence flags.
            assert.deepStrictEqual(numeric.parse(numeric.stringify(result)), numeric.parse(c.resultJSON));
        });
    }
});
