'use strict';
const assert = require('assert');
const http = require('http');
const httpget = require('./support/httpget.cjs');

async function fixture(handler, test) {
    const server = http.createServer(handler);
    await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
    try { await test('http://127.0.0.1:' + server.address().port); }
    finally { await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())); }
    assert.strictEqual(server.listening, false);
}

describe('Standard-library fixture HTTP helper', function() {
    it('resolves the original asynchronous JSON transport', async function() {
        await fixture((_request, response) => response.end('{"value":42}'), async url => {
            assert.deepStrictEqual(await httpget(url), {value: 42});
        });
    });
    it('rejects an invalid scheme without network access', async function() {
        await assert.rejects(httpget('htttttps://invalid.example/test'));
    });
    it('rejects malformed JSON and releases the fixture', async function() {
        await fixture((_request, response) => response.end('not JSON'), async url => {
            await assert.rejects(httpget(url), SyntaxError);
        });
    });
    it('rejects a broken connection and releases the fixture', async function() {
        await fixture((request, _response) => request.socket.destroy(), async url => {
            await assert.rejects(httpget(url));
        });
    });
});
