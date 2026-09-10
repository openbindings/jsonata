(function () {
    'use strict';

    // Pure policy validation: importing worker composition must not load an
    // evaluator, parser, decimal library or caller data into the parent process.
    /** Validate and freeze the per-instance decimal work domain.
     * @param {Object} value - Optional requested limits
     * @returns {Object} Immutable validated limits
     */
    function normalizeNumericLimits(value) {
        const limits = value === undefined ? {maxDigits:4096, maxExponent:4096} :
            {maxDigits:value && value.maxDigits, maxExponent:value && value.maxExponent};
        for (const key of ['maxDigits', 'maxExponent']) {
            if (!Number.isInteger(limits[key]) || limits[key] < 1 || limits[key] > 100000) throw new TypeError('Numeric work limits require integers from 1 to 100000');
        }
        return Object.freeze(limits);
    }

    /** Validate known resource settings; no setting changes assigned precision.
     * @param {Object} options - Requested resource settings
     * @returns {Object} Immutable effective settings
     */
    function optionsFor(options = {}) {
        const defaults = {timeout:1000, stack:100, sequence:10000000, cacheSize:64,
            maxExpressionLength:262144, maxInputLength:8388608, maxOutputLength:8388608};
        if (options === null || typeof options !== 'object' || Array.isArray(options)) throw new TypeError('JSON executor options must be an object');
        const result = {};
        for (const key of Reflect.ownKeys(options)) {
            if (!Object.prototype.hasOwnProperty.call(defaults,key) && key !== 'numericWork') throw new TypeError('Unknown JSON executor option: ' + String(key));
        }
        for (const key of Object.keys(defaults)) {
            const value = options[key] === undefined ? defaults[key] : options[key];
            if (!Number.isSafeInteger(value) || value < 1) throw new TypeError('JSON executor budgets must be positive safe integers');
            result[key] = value;
        }
        result.numericWork = normalizeNumericLimits(options.numericWork);
        return Object.freeze(result);
    }

    module.exports = {normalizeNumericLimits, optionsFor};
})();
