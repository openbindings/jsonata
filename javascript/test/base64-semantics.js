'use strict';
const assert=require('assert'),jsonata=require('../src/jsonata');
describe('Documented base64 byte and UTF-8 domains',function(){
    it('encodes every admitted byte as one byte, not its UTF-8 encoding',async function(){
        const e=jsonata('$base64encode($)');
        for(let n=0;n<256;n++)assert.strictEqual(await e.evaluate(String.fromCharCode(n)),Buffer.from([n]).toString('base64'));
    });
    it('decodes valid UTF-8 text without Latin-1 mojibake or normalization',async function(){
        const e=jsonata('$base64decode($)');
        for(const s of ['', 'ASCII', 'é', 'e\u0301', '😀', '\u0000', '\ufefftext', '漢字'])assert.strictEqual(await e.evaluate(Buffer.from(s,'utf8').toString('base64')),s);
        assert.strictEqual(await e.evaluate(' w6k\n'),'é');
        assert.strictEqual(await e.evaluate('w6k='),'é');
    });
    it('rejects non-byte encode inputs and malformed base64/UTF-8',async function(){
        for(const s of ['Ā','😀','\ud800','\udfff'])await assert.rejects(jsonata('$base64encode($)').evaluate(s),e=>e.code==='D3137');
        for(const s of ['a','?','6Q==','wK8=','7aCA','9JCAgA==','_w==','===='])await assert.rejects(jsonata('$base64decode($)').evaluate(s),e=>e.code==='D3137');
    });
});
