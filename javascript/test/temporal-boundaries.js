'use strict';
const assert=require('assert'),jsonata=require('../src/jsonata');
const {cases}=require('./official-temporal-cases.json');
describe('Temporal boundary regressions',function(){
    it('supports the exported integer helpers without an evaluation environment',function(){
        const datetime=require('../src/datetime');
        const {formatInteger,parseInteger}=datetime;
        assert.strictEqual(formatInteger(5,'1'),'5');
        assert.strictEqual(parseInteger('5','1'),5);
        assert.strictEqual(datetime.formatInteger(5,'1'),'5');
    });
    it('rejects malformed and excessive date-picture widths',async function(){
        for(const width of ['0','-1','bad','1z','3-2','1-0'])await assert.rejects(jsonata('$fromMillis(0,"[f1,'+width+']")').evaluate({}),e=>e.code==='D3135');
        await assert.rejects(jsonata('$fromMillis(0,"[f1,10001]")').evaluate({}),e=>e.code==='D3137');
    });
    for(const c of cases)it(c.id,async function(){
        const e=jsonata(c.expr,{standardLibraryOnly:!!c.standardOnly});
        if(c.error)await assert.rejects(e.evaluate({}),error=>error.code===c.error);
        else assert.deepStrictEqual(await e.evaluate({}),JSON.parse(c.resultJSON));
    });
});
