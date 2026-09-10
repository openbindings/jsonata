'use strict';
const assert=require('assert'),jsonata=require('../src/jsonata'),numeric=require('../src/numeric');
const {cases}=require('./official-string-cases.json');assert.strictEqual(cases.length,72);
describe('Non-regex string semantics',function(){for(const c of cases){it(c.id,async function(){const result=await jsonata(c.expr).evaluate(numeric.parse(c.inputJSON));assert.deepStrictEqual(numeric.parse(numeric.stringify(result)),numeric.parse(c.resultJSON))})}});
