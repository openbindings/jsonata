'use strict';
// Test-fixture transport only. No HTTP facility is exposed by the runtime.
const http = require('http');
module.exports = function httpget(url) {
    return new Promise((resolve, reject) => {
        const request = http.get(url, response => {
            let body = '';
            response.setEncoding('utf8');
            response.on('data', chunk => { body += chunk; });
            response.once('error', reject);
            response.once('aborted', () => reject(new Error('Fixture response aborted')));
            response.once('end', () => {
                try { resolve(JSON.parse(body)); } catch (error) { reject(error); }
            });
        });
        request.once('error', reject);
        request.setTimeout(2000, () => request.destroy(new Error('Fixture request timeout')));
    });
};
