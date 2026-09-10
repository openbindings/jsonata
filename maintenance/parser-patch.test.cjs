'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const {render,generate} = require('../javascript/scripts/vendor-lossless.cjs');
const js = path.resolve(__dirname,'../javascript');
const original = fs.readFileSync(require.resolve('lossless-json',{paths:[js]}),'utf8');
test('source changes cannot silently refresh the parser correction',()=>{
 assert.throws(()=>render(original+' '),/requires review/);
 assert.equal(generate(true).status,'PASS');
});
test('duplicate patch catches equal values and escaped names; safe assignment preserves data',()=>{
 const context={exports:{},module:{}};vm.runInNewContext(render(original),context);
 const parse=context.exports.parse;
 for(const raw of ['{"a":1,"a":1}','{"a":1,"\\u0061":1}','{"a":{},"a":[]}','{"__proto__":null,"__proto__":null}'])assert.throws(()=>parse(raw),/Duplicate/);
 const value=parse('{"__proto__":{"x":7}}');
 assert.equal(Object.hasOwn(value,'__proto__'),true);
 assert.equal(String(value.__proto__.x),'7');
 assert.equal(Object.getPrototypeOf(value).x,undefined);
});
test('inherited setters are never invoked, including on the duplicate-callback path',()=>{
 const context={exports:{},module:{}};vm.runInNewContext(render(original),context);
 vm.runInNewContext('Object.defineProperty(Object.prototype,"trap",{set:function(){throw Error("inherited setter called")},configurable:true})',context);
 const parse=context.exports.parse;
 const value=parse('{"trap":7,"constructor":8,"toString":9}');
 for(const name of ['trap','constructor','toString'])assert.equal(Object.hasOwn(value,name),true);
 assert.equal(String(value.trap),'7');
 const duplicated=parse('{"__proto__":1,"__proto__":2}',undefined,{onDuplicateKey:({newValue})=>newValue});
 assert.equal(Object.hasOwn(duplicated,'__proto__'),true);
 assert.equal(String(duplicated.__proto__),'2');
});
test('removing either behavioral correction makes its witness fail',()=>{
 const duplicateMutant=render(original).replace('Object.prototype.hasOwnProperty.call(e,r)){','Object.prototype.hasOwnProperty.call(e,r)&&!v(o,e[r])){');
 const context={exports:{},module:{}};vm.runInNewContext(duplicateMutant,context);
 assert.doesNotThrow(()=>context.exports.parse('{"a":1,"a":1}'));
 const plain={exports:{},module:{}};vm.runInNewContext(original,plain);
 assert.equal(Object.hasOwn(plain.exports.parse('{"__proto__":{"x":7}}'),'__proto__'),false);
});
