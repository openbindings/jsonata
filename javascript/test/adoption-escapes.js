'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');
const cases = require('../../contract/cases/adoption-escape-cases.json').cases;
describe('Adoption Unicode escape regressions', function () {
    for (const test of cases) {
        it(test.id, function () {
            assert.throws(() => jsonata(test.expr), error => error.code === test.error);
        });
    }
});
