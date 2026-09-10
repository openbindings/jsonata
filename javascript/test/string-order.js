'use strict';
const assert=require('assert'),jsonata=require('../src/jsonata');
const {cases}=require('./official-string-order-cases.json');
describe('Documented Unicode codepoint ordering',function(){
    for(const c of cases)it(c.id,async function(){
        const actual=await jsonata(c.expr).evaluate(JSON.parse(c.inputJSON));
        assert.deepStrictEqual(Array.from(actual),JSON.parse(c.resultJSON));
    });
});
