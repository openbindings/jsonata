(function () {
    'use strict';
    // Backend helpers remain private even though the upstream-derived source
    // tree and its own tests retain broader internal entry points.
    const {createJSONataExecutor} = require('./json-executor');
    module.exports = Object.freeze({createJSONataExecutor});
})();
