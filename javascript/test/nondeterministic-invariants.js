'use strict';
const assert = require('assert');
const jsonata = require('../src/jsonata');
const {createJSONExecutor} = require('../src/json-executor');

describe('Nondeterministic function invariants', function() {
    it('preserves random type/range/carriage and shuffle multiplicities under reuse', async function() {
        // No equality-between-samples, statistical or cryptographic assertion.
        const expression = `(
            $r := $random();
            $values := [9007199254740993, 0, 0, 9007199254740992];
            $shuffled := $shuffle($values);
            $type($r) = "number" and $r >= 0 and $r < 1 and
            $eval("$r") = $r and $number($string($r)) = $r and
            $count($shuffled) = 4 and
            $sort($shuffled) = [0, 0, 9007199254740992, 9007199254740993] and
            $values = [9007199254740993, 0, 0, 9007199254740992]
        )`;
        const compiled = jsonata(expression);
        const executor = createJSONExecutor();
        await Promise.all(Array.from({length: 8}, async function() {
            for (let sample = 0; sample < 64; sample++) {
                if (sample % 2 === 0) {
                    assert.strictEqual(await compiled.evaluate({}), true);
                } else {
                    assert.strictEqual(await executor.evaluate(expression, '{}'), 'true');
                }
            }
        }));
    });
});
