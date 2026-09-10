/**
 * © Copyright IBM Corp. 2016, 2018 All Rights Reserved
 *   Project name: JSONata
 *   This project is licensed under the MIT License, see LICENSE
 */

const numeric = require('./numeric');
const functionValues = new WeakMap();
const utils = (() => {
    'use strict';

    /**
     * Check if value is a finite number
     * @param {float} n - number to evaluate
     * @returns {boolean} True if n is a finite number
     */
    function isNumeric(n) {
        if (numeric.isNumeric(n)) return true;
        var isNum = false;
        if(typeof n === 'number') {
            isNum = !isNaN(n);
            if (isNum && !isFinite(n)) {
                throw {
                    code: "D1001",
                    value: n,
                    stack: (new Error()).stack
                };
            }
        }
        return isNum;
    }

    /**
     * Returns true if the arg is an array of strings
     * @param {*} arg - the item to test
     * @returns {boolean} True if arg is an array of strings
     */
    function isArrayOfStrings(arg) {
        var result = false;
        /* istanbul ignore else */
        if(Array.isArray(arg)) {
            result = (arg.filter(function(item){return typeof item !== 'string';}).length === 0);
        }
        return result;
    }

    /**
     * Returns true if the arg is an array of numbers
     * @param {*} arg - the item to test
     * @returns {boolean} True if arg is an array of numbers
     */
    function isArrayOfNumbers(arg) {
        var result = false;
        if(Array.isArray(arg)) {
            result = (arg.filter(function(item){return !isNumeric(item);}).length === 0);
        }
        return result;
    }

    /**
     * Tests if a value is a sequence
     * @param {*} value the value to test
     * @returns {boolean} true if it's a sequence
     */
    function isSequence(value) {
        return value.sequence === true && Array.isArray(value);
    }

    /**
     *
     * @param {Object} arg - expression to test
     * @returns {boolean} - true if it is a function (lambda or built-in)
     */
    function isFunction(arg) {
        return typeof arg === 'function' || functionValues.has(arg);
    }

    /** Privately brand an evaluator-created callable, never a JSON data object.
     * @param {Object} value - Evaluator-created callable
     * @param {string} kind - Internal callable kind
     * @returns {Object} Unchanged branded callable
     */
    function functionValue(value, kind) {
        functionValues.set(value, kind);
        return value;
    }

    /** Test authenticated native callables without consulting data markers.
     * @param {*} value - Candidate value
     * @returns {boolean} Whether it is a branded native callable
     */
    function isNativeFunction(value) {
        return functionValues.get(value) === 'native';
    }

    /**
     * Returns the arity (number of arguments) of the function
     * @param {*} func - the function
     * @returns {*} - the arity
     */
    function getFunctionArity(func) {
        var arity = typeof func.arity === 'number' ? func.arity :
            typeof func.implementation === 'function' ? func.implementation.length :
                typeof func.length === 'number' ? func.length : func.arguments.length;
        return arity;
    }

    /**
     * Tests whether arg is a lambda function
     * @param {*} arg - the value to test
     * @returns {boolean} - true if it is a lambda function
     */
    function isLambda(arg) {
        return functionValues.get(arg) === 'lambda';
    }

    // istanbul ignore next
    var iteratorSymbol = (typeof Symbol === "function" ? Symbol : {}).iterator || "@@iterator";

    /**
     * @param {Object} arg - expression to test
     * @returns {boolean} - true if it is iterable
     */
    function isIterable(arg) {
        return (
            typeof arg === 'object' &&
            arg !== null &&
            iteratorSymbol in arg &&
            'next' in arg &&
            typeof arg.next === 'function'
        );
    }

    /**
     * Compares two values for equality
     * @param {*} lhs first value
     * @param {*} rhs second value
     * @returns {boolean} true if they are deep equal
     */
    function isDeepEqual(lhs, rhs) {
        if (isNumeric(lhs) || isNumeric(rhs)) return isNumeric(lhs) && isNumeric(rhs) && numeric.compare(lhs, rhs) === 0;
        if (lhs === rhs) {
            return true;
        }
        if(typeof lhs === 'object' && typeof rhs === 'object' && lhs !== null && rhs !== null) {
            if(Array.isArray(lhs) && Array.isArray(rhs)) {
                // both arrays (or sequences)
                // must be the same length
                if(lhs.length !== rhs.length) {
                    return false;
                }
                // must contain same values in same order
                for(var ii = 0; ii < lhs.length; ii++) {
                    if(!isDeepEqual(lhs[ii], rhs[ii])) {
                        return false;
                    }
                }
                return true;
            }
            // both objects
            // must have the same set of keys (in any order)
            var lkeys = Object.getOwnPropertyNames(lhs);
            var rkeys = Object.getOwnPropertyNames(rhs);
            if(lkeys.length !== rkeys.length) {
                return false;
            }
            lkeys = lkeys.sort();
            rkeys = rkeys.sort();
            for(ii=0; ii < lkeys.length; ii++) {
                if(lkeys[ii] !== rkeys[ii]) {
                    return false;
                }
            }
            // must have the same values
            for(ii=0; ii < lkeys.length; ii++) {
                var key = lkeys[ii];
                if(!isDeepEqual(lhs[key], rhs[key])) {
                    return false;
                }
            }
            return true;
        }
        return false;
    }

    /**
     * @param {Object} arg - expression to test
     * @returns {boolean} - true if it is a promise
     */
    function isPromise(arg) {
        return (
            typeof arg === 'object' &&
                arg !== null &&
                'then' in arg &&
                typeof arg.then === 'function'
        );
    }

    /**
     * converts a string to an array of characters
     * @param {string} str - the input string
     * @returns {Array} - the array of characters
     */
    function stringToArray(str) {
        var arr = [];
        for (let char of str) {
            arr.push(char);
        }
        return arr;
    }

    /** Compare strings by Unicode codepoint, as required by sort/order-by.
     * No locale tailoring or normalization is performed. Unpaired UTF-16
     * units retain their own codepoint values rather than becoming U+FFFD.
     * @param {string} left - Left operand
     * @param {string} right - Right operand
     * @returns {number} Signed lexical ordering
     */
    function compareStrings(left, right) {
        let i = 0, j = 0;
        while (i < left.length && j < right.length) {
            const a = left.codePointAt(i), b = right.codePointAt(j);
            if (a !== b) return a - b;
            i += a > 0xFFFF ? 2 : 1;
            j += b > 0xFFFF ? 2 : 1;
        }
        return left.length - right.length;
    }

    /**
     *
     * @param {Object} arg - the object to inspect
     * @returns {Array} - the keys (property names) of the object
     */
    function keys(arg) {
        return typeof arg === 'object' && arg !== null && !isNumeric(arg) ? Object.keys(arg) : [];
    }

    return {
        isNumeric,
        isArrayOfStrings,
        isArrayOfNumbers,
        isSequence,
        isFunction,
        functionValue,
        isNativeFunction,
        isLambda,
        isIterable,
        getFunctionArity,
        isDeepEqual,
        stringToArray,
        compareStrings,
        isPromise,
        keys
    };
})();

module.exports = utils;
