'use strict';
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const assert = require('assert');
const numeric = require('../src/numeric');
const overlay = require('./official-expectation-overlays.json');
const reference = process.env.JSONATA_REFERENCE_EXPECTATIONS === '1';
assert.strictEqual(overlay.schema, 1);
assert.strictEqual(Object.keys(overlay.cases).length, 20);
for (const [id, delta] of Object.entries(overlay.cases)) {
    assert(delta.reason && delta.sources['groups/' + id], 'Unexplained/unpinned delta ' + id);
    assert.strictEqual('resultJSON' in delta, !('code' in delta));
    for (const [name, sha] of Object.entries(delta.sources)) {
        const bytes = fs.readFileSync(path.join(__dirname, 'test-suite', name));
        assert.strictEqual(crypto.createHash('sha256').update(bytes).digest('hex'), sha, 'Changed original fixture: ' + name);
    }
}

module.exports = function applyOfficialExpectation(id, testcase) {
    const delta = reference ? undefined : overlay.cases[id];
    if (!delta) return;
    for (const key of ['result', 'error', 'code', 'undefinedResult']) delete testcase[key];
    if ('code' in delta) testcase.code = delta.code;
    else testcase.result = numeric.parse(delta.resultJSON);
};
