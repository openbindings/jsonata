'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');
const numeric = require('../src/numeric');
const {cases} = require('./official-object-cases.json');
describe('Object traversal and array collection conventions', function() {
    for (const c of cases) it(c.id, async function() {
        const actual = await jsonata(c.expr).evaluate(numeric.parse(c.inputJSON));
        assert.strictEqual(numeric.stringify(actual), numeric.stringify(numeric.parse(c.resultJSON)));
    });
});
