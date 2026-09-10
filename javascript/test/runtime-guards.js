'use strict';
const assert = require('assert');
const numeric = require('../src/numeric');
const utils = require('../src/utils');
const functions = require('../src/functions');
const datetime = require('../src/datetime');
const jsonata = require('../src/jsonata');
const {optionsFor,normalizeNumericLimits} = require('../src/execution-options');
const {createJSONExecutor,assertResult} = require('../src/json-executor');
const {createNodeExecutor} = require('../src/node-executor');
const number = numeric.fromText;
const code = wanted => error => error.code === wanted;

describe('Resource and result-domain guards', function() {
    it('validates every host resource control without selecting a precision mode', async function() {
        assert(Object.isFrozen(optionsFor()));
        for (const value of [null,[],true,'limits']) assert.throws(()=>optionsFor(value),TypeError);
        for (const key of ['unknown','precision','__proto__',Symbol('policy')]) assert.throws(()=>optionsFor({[key]:1}),TypeError);
        for (const key of ['timeout','stack','sequence','cacheSize','maxExpressionLength','maxInputLength','maxOutputLength']) {
            for (const value of [0,-1,1.5,Infinity,Number.MAX_SAFE_INTEGER+1,'1']) assert.throws(()=>optionsFor({[key]:value}),TypeError);
        }
        for (const value of [null,{},false,{maxDigits:0,maxExponent:1},{maxDigits:1,maxExponent:100001},{maxDigits:1.5,maxExponent:1}]) assert.throws(()=>normalizeNumericLimits(value),TypeError);
        assert.deepStrictEqual(normalizeNumericLimits({maxDigits:1,maxExponent:100000}),{maxDigits:1,maxExponent:100000});
        for (const value of [{workers:0},{workers:33},{maxPending:0},{maxWorkerHeapMB:15},{maxWorkerHeapMB:16.5}]) assert.throws(()=>createNodeExecutor(value),TypeError);
        const worker=createNodeExecutor();await worker.close();
        const executor=createJSONExecutor();
        for (const bindingsJSON of ['null','[]','1','"x"','true','1e400']) await assert.rejects(executor.evaluate('$','{}',{bindingsJSON}),TypeError);
        for (const [expression,inputJSON,options] of [[null,'{}'],['$',null],['$','{}',{bindingsJSON:7}]]) await assert.rejects(executor.evaluate(expression,inputJSON,options),RangeError);
        await assert.rejects(executor.evaluate('$','{}',{signal:{aborted:true}}), e=>e.name==='AbortError');
    });
    it('rejects host objects, members, cycles and excessive result depth before serialization', function() {
        const cyclic={};cyclic.self=cyclic;
        let deep=0;for(let i=0;i<514;i++)deep={child:deep};
        const accessor=Object.defineProperty({},'value',{enumerable:true,get(){throw Error('getter must not run')}});
        const hidden=Object.defineProperty({},'value',{value:1});
        for(const value of [cyclic,deep,new Date(),new Map(),accessor,hidden,{[Symbol('member')]:1},Array(1),Infinity,NaN,undefined]) assert.throws(()=>assertResult(value),code('U_JSON_VALUE'));
        assertResult(Object.assign(Object.create(null),{value:null}));
        assert.throws(()=>numeric.stringify(cyclic),/Cyclic/);
        assert.throws(()=>numeric.stringify(accessor),/accessors/);
        assert.throws(()=>numeric.stringify(deep),code('U_NUMERIC_LIMIT'));
        assert.strictEqual(numeric.stringify(Array(1)),'[null]'); // codec, not strict result admission
    });
    it('keeps private number and control guards independent from host values', function() {
        for(const value of [Infinity,NaN,{},'1']) assert.throws(()=>numeric.text(value),TypeError);
        for(const raw of ['NaN','Infinity','01','1.','.1','--1']) assert.throws(()=>number(raw),TypeError);
        for(const bounds of [[1,0],[0,Infinity],[0,.5]]) assert.throws(()=>numeric.clippedTrunc(1,...bounds),TypeError);
        assert.strictEqual(numeric.index(-.5,.6),undefined);
        assert.strictEqual(utils.isNumeric(NaN),false);
        assert.throws(()=>utils.isNumeric(Infinity),code('D1001'));
        assert.throws(()=>functions.string(Infinity),code('D3001'));
        assert.throws(()=>functions.number(null),code('D3030'));
        assert.strictEqual(functions.sum([number('0.1'),number('0.2')]),.3);
        assert.strictEqual(numeric.fixed(1.25,1),'1.2');
        assert.deepStrictEqual(numeric.mantissa(0,1),{value:0,exponent:0});
        assert.strictEqual(numeric.radixString(255,16),'ff');
        assert.throws(()=>numeric.integer(1.2),TypeError);
        for(const places of [-1,.5,5000])assert.throws(()=>numeric.fixed(1,places),code('U_NUMERIC_LIMIT'));
        for(const leading of [-1,.5,5000])assert.throws(()=>numeric.mantissa(1,leading),code('U_NUMERIC_LIMIT'));
        for(const [value,radix] of [[1.2,16],[1,1],[1,37],[1,2.5]])assert.throws(()=>numeric.radixString(value,radix),TypeError);
        assert.throws(()=>numeric.calculate('unknown',1),/Unknown numerical operation/);
        assert.throws(()=>numeric.calculate('%',1,0),code('D1001'));
        assert.throws(()=>numeric.calculate('sqrt',-1),code('D3060'));
        assert.strictEqual(numeric.calculate('/',0,1),0);
        assert.strictEqual(numeric.calculate('sqrt',0),0);
        for(const places of [.5,5000])assert.throws(()=>numeric.calculate('round',1,undefined,places),code('U_NUMERIC_LIMIT'));
        assert.throws(()=>numeric.calculate('+',number('1e5000'),1),code('U_NUMERIC_LIMIT'));
        const small={maxDigits:2,maxExponent:2};
        assert.throws(()=>numeric.calculate('/',1,3,0,small),code('U_NUMERIC_LIMIT'));
        assert.throws(()=>numeric.calculate('sqrt',2,undefined,0,small),code('U_NUMERIC_LIMIT'));
        for(const [left,right] of [[-1,.1],[0,-.1]])assert.throws(()=>numeric.calculate('pow',left,right),code('D3061'));
        assert.strictEqual(numeric.calculate('pow',0,.1),0);
        assert.strictEqual(numeric.calculate('pow',1,.1),1);
        assert.throws(()=>numeric.calculate('pow',2,.1,0,small),code('U_NUMERIC_LIMIT'));
        assert.throws(()=>numeric.calculate('pow',2,number('9007199254740993')),code('U_NUMERIC_LIMIT'));
        for(const right of [1000001.1,-1000001.1])assert.throws(()=>numeric.calculate('pow',10,right),code('U_NUMERIC_LIMIT'));
    });
    it('covers language-level comparison, order, control and date error boundaries', async function() {
        for(const [expression,want] of [['"a"<"b"',true],['"b">"a"',true],['"a">="a"',true],['[3,1,2]^(<$)',[1,2,3]],['- -1',1]])assert.deepStrictEqual(await jsonata(expression).evaluate({}),want);
        for(const [expression,wanted] of [['$pad("x",10001)','U_OUTPUT_LIMIT'],['$pad("x",-10001)','U_OUTPUT_LIMIT'],['$round(1,9007199254740993)','U_NUMERIC_LIMIT'],['$round(1,0.5)','T0410'],['$power(0,-1)','D3061'],['$fromMillis(8640000000000001)','D3110'],['$fromMillis(-8640000000000001)','D3110']])await assert.rejects(jsonata(expression).evaluate({}),code(wanted));
        assert.throws(()=>datetime.formatInteger.call({environment:{base:{options:{numericWork:{maxDigits:2,maxExponent:8}}}}},100000,'i'),code('U_NUMERIC_LIMIT'));
        await assert.rejects(jsonata('$parseInteger("not a number","w")').evaluate({}),code('D3137'));
        await assert.rejects(jsonata('$parseInteger("Z","a")').evaluate({}),code('D3137'));
        const ordered=await jsonata('[{"s":"b"},{"s":"a"},{"s":"a"},{"s":"c"}]^(s).s').evaluate({});
        assert.deepStrictEqual(Array.from(ordered),['a','a','b','c']);
    });
    it('rejects disagreement between numerical-library guard computations', function() {
        // Deliberately faulty backend: this verifies the failure guard, not a
        // claim that a naturally occurring difficult-rounding case was found.
        const Decimal=require('decimal.js'),original=Decimal.clone;
        try {
            Decimal.clone=options=>{
                const D=original.call(Decimal,options);
                return function FaultyDecimal(input) {
                    const value=new D(input),power=value.pow;
                    value.pow=function(exponent){const result=power.call(value,exponent);return options.precision===120?result.plus('0.01'):result;};
                    return value;
                };
            };
            assert.throws(()=>numeric.calculate('pow',2,.1),code('U_NUMERIC_LIMIT'));
        } finally {Decimal.clone=original;}
        assert.strictEqual(numeric.text(numeric.calculate('pow',2,.1)),'1.071773462536293164213006325023342');
    });
});
