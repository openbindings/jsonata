(function () {
    'use strict';

    // Optional Node worker integration; the base evaluator/browser bundle does not
    // import worker_threads. A cancelled running job settles only AFTER terminate()
    // confirms worker exit. There is no abandoned background regex computation.
    const {Worker} = require('worker_threads');
    const {optionsFor} = require('./execution-options');

    /** Construct an optional bounded Node worker pool; the caller owns close().
     * @returns {Object} Text evaluation and asynchronous close interface
     */
    function createNodeExecutor({workers = 2, maxPending = 64, maxWorkerHeapMB = 128, ...options} = {}) {
        if (!Number.isSafeInteger(workers) || workers < 1 || workers > 32 || !Number.isSafeInteger(maxPending) || maxPending < 1) throw new TypeError('Invalid worker or queue budget');
        if (!Number.isSafeInteger(maxWorkerHeapMB) || maxWorkerHeapMB < 16) throw new TypeError('Worker heap budget must be an integer of at least 16 MiB');
        const limits = optionsFor(options);
        if (limits.timeout > 2147483647) throw new RangeError('Node JSONata timeout exceeds the timer limit of 2147483647 milliseconds');
        const slots = new Set(), queue = [], tasks = new Set();
        let closed = false, nextID = 0;
        /** Complete a task once and release its queue, timer and signal references.
         * @param {Object} task - Pending task
         * @param {Error} error - Optional failure
         * @param {string} value - Optional successful JSON text
         */
        function settle(task, error, value) {
            if (task.done) return;
            task.done = true;
            clearTimeout(task.timer);
            if (task.signal) task.signal.removeEventListener('abort', task.onAbort);
            tasks.delete(task);
            const index = queue.indexOf(task); if (index !== -1) queue.splice(index, 1);
            if (error) task.reject(error); else task.resolve(value);
        }
        /** Confirm worker termination before settling its active task.
         * @param {Object} slot - Worker slot
         * @param {Error} error - Failure to report after exit
         */
        async function retire(slot, error) {
            if (slot.retiring) return slot.retiring;
            slot.retiring = (async () => {
                // Keep lifecycle listeners until exit: an asynchronous worker
                // error during termination must not become an unhandled event.
                slot.worker.removeAllListeners('message');
                try { await slot.worker.terminate(); } finally {
                    slot.worker.removeAllListeners();
                    slots.delete(slot);
                    if (slot.task) settle(slot.task, error);
                    drain();
                }
            })();
            return slot.retiring;
        }
        /** Cancel queued work immediately or terminate the worker running it.
         * @param {Object} task - Pending task
         * @param {Error} error - Cancellation reason
         */
        function cancel(task, error) {
            if (task.done) return;
            if (task.slot) { void retire(task.slot, error).catch(failure => settle(task,failure)); } else { settle(task, error); drain(); }
        }
        /** Start one private worker and install its lifecycle handlers.
         * @returns {Object} New worker slot
         */
        function createSlot() {
            const worker = new Worker(require.resolve('./json-worker'), {workerData:limits, resourceLimits:{maxOldGenerationSizeMb:maxWorkerHeapMB, maxYoungGenerationSizeMb:Math.min(32,Math.floor(maxWorkerHeapMB/4))}});
            const slot = {worker, task:undefined, retiring:undefined};
            slots.add(slot);
            worker.on('message', message => {
                const task = slot.task;
                if (!task || task.done || task.id !== message.id || slot.retiring) return;
                slot.task = undefined; task.slot = undefined;
                worker.unref();
                const error = message.error ? Object.assign(new Error(message.error.message), {name:message.error.name, code:message.error.code}) : undefined;
                settle(task, error, message.value); drain();
            });
            worker.on('error', error => { void retire(slot,error).catch(() => {}); });
            const onExit = code => { if (!slot.retiring) { void retire(slot,new Error('JSONata worker exited: '+code)).catch(() => {}); } };
            worker.on('exit', onExit);
            return slot;
        }
        /** Assign queued work to idle workers without exceeding pool cardinality. */
        function drain() {
            if (closed) return;
            while (queue.length) {
                let slot = Array.from(slots).find(slot => !slot.task && !slot.retiring);
                if (!slot && slots.size < workers) {
                    try { slot = createSlot(); } catch (error) { settle(queue[0],error); continue; }
                }
                if (!slot) return;
                const task = queue.shift();
                slot.task = task; task.slot = slot;
                slot.worker.ref();
                try { slot.worker.postMessage({id:task.id, expression:task.expression, inputJSON:task.inputJSON, bindingsJSON:task.bindingsJSON}); } catch (error) { void retire(slot,error).catch(() => {}); }
            }
        }
        return Object.freeze({
            evaluate(expression, inputJSON, {bindingsJSON, signal} = {}) {
                if (closed) return Promise.reject(new Error('JSONata executor is closed'));
                if (signal && signal.aborted) return Promise.reject(signal.reason || new Error('Evaluation cancelled'));
                if (tasks.size >= maxPending) return Promise.reject(new RangeError('JSONata pending work budget exceeded'));
                // Bound copies BEFORE placing text in the queue or worker message.
                if (typeof expression !== 'string' || expression.length > limits.maxExpressionLength || typeof inputJSON !== 'string' || (bindingsJSON !== undefined && typeof bindingsJSON !== 'string') || inputJSON.length + (bindingsJSON ? bindingsJSON.length : 0) > limits.maxInputLength) return Promise.reject(new RangeError('JSONata expression/input length budget exceeded'));
                return new Promise((resolve, reject) => {
                    const task = {id:++nextID, expression, inputJSON, bindingsJSON, signal, resolve, reject, done:false};
                    task.onAbort = () => cancel(task,signal.reason || new Error('Evaluation cancelled'));
                    task.timer = setTimeout(() => cancel(task,Object.assign(new Error('JSONata execution deadline exceeded'),{code:'U_TIMEOUT'})),limits.timeout);
                    if (signal) signal.addEventListener('abort',task.onAbort,{once:true});
                    tasks.add(task); queue.push(task); drain();
                });
            },
            async close() {
                closed = true;
                const error = new Error('JSONata executor is closed');
                for (const task of queue.slice()) settle(task,error);
                await Promise.all(Array.from(slots,slot => retire(slot,error)));
            }
        });
    }

    module.exports = {createNodeExecutor};
})();
