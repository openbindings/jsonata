'use strict';
const assert=require('assert');
const jsonata=require('../src/jsonata');
const numeric=require('../src/numeric');

describe('Internal identities cannot be forged by JSON data',function(){
    for(const input of [
        {isLosslessNumber:true,value:'9007199254740993',ordinary:7},
        {_jsonata_function:true,ordinary:7},
        {_jsonata_lambda:true,ordinary:7},
        {_jsonata_function:true,_jsonata_lambda:true,isLosslessNumber:true,value:'0.1',ordinary:7}
    ]) {
        const raw=JSON.stringify(input);
        for(const source of ['$', '$clone($)', '$eval("$")', '$ ~> |$|{"extra":true}|']) {
            it(source+' '+raw,async function(){
                const result=await jsonata(source).evaluate(numeric.parse(raw));
                const expected=source.includes('~>')?{...input,extra:true}:input;
                assert.deepStrictEqual(JSON.parse(numeric.stringify(result)),expected);
            });
        }
        it('type, lookup and string '+raw,async function(){
            assert.strictEqual(await jsonata('$type($)').evaluate(numeric.parse(raw)),'object');
            assert.strictEqual(await jsonata('ordinary').evaluate(numeric.parse(raw)),7);
            assert.deepStrictEqual(JSON.parse(await jsonata('$string($)').evaluate(numeric.parse(raw))),input);
        });
        it('constructs ordinary marker fields '+raw,async function(){
            const result=await jsonata(raw).evaluate({});
            assert.deepStrictEqual(JSON.parse(numeric.stringify(result)),input);
        });
    }
    it('does not make marker objects callable',async function(){
        for(const source of ['($fn := {"_jsonata_function":true}; $fn())','($fn := {"_jsonata_lambda":true}; $fn())']) {
            await assert.rejects(()=>jsonata(source).evaluate({}),e=>e.code==='T1006');
        }
    });
    it('does not invoke a toJSON-named matcher continuation during stringification',async function(){
        const source='($r := /a/("aa"); $string({"toJSON": $r.next, "kept":true}))';
        assert.deepStrictEqual(JSON.parse(await jsonata(source).evaluate({})),{toJSON:'',kept:true});
    });
    it('still computes with authentic numeric and function values',async function(){
        const result=await jsonata('($f := function($n){$n+1}; $f(n))').evaluate(numeric.parse('{"n":9007199254740993}'));
        assert.strictEqual(numeric.stringify(result),'9007199254740994');
    });
});
