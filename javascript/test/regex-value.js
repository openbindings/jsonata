'use strict';
const assert=require('assert');
const jsonata=require('../src/jsonata');
const numeric=require('../src/numeric');
const {cases}=require('./official-regex-cases.json');
assert.strictEqual(cases.length,59);
describe('Matcher and UTF-16 boundary witnesses',function(){for(const c of cases){it(c.id,async function(){
 if(c.error==='syntax'){assert.throws(()=>jsonata(c.expr));return}
 if(c.error){await assert.rejects(()=>jsonata(c.expr).evaluate(numeric.parse(c.inputJSON)),e=>e.code===c.error);return}
 const result=await jsonata(c.expr).evaluate(numeric.parse(c.inputJSON));assert.deepStrictEqual(numeric.parse(numeric.stringify(result)),numeric.parse(c.resultJSON));
})}});
