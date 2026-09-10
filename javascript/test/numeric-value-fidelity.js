'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');
const numeric = require('../src/numeric');

describe('Official candidate assigned-decimal fidelity (not a Core floor)', function () {
    it('keeps successful results invariant across immutable host budgets', async function () {
        for (const source of ['1/3', '$eval("1/3")', '$average([1,0,0])', '$power(2,-100)', '$round(9007199254740993.25,1)', '($f:=function($x){$sum([$x,0.1,0.2])};$f(9007199254740993))']) {
            const values = [];
            for (const size of [512, 4096]) {
                const numericWork = {maxDigits: size, maxExponent: size};
                const expression = jsonata(source, {numericWork});
                numericWork.maxDigits = 1;
                values.push(await expression.evaluate({}));
            }
            assert.strictEqual(numeric.compare(values[0], values[1]), 0, source);
        }
    });
    it('rejects an insufficient budget without reducing precision or affecting another evaluator', async function () {
        const low = jsonata('1/3', {numericWork: {maxDigits: 16, maxExponent: 512}});
        const high = jsonata('1/3');
        await Promise.all(Array.from({length: 40}, async () => {
            await assert.rejects(() => low.evaluate({}), error => error.code === 'U_NUMERIC_LIMIT');
            assert.strictEqual(numeric.compareText(numeric.text(await high.evaluate({})), '0.3333333333333333333333333333333333'), 0);
        }));
    });
    const cases = [
        ["$formatInteger(n,\"#,##0\")","{\"n\":9007199254740993}","9,007,199,254,740,993"],
        ["$formatInteger(n,\"0\")","{\"n\":1000000000000000000000000000000000000000000000000001}","1000000000000000000000000000000000000000000000000001"],
        ["$formatInteger(n,\"0;o\")","{\"n\":9007199254740993}","9007199254740993rd"],
        ["$parseInteger(\"9007199254740993\",\"0\") = 9007199254740993","{}",true],
        ["$parseInteger($formatInteger(n,\"w\"),\"w\") = n","{\"n\":1000000000000000000000000000000000000000000000000001}",true],
        ["$parseInteger($formatInteger(n,\"A\"),\"A\") = n","{\"n\":9007199254740993}",true],
        ["$parseInteger($formatInteger(n,\"Ww;o\"),\"Ww;o\") = n","{\"n\":9007199254740993}",true],
        ["$formatInteger(-2.00000000000000000000001,\"0\")","{}","-3"],
        ["(n^(<$))[0] = 9007199254740992","{\"n\":[9007199254740993,9007199254740992]}",true],
        ["$string($power(2,0.1))","{}","1.071773462536293164213006325023342"],
        ["$string($power(1e-1000,0.125))","{}","1e-125"],
        ["$substring(\"abcd\",1.99999999999999999999999,1.99999999999999999999999)","{}","b"],
        ["$substring(\"abcd\",-1e-1000,2)","{}","ab"],
        ["$substring(\"abcd\",-1e1000,1e1000)","{}","abcd"],
        ["$substring(\"abcd\",1e1000)","{}",""],
        ["$pad(\"a\",2.99999999999999999999999,\".\")","{}","a."],
        ["$pad(\"a\",-2.99999999999999999999999,\".\")","{}",".a"],
        ["$fromMillis(1.99999999999999999999999)","{}","1970-01-01T00:00:00.001Z"],
        ["$fromMillis(-0.99999999999999999999999)","{}","1970-01-01T00:00:00.000Z"],
        ["$formatBase(9007199254740993,16)","{}","20000000000001"],
        ["$formatBase(2.5)","{}","2"],
        ["$formatBase(-2.5)","{}","-2"],
        ["$formatBase(99.5,2.5)","{}","1100100"],
        ['[10,20][-1e-1000]', '{}', 20],
        ['[10,20][0.99999999999999999999999999999]', '{}', 10],
        ['$exists([10,20][1e1000])', '{}', false],
        ['$string([9007199254740992..9007199254740994])', '{}', '[9007199254740992,9007199254740993,9007199254740994]'],
        ['$string($sqrt(2))', '{}', '1.414213562373095048801688724209698'],
        ['$string($sqrt(1e-1000))', '{}', '1e-500'],
        ['$string($sqrt(15241578753238836750495351562536198787501905199875019052100))', '{}', '1.2345678901234567890123456789e+29'],
        ['$formatNumber(n,"#,##0.0")', '{"n":9007199254740993.25}', '9,007,199,254,740,993.2'],
        ['$formatNumber(n,"0.000e0")', '{"n":1.23456789e-1000}', '1.235e-1000'],
        ['$formatNumber(n,"0.00%")', '{"n":0.1000000000000000000000000000000000001}', '10.00%'],
        ['$formatNumber(n,"0.00e0")', '{"n":9.999}', '10.00e0'],
        ['$formatNumber(n,"0.0;[0.0]")', '{"n":-9007199254740993.25}', '[9007199254740993.2]'],
        ['n = 9007199254740992', '{"n":9007199254740993.0}', false],
        ['n = 0', '{"n":1e-1000}', false],
        ['$boolean(n)', '{"n":1e-1000}', true],
        ['$not(n)', '{"n":1e-1000}', false],
        ['$string($number(n))', '{"n":"0x20000000000001"}', '9007199254740993'],
        ['$string($sum(n))', '{"n":[9007199254740993,0.1,0.2]}', '9007199254740993.3'],
        ['$string($average(n))', '{"n":[9007199254740993,9007199254740995]}', '9007199254740994'],
        ['$max(n) = 9007199254740993', '{"n":[9007199254740992,9007199254740993]}', true],
        ['$min(n) = 1e-1001', '{"n":[1e-1000,1e-1001]}', true],
        ['$string($floor(n))', '{"n":9007199254740993.75}', '9007199254740993'],
        ['$string($ceil(n))', '{"n":9007199254740993.25}', '9007199254740994'],
        ['$string($abs(n))', '{"n":-9007199254740993}', '9007199254740993'],
        ['$string($round(n))', '{"n":9007199254740993.5}', '9007199254740994'],
        ['$string($round(n,1))', '{"n":9007199254740993.25}', '9007199254740993.2'],
        ['$sort(n)[0] = 9007199254740992', '{"n":[9007199254740993,9007199254740992]}', true],
        ['$count($distinct(n)) = 2', '{"n":[9007199254740993,9007199254740992,9007199254740993.0]}', true],
        ['n in [9007199254740992]', '{"n":9007199254740993}', false],
        ['$eval("9007199254740993 = 9007199254740992")', '{}', false],
        ['0.1 + 0.2 = 0.3', '{}', true],
        ['$string((1/3)*3)', '{}', '0.9999999999999999999999999999999999'],
        ['$type(1e-1000)', '{}', 'number'],
        ['$count($keys(n))', '{"n":9007199254740993}', 0],
        ['$exists(n.value)', '{"n":9007199254740993}', false],
        ['$clone($).n = 9007199254740993', '{"n":9007199254740993}', true]
    ];
    for (const [expression, input, expected] of cases) {
        it(expression, async function () {
            const data = numeric.parse(input);
            const result = await jsonata(expression).evaluate(data);
            assert.strictEqual(result, expected);
        });
    }
    it('arithmetic matches an independent integer oracle', function () {
        // BigInt is test-only; the candidate browser runtime does not require it.
        let seed = 183784;
        for (let i = 0; i < 1000; i++) {
            seed = (seed * 1664525 + 1013904223) >>> 0;
            const a = BigInt(seed) * 9007199254740993n - 2333333333333333n;
            seed = (seed * 1664525 + 1013904223) >>> 0;
            const b = BigInt(seed) * 12345678910111213n;
            for (const [op, expected] of [['+', a+b], ['-', a-b], ['*', a*b]]) {
                const value = numeric.calculate(op, numeric.fromText(String(a)), numeric.fromText(String(b)));
                const serialized = numeric.stringify(value);
                assert.strictEqual(numeric.compareText(serialized, String(expected)), 0);
            }
        }
    });
    it('compares huge exponents without expanding their values', function () {
        assert.strictEqual(numeric.compareText('1e999999999999999999999999', '10e999999999999999999999998'), 0);
        assert.strictEqual(numeric.compareText('-1e-999999999999999999999999', '0'), -1);
    });
    it('keeps an exact terminating quotient longer than 34 digits', function () {
        const value = numeric.calculate('/', 1, numeric.fromText('1267650600228229401496703205376'));
        assert.strictEqual(numeric.compareText(numeric.text(value), '0.0000000000000000000000000000007888609052210118054117285652827862296732064351090230047702789306640625'), 0);
    });
});
