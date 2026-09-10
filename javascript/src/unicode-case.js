(function () {
    'use strict';
    const data = require('./unicode-data.json');

    /** Look up a sorted Unicode property range.
     * @param {number} cp - Code point
     * @param {Array} ranges - Sorted inclusive ranges
     * @returns {boolean} Whether the property holds
     */
    function contains(cp, ranges) {
        let low = 0, high = ranges.length;
        while (low < high) {
            const middle = (low + high) >>> 1, range = ranges[middle];
            if (cp < range[0]) high = middle;
            else if (cp > range[1]) low = middle + 1;
            else return true;
        }
        return false;
    }

    /** Apply Unicode 16 default full casing without host-version dependence.
     * @param {string} source - Original UTF-16 string, including admitted lone units
     * @param {boolean} lower - Lowercase when true, uppercase otherwise
     * @returns {string} Case-mapped string without normalization
     */
    function convert(source, lower) {
        if (/^[\x00-\x7f]*$/.test(source)) return lower ? source.toLowerCase() : source.toUpperCase(); // eslint-disable-line no-control-regex -- deliberately includes NUL in the ASCII fast path
        const characters = Array.from(source), mapping = lower ? data.lower : data.upper;
        const following = [];
        let next = false;
        // Possessive Case_Ignorable matching has priority over Cased where
        // the properties overlap (notably U+0345); contexts use original data.
        if (lower) for (let i = characters.length - 1; i >= 0; i--) {
            following[i] = next;
            const cp = characters[i].codePointAt(0);
            if (!contains(cp, data.ignorable)) next = contains(cp, data.cased);
        }
        let previous = false;
        return characters.map((character, index) => {
            const cp = character.codePointAt(0);
            const result = lower && cp === 0x3a3 && previous && !following[index] ? '\u03c2' : (mapping[cp] || character);
            if (lower && !contains(cp, data.ignorable)) previous = contains(cp, data.cased);
            return result;
        }).join('');
    }

    module.exports = {upper: source => convert(source, false), lower: source => convert(source, true), version: data.version};
})();
