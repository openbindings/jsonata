'use strict';
// Audit-only preload; records loaded build modules, not a production hook.
const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '..', 'javascript');
const output = process.env.JSONATA_BUILD_TRACE;
if (!output) throw Error('JSONATA_BUILD_TRACE is required');
process.on('exit', () => {
    fs.writeFileSync(output, JSON.stringify(Object.keys(require.cache).map(file => path.relative(root, file)).sort(), null, 2) + '\n');
});
