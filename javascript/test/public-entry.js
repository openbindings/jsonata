'use strict';
const assert = require('assert');
const api = require('../src/executor-public');
describe('Independent public entry', function () {
    it('exports only the closed executor, without backend helpers', async function () {
        assert.deepStrictEqual(Object.keys(api), ['createJSONExecutor']);
        assert(Object.isFrozen(api));
        const executor = api.createJSONExecutor();
        assert.strictEqual(await executor.evaluate('0.1+0.2', 'null'), '0.3');
        await assert.rejects(executor.evaluate('{"bad":function(){1}}', '{}'));
    });
});
