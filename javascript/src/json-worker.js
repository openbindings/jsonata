(function () {
    'use strict';
    const {parentPort, workerData} = require('worker_threads');
    const {createJSONExecutor} = require('./json-executor');
    const executor = createJSONExecutor(workerData);
    parentPort.on('message', async message => {
        try {
            const value = await executor.evaluate(message.expression, message.inputJSON, {bindingsJSON:message.bindingsJSON});
            parentPort.postMessage({id:message.id, value});
        } catch (error) {
            parentPort.postMessage({id:message.id, error:{name:error && error.name || 'Error', message:error && error.message || 'JSONata evaluation failed', code:error && error.code}});
        }
    });
})();
