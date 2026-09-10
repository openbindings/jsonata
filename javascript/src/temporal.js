(function () {
    'use strict';

    // Calendar operations stay in the existing Date-compatible millisecond
    // domain. Use UTC setters: Date.UTC incorrectly maps years 0..99 to 1900..1999.
    /** Construct a UTC instant without the two-digit-year host convenience.
     * @param {number} year - Full Gregorian year
     * @param {number} month - Zero-based month
     * @param {number} day - Day in month
     * @param {number} hour - Hour
     * @param {number} minute - Minute
     * @param {number} second - Second
     * @param {number} millis - Millisecond
     * @returns {number} UTC milliseconds
     */
    function utcMillis(year, month, day = 1, hour = 0, minute = 0, second = 0, millis = 0) {
        // Gregorian calendars repeat every 400 years. Let Date perform calendar
        // normalization in a safe cycle, then translate the epoch. TimeClip must
        // be applied to the final UTC instant, not to a local wall clock before
        // its offset is subtracted (which could be just outside Date's range).
        const anchorYear = 2000 + ((year % 400) + 400) % 400;
        const date = new Date(0);
        date.setUTCFullYear(anchorYear, month, day);
        date.setUTCHours(hour, minute, second, millis);
        return date.getTime() + (year - anchorYear) / 400 * 146097 * 86400000;
    }

    /** Validate a numeric offset; timezone names are not a closed-language input.
     * @param {string} value - Signed hours and minutes, with optional colon
     * @returns {number} Signed offset minutes
     */
    function offsetMinutes(value) {
        const match = /^([+-])(\d{2}):?(\d{2})$/.exec(value);
        if (!match || Number(match[2]) > 23 || Number(match[3]) > 59) throw {code:'D3110', value};
        return (match[1] === '-' ? -1 : 1) * (Number(match[2]) * 60 + Number(match[3]));
    }

    // ISO calendar interchange forms; compact numeric offsets remain admitted.
    // Absent timezone uses UTC, not the machine's ambient timezone. Submillisecond
    // fractions are deliberately truncated, matching the millisecond time domain.
    const iso = /^([+-]\d{6}|\d{4})(?:-(\d{2})(?:-(\d{2}))?)?(?:T(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d+))?)?(Z|[+-]\d{2}:?\d{2})?)?$/;
    /** Parse and validate before host calendar normalization can hide bad fields.
     * @param {string} value - Calendar timestamp
     * @returns {number} UTC milliseconds
     */
    function parseISO(value) {
        const parts = iso.exec(value);
        if (!parts || parts[1] === '-000000') throw {code:'D3110', value};
        const year = Number(parts[1]), month = Number(parts[2] || 1), day = Number(parts[3] || 1);
        const hour = Number(parts[4] || 0), minute = Number(parts[5] || 0), second = Number(parts[6] || 0);
        const fraction = parts[7] || '';
        const millis = Number((fraction + '000').substring(0,3));
        const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
        const days = [31,leap ? 29 : 28,31,30,31,30,31,31,30,31,30,31];
        if (month < 1 || month > 12 || day < 1 || day > days[month - 1] || hour > 24 || minute > 59 || second > 59 || (hour === 24 && (minute !== 0 || second !== 0 || /[1-9]/.test(fraction)))) throw {code:'D3110', value};
        const offset = parts[8] && parts[8] !== 'Z' ? offsetMinutes(parts[8]) : 0;
        const result = utcMillis(year,month - 1,day,hour,minute,second,millis) - offset * 60000;
        if (!Number.isFinite(result) || Math.abs(result) > 8640000000000000) throw {code:'D3110', value};
        return result;
    }

    /** Render without a year-width picture that would truncate expanded years.
     * @param {number} millis - UTC milliseconds
     * @param {number} offset - Signed offset minutes
     * @returns {string} ISO calendar timestamp
     */
    function formatISO(millis, offset) {
        const shifted = millis + offset * 60000;
        if (Math.abs(shifted) > 8640000000000000) throw {code:'D3110', value:millis};
        const iso = new Date(shifted).toISOString();
        if (offset === 0) return iso;
        const absolute = Math.abs(offset);
        const zone = (offset < 0 ? '-' : '+') + ('0' + Math.floor(absolute / 60)).slice(-2) + ':' + ('0' + absolute % 60).slice(-2);
        return iso.slice(0,-1) + zone;
    }

    module.exports = {utcMillis, offsetMinutes, parseISO, formatISO};
})();
