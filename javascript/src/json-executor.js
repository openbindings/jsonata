(function () {
    'use strict';

    // A reusable, closed JSON-text boundary. It has no OpenBindings dependency and
    // no custom function/host binding registration surface. Hosts own work budgets.
    const jsonata = require('./jsonata');
    const numeric = require('./numeric');
    const utils = require('./utils');
    const {optionsFor} = require('./execution-options');

    /** Reject non-JSON runtime values before serialization can disguise them.
     * @param {*} value - Evaluator result
     */
    function assertResult(value) {
        const active = new Set();
        let remaining = 1000000;
        const invalid = message => { throw Object.assign(new Error(message), {code:'U_JSON_VALUE'}); };
        /** Walk own data members with cycle, cardinality and depth guards.
         * @param {*} value - Current result member
         * @param {number} depth - Current nesting depth
         */
        function visit(value, depth) {
            if (--remaining < 0 || depth > 512) invalid('JSON result exceeds materialization budget');
            if (value === null || typeof value === 'string' || typeof value === 'boolean' || numeric.isNumeric(value)) return;
            if (utils.isFunction(value) || typeof value !== 'object' || active.has(value)) invalid('Result is not a JSON value');
            const array = Array.isArray(value);
            const prototype = Object.getPrototypeOf(value);
            if (!array && prototype !== null && prototype !== Object.prototype) invalid('Result contains a host object');
            active.add(value);
            const keys = array ? Array.from({length:value.length}, (_, i) => String(i)) : Reflect.ownKeys(value);
            for (const key of keys) {
                const descriptor = Object.getOwnPropertyDescriptor(value, key);
                if (typeof key !== 'string' || !descriptor || !descriptor.enumerable || !Object.prototype.hasOwnProperty.call(descriptor,'value')) invalid('Result contains a non-JSON member');
                visit(descriptor.value, depth+1);
            }
            active.delete(value);
        }
        visit(value, 0);
    }

    /** Construct a closed JSON-text executor with an instance-owned compile cache.
     * @param {Object} options - Optional resource settings
     * @returns {Object} Closed text-evaluation interface
     */
    function createJSONataExecutor(options) {
        const limits = optionsFor(options);
        const cache = new Map();
        return Object.freeze({
            async evaluate(expression, inputJSON, {bindingsJSON, signal} = {}) {
                const check = () => { if (signal && signal.aborted) throw signal.reason || Object.assign(new Error('Evaluation cancelled'), {name:'AbortError'}); };
                check();
                if (typeof expression !== 'string' || expression.length > limits.maxExpressionLength) throw new RangeError('JSONata expression length budget exceeded');
                if (typeof inputJSON !== 'string' || (bindingsJSON !== undefined && typeof bindingsJSON !== 'string') || inputJSON.length + (bindingsJSON === undefined ? 0 : bindingsJSON.length) > limits.maxInputLength) throw new RangeError('JSONata input length budget exceeded');
                let compiled = cache.get(expression);
                if (!compiled) {
                    compiled = jsonata(expression, {timeout:limits.timeout, stack:limits.stack, sequence:limits.sequence, numericWork:limits.numericWork, standardLibraryOnly:true});
                    if (cache.size === limits.cacheSize) cache.delete(cache.keys().next().value);
                    cache.set(expression, compiled);
                }
                check();
                const input = numeric.parse(inputJSON);
                const bindings = bindingsJSON === undefined ? undefined : numeric.parse(bindingsJSON);
                if (bindings !== undefined && (bindings === null || typeof bindings !== 'object' || Array.isArray(bindings) || numeric.isNumeric(bindings))) throw new TypeError('JSONata bindings must be a JSON object');
                if (bindings !== undefined && Object.keys(bindings).some(name => name[0] === '$')) throw new TypeError('JSONata binding names must omit the dollar prefix');
                check();
                const result = await compiled.evaluate(input, bindings, {signal});
                check();
                assertResult(result); // before any serializer can omit/disguise values
                const output = numeric.stringify(result);
                if (output.length > limits.maxOutputLength) throw new RangeError('JSONata output length budget exceeded');
                check();
                return output;
            }
        });
    }

    module.exports = {createJSONataExecutor, optionsFor, assertResult};
})();
