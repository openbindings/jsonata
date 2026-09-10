(function () {
    'use strict';

    // Isolated official-policy candidate. Arithmetic is delegated to big.js; JSON
    // number carriage uses lossless-json. Neither library's global settings change.
    const Big = require('big.js');
    const Decimal = require('decimal.js');
    const lossless = require('../vendor/lossless-json');
    const rawJSON = require('core-js-pure/actual/json/raw-json');
    const jsonStringify = require('core-js-pure/actual/json/stringify');
    // JSON member names are data, not authority to impersonate an engine number.
    const numberValues = new WeakSet();
    const B = Big(); // eslint-disable-line new-cap -- documented factory creates an isolated constructor
    B.strict = true;
    B.RM = Big.roundHalfEven;
    const grammar = /^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$/;
    const castGrammar = /^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$/;
    const defaultLimits = Object.freeze({maxDigits: 4096, maxExponent: 4096});

    const normalizeLimits = require('./execution-options').normalizeNumericLimits;

    /** Test native finite numbers and privately authenticated exact carriers.
     * @param {*} value - Candidate value
     * @returns {boolean} Whether this is an admitted number
     */
    function isNumeric(value) {
        return typeof value === 'number' ? Number.isFinite(value) :
            value !== null && typeof value === 'object' && numberValues.has(value);
    }
    /** Obtain the decimal image without a native-number conversion.
     * @param {*} value - Admitted number
     * @returns {string} Exact decimal image
     */
    function text(value) {
        if (!isNumeric(value)) throw new TypeError('Expected a finite JSON number');
        return typeof value === 'number' ? String(value) : value.value;
    }
    /** Split a decimal into sign, significant digits and an arbitrary-size exponent.
     * @param {string} source - Decimal image
     * @returns {Object} Sign, coefficient and scientific exponent
     */
    function parts(source) {
        const negative = source[0] === '-';
        const sections = source.replace(/^-/, '').toLowerCase().split('e');
        const point = sections[0].indexOf('.');
        const fraction = point < 0 ? 0 : sections[0].length - point - 1;
        const digits = sections[0].replace('.', '').replace(/^0+/, '');
        if (!digits) return {negative: false, coefficient: '0', exponent: new B('0')};
        return {negative, coefficient: digits.replace(/0+$/, ''),
            exponent: new B((sections[1] || '0').replace(/^\+/, '')).plus(String(digits.length - fraction - 1))};
    }
    /** Compare assigned decimal values without exponent expansion.
     * @param {string} a - Left decimal
     * @param {string} b - Right decimal
     * @returns {number} Negative, zero or positive ordering
     */
    function compareText(a, b) {
        const x = parts(a), y = parts(b);
        if (x.coefficient === '0' && y.coefficient === '0') return 0;
        if (x.negative !== y.negative) return x.negative ? -1 : 1;
        let order;
        if (x.coefficient === '0') order = -1;
        else if (y.coefficient === '0') order = 1;
        else {
            order = x.exponent.cmp(y.exponent);
            if (!order) {
                for (let i = 0; i < Math.max(x.coefficient.length, y.coefficient.length); i++) {
                    const p = x.coefficient[i] || '0', q = y.coefficient[i] || '0';
                    if (p !== q) { order = p < q ? -1 : 1; break; }
                }
            }
        }
        return order === 0 ? 0 : x.negative ? -order : order;
    }
    /** Compare two authenticated numbers.
     * @param {*} a - Left number
     * @param {*} b - Right number
     * @returns {number} Negative, zero or positive ordering
     */
    function compare(a, b) { return compareText(text(a), text(b)); }
    /** Test mathematical integrality, independent of decimal spelling.
     * @param {*} value - Admitted number
     * @returns {boolean} Whether the assigned value is an integer
     */
    function isInteger(value) {
        const p = parts(text(value));
        return p.coefficient === '0' || p.exponent.gte(String(p.coefficient.length - 1));
    }
    /** Floor and resolve a bounded array index, retaining tiny negative values.
     * @param {*} value - Admitted number
     * @param {number} length - Array length
     * @returns {number|undefined} Resolved index, or absent when out of range
     */
    function index(value, length) {
    // Compare exact bounds before narrowing to an array index. Thus a tiny
    // negative index floors to -1 instead of underflowing to positive zero.
        if (compare(value, -length) < 0 || compare(value, length) >= 0) return undefined;
        const p = parts(text(value));
        let integer = 0;
        if (p.exponent.gte('0')) {
            const point = Number(p.exponent.toFixed()) + 1;
            integer = Number((p.coefficient.slice(0, point) + '0'.repeat(Math.max(0, point - p.coefficient.length))));
            if (p.negative) integer = -integer;
        }
        if (compare(value, integer) < 0) integer--;
        if (integer < 0) integer += length;
        return integer >= 0 && integer < length ? integer : undefined;
    }
    // Deliberate integer control conversion with exact clipping. The bound must be
    // a safe native integer; this never converts an arbitrary value to binary64.
    /** Truncate a control value after exact comparison with safe native bounds.
     * @param {*} value - Admitted number
     * @param {number} lower - Inclusive safe-integer lower bound
     * @param {number} upper - Inclusive safe-integer upper bound
     * @returns {number} Clipped and truncated control
     */
    function clippedTrunc(value, lower, upper) {
        if (!Number.isSafeInteger(lower) || !Number.isSafeInteger(upper) || lower > upper) throw new TypeError('Invalid control bounds');
        if (compare(value, lower) <= 0) return lower;
        if (compare(value, upper) >= 0) return upper;
        const p = parts(text(value));
        if (p.exponent.lt('0')) return 0;
        const point = Number(p.exponent.toFixed()) + 1;
        const integer = Number(p.coefficient.slice(0, point) + '0'.repeat(Math.max(0, point - p.coefficient.length)));
        return p.negative ? -integer : integer;
    }
    /** Render a stable decimal spelling without expanding unbounded exponents.
     * @param {string} source - Decimal image
     * @returns {string} Stable assigned-value spelling
     */
    function format(source) {
        const p = parts(source), c = p.coefficient;
        if (c === '0') return '0';
        const sign = p.negative ? '-' : '';
        if (p.exponent.gte('-6') && p.exponent.lt('21')) {
            const point = Number(p.exponent.toFixed()) + 1;
            if (point <= 0) return sign + '0.' + '0'.repeat(-point) + c;
            if (point >= c.length) return sign + c + '0'.repeat(point - c.length);
            return sign + c.slice(0, point) + '.' + c.slice(point);
        }
        return sign + c[0] + (c.length > 1 ? '.' + c.slice(1) : '') +
        'e' + (p.exponent.gte('0') ? '+' : '') + p.exponent.toFixed();
    }
    /** Admit a JSON number; use native storage only when its decimal image agrees.
     * @param {string} source - JSON number text
     * @returns {*} Native number or privately authenticated exact carrier
     */
    function fromText(source) {
        if (!grammar.test(source)) throw new TypeError('Invalid JSON number');
        const native = Number(source);
        if (Number.isFinite(native) && compareText(source, String(native)) === 0) return native;
        const value = Object.freeze(new lossless.LosslessNumber(format(source)));
        numberValues.add(value);
        return value;
    }
    /** Implement decimal and radix string conversion without native rounding.
     * @param {string} source - Numeric string
     * @returns {*} Admitted number
     */
    function cast(source) {
        if (castGrammar.test(source)) return fromText(format(source));
        const m = /^(0[xX]([0-9a-fA-F]+)|0[oO]([0-7]+)|0[bB]([01]+))$/.exec(source);
        if (m) {
            const digits = m[2] || m[3] || m[4], radix = m[2] ? 16 : m[3] ? 8 : 2;
            let value = new B('0');
            for (const digit of digits) value = value.times(String(radix)).plus(String(parseInt(digit, radix)));
            return fromText(value.toFixed());
        }
        throw {code: 'D3030', value: source};
    }
    /** Report resource rejection; never substitute a lower-fidelity result.
     * @param {string} detail - Resource failure explanation
     */
    function fail(detail) { throw Object.assign(new Error('JSONata numeric work-limit: ' + detail), {code: 'U_NUMERIC_LIMIT'}); }
    /** Check an arithmetic intermediate against the instance work domain.
     * @param {Object} value - Library decimal
     * @param {Object} limits - Immutable work limits
     * @returns {Object} The unchanged decimal
     */
    function bounded(value, limits) {
        if (value.c.length > limits.maxDigits || (value.c[0] !== 0 && Math.abs(value.e) > limits.maxExponent)) fail('decimal work budget exceeded');
        return value;
    }
    /** Check before expanding the arithmetic representation of an operand.
     * @param {*} value - Admitted number
     * @param {Object} limits - Immutable work limits
     * @returns {Object} Bounded library decimal
     */
    function admit(value, limits) {
        const source = text(value), p = parts(source);
        if (p.coefficient.length > limits.maxDigits || p.exponent.abs().gt(String(limits.maxExponent))) fail('operand exceeds decimal work budget');
        return bounded(new B(format(source)), limits);
    }
    /** Divide at a specified significant-digit working precision, ties to even.
     * @param {Object} x - Dividend
     * @param {Object} y - Divisor
     * @param {number} precision - Working significant-digit count
     * @returns {Object} Assigned library quotient
     */
    function significantDivide(x, y, precision) {
        if (x.eq('0')) return new B('0');
        let exponent = x.e - y.e;
        let scaled = x.times(new B('1e' + -exponent));
        if (scaled.abs().lt(y.abs())) { exponent--; scaled = x.times(new B('1e' + -exponent)); }
        const Q = Big(); // eslint-disable-line new-cap -- isolated library constructor factory
        Q.strict = true; Q.DP = precision - 1; Q.RM = Big.roundHalfEven;
        return new B(new Q(scaled.toExponential()).div(new Q(y.toExponential())).toExponential()).times(new B('1e' + exponent));
    }
    /** Preserve terminating quotients; assign other quotients at 34 digits.
     * @param {Object} x - Dividend
     * @param {Object} y - Divisor
     * @param {Object} limits - Immutable work limits
     * @returns {Object} Assigned library quotient
     */
    function divide(x, y, limits) {
        if (y.eq('0')) throw {code: 'D1001', message: 'division by zero'};
        const precision = x.c.length + 4 * y.c.length + 2;
        if (precision > limits.maxDigits) fail('exact quotient probe exceeds decimal work budget');
        const candidate = significantDivide(x, y, precision);
        return candidate.times(y).eq(x) ? candidate : significantDivide(x, y, 34);
    }
    /** Compute a scaled significant-digit square root using the numerical library.
     * @param {Object} x - Radicand
     * @param {number} precision - Working significant-digit count
     * @returns {Object} Assigned library root
     */
    function significantSqrt(x, precision) {
        if (x.eq('0')) return new B('0');
        const exponent = Math.floor(x.e / 2);
        const scaled = x.times('1e' + (-2 * exponent));
        const Q = Big(); // eslint-disable-line new-cap -- isolated library constructor factory
        Q.strict = true; Q.DP = precision - 1; Q.RM = Big.roundHalfEven;
        return new B(new Q(scaled.toExponential()).sqrt().toExponential()).times('1e' + exponent);
    }
    /** Format a magnitude with an explicitly bounded decimal-place count.
     * @param {*} value - Admitted number
     * @param {number} places - Decimal-place count
     * @param {Object} limits - Immutable work limits
     * @returns {string} Fixed-place magnitude
     */
    function fixed(value, places, limits = defaultLimits) {
        if (!Number.isInteger(places) || places < 0 || places > limits.maxExponent) fail('format width exceeds decimal work budget');
        return admit(value, limits).abs().toFixed(places, Big.roundHalfEven);
    }
    /** Split the magnitude for a bounded scientific-format picture.
     * @param {*} value - Admitted number
     * @param {number} leading - Leading picture-digit count
     * @param {Object} limits - Immutable work limits
     * @returns {Object} Exact mantissa and bounded native exponent
     */
    function mantissa(value, leading, limits = defaultLimits) {
        if (!Number.isInteger(leading) || leading < 0 || leading > limits.maxDigits) fail('picture width exceeds decimal work budget');
        const x = admit(value, limits);
        if (x.eq('0')) return {value: 0, exponent: 0};
        const exponent = x.e - leading + 1;
        return {value: fromText(x.times('1e' + -exponent).toExponential()), exponent};
    }
    // Private helper for the existing integer picture engine; not a public carrier.
    /** Obtain a bounded library integer for the picture formatter.
     * @param {*} value - Admitted integer
     * @param {Object} limits - Immutable work limits
     * @returns {Object} Bounded library integer
     */
    function integer(value, limits = defaultLimits) {
        const x = admit(value, limits);
        if (!isInteger(value)) throw new TypeError('Expected an integer');
        return x;
    }
    /** Render an integer in radix 2 through 36 without a native-value conversion.
     * @param {*} value - Admitted integer
     * @param {number} radix - Base from 2 through 36
     * @param {Object} limits - Immutable work limits
     * @returns {string} Signed digit string
     */
    function radixString(value, radix, limits = defaultLimits) {
        let x = admit(value, limits);
        if (!isInteger(value) || !Number.isInteger(radix) || radix < 2 || radix > 36) throw new TypeError('Invalid integer radix conversion');
        const negative = x.lt('0');
        x = x.abs();
        const Q = Big(); // eslint-disable-line new-cap -- isolated library constructor factory
        Q.strict = true; Q.DP = 0; Q.RM = Big.roundDown;
        const digits = [];
        do {
            digits.push('0123456789abcdefghijklmnopqrstuvwxyz'[Number(x.mod(String(radix)).toFixed())]);
            x = new B(new Q(x.toFixed()).div(String(radix)).toFixed());
        } while (!x.eq('0'));
        return (negative ? '-' : '') + digits.reverse().join('');
    }
    /** Apply the official arithmetic policy under immutable instance work limits.
     * @param {string} op - Internal arithmetic operation
     * @param {*} left - Left admitted number
     * @param {*} right - Optional right admitted number
     * @param {number} places - Rounding scale when applicable
     * @param {Object} limits - Immutable work limits
     * @returns {*} Assigned admitted result
     */
    function calculate(op, left, right, places = 0, limits = defaultLimits) {
        const x = admit(left, limits), y = typeof right === 'undefined' ? undefined : admit(right, limits);
        let value;
        switch (op) {
            case '+': value = x.plus(y); break;
            case '-': value = x.minus(y); break;
            case '*': value = x.times(y); break;
            case '/': value = divide(x, y, limits); break;
            case '%': if (y.eq('0')) throw {code: 'D1001', message: 'modulo by zero'}; value = x.mod(y); break;
            case 'neg': value = x.neg(); break;
            case 'abs': value = x.abs(); break;
            case 'floor': value = x.round(0, Big.roundDown); if (x.lt(value)) value = value.minus('1'); break;
            case 'ceil': value = x.round(0, Big.roundDown); if (x.gt(value)) value = value.plus('1'); break;
            case 'round':
                if (!Number.isInteger(places) || Math.abs(places) > limits.maxExponent) fail('round scale exceeds decimal work budget');
                value = x.times('1e' + places).round(0, Big.roundHalfEven).times('1e' + -places); break;
            case 'sqrt': {
                if (x.lt('0')) throw {code: 'D3060', message: 'square root of a negative number'};
                const precision = x.c.length + 2;
                if (precision > limits.maxDigits) fail('exact root probe exceeds decimal work budget');
                const probe = significantSqrt(x, precision);
                value = probe.times(probe).eq(x) ? probe : significantSqrt(x, 34);
                break;
            }
            case 'pow': {
                if (!isInteger(right)) {
                    if (x.lt('0') || (x.eq('0') && y.lt('0'))) throw {code:'D3061',message:'Power has no finite real result'};
                    if (x.eq('0') || x.eq('1')) {value=x;break;}
                    if (limits.maxDigits < 120) fail('fractional power requires the fixed guard work budget');
                    let assigned;
                    for (const precision of [80,120]) {
                        const D=Decimal.clone({precision,rounding:Decimal.ROUND_HALF_EVEN,minE:-1000000,maxE:1000000});
                        const probe=new D(x.toExponential()).pow(new D(y.toExponential()));
                        if (!probe.isFinite() || probe.isZero()) fail('power exceeds the supported work domain');
                        const result=new B(probe.toSignificantDigits(34,Decimal.ROUND_HALF_EVEN).toExponential());
                        bounded(result,limits);
                        if (assigned && !assigned.eq(result)) fail('fractional power rounding is unstable at the fixed guard precisions');
                        assigned=result;
                    }
                    value=assigned;break;
                }
                const exponent = Number(text(right));
                if (!Number.isSafeInteger(exponent) || compareText(text(right), String(exponent)) !== 0) fail('integral exponent exceeds the supported work domain');
                if (Math.abs(exponent) > limits.maxDigits) fail('power exceeds decimal work budget');
                let count = Math.abs(exponent), base = x;
                value = new B('1');
                while (count > 0) {
                    if (count % 2) value = bounded(value.times(base), limits);
                    count = Math.floor(count / 2);
                    if (count) base = bounded(base.times(base), limits);
                }
                if (exponent < 0) value = divide(new B('1'), value, limits);
                break;
            }
            default: throw new Error('Unknown numerical operation: ' + op);
        }
        return fromText(bounded(value, limits).toExponential());
    }

    // Reuse the same pure standard-JSON codec family as ordinary SDK carriage.
    // Prepare a hook-free image first. lossless-json's serializer identifies numbers
    // by a forgeable `isLosslessNumber` data field and cannot preserve such objects.
    /** Serialize exact numbers through a hook-free JSON image.
     * @param {*} value - Value to serialize
     * @param {Function} replacer - Optional trusted internal conversion
     * @param {number|string} space - JSON indentation
     * @returns {string|undefined} JSON text, or absent for an absent value
     */
    function stringify(value, replacer, space) {
        const active = new Set();
        let remaining = 10000000 + 513;
        /** Copy own data members, rejecting cycles, accessors and excessive depth.
         * @param {*} value - Current value
         * @param {string} key - Own member name
         * @param {Object} parent - Parent supplied to the trusted replacer
         * @param {number} depth - Materialization depth
         * @returns {*} Hook-free image
         */
        function walk(value, key, parent, depth) {
            if (--remaining < 0 || depth > 512) fail('JSON materialization budget exceeded');
            if (typeof replacer === 'function') value = replacer.call(parent, key, value);
            if (isNumeric(value)) return typeof value === 'number' ? value : rawJSON(text(value));
            if (value === null || typeof value !== 'object') return value;
            if (active.has(value)) throw new TypeError('Cyclic JSON value');
            active.add(value);
            const result = Array.isArray(value) ? [] : Object.create(null);
            const keys = Array.isArray(value) ? Array.from({length: value.length}, (_, i) => String(i)) : Object.keys(value);
            for (const childKey of keys) {
                const descriptor = Object.getOwnPropertyDescriptor(value, childKey);
                if (descriptor && !Object.prototype.hasOwnProperty.call(descriptor, 'value')) throw new TypeError('JSON serialization cannot invoke accessors');
                const child = walk(descriptor ? descriptor.value : undefined, childKey, value, depth+1);
                Object.defineProperty(result, childKey, {value: child, writable: true, enumerable: true, configurable: true});
            }
            active.delete(value);
            return result;
        }
        return jsonStringify(walk(value, '', {'': value}, 0), undefined, space);
    }

    module.exports = {isNumeric, isInteger, index, clippedTrunc, text, compare, compareText, fromText, format, cast, calculate, fixed, mantissa, integer, radixString, defaultLimits, normalizeLimits,
        parse: source => lossless.parse(source, undefined, {parseNumber: fromText}),
        stringify};
})();
