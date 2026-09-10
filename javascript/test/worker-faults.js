'use strict';
const assert = require('assert');
const {EventEmitter} = require('events');
const workers = require('worker_threads');

// Private lifecycle fault injection, not alternate evaluator answers. Real
// worker execution/termination is exercised separately in json-executor.js.
function controlledPool(behavior) {
    const created=[];
    class FakeWorker extends EventEmitter {
        constructor() { super(); if(behavior.construct)throw Error('construct failure');created.push(this); }
        ref() {} unref() {}
        postMessage(message) { if(behavior.post)throw Error('post failure');this.request=message; }
        terminate() {
            if(behavior.rejectTerminate)return Promise.reject(Error('terminate failure'));
            if(behavior.pendingTerminate)return new Promise(resolve=>{this.finish=()=>{this.emit('exit',1);resolve(1)}});
            return Promise.resolve(1);
        }
        reply(value='1') { this.emit('message',{id:this.request.id,value}); }
    }
    const target=require.resolve('../src/node-executor');const old=workers.Worker,cached=require.cache[target];
    try { workers.Worker=FakeWorker;delete require.cache[target];return {factory:require(target).createNodeExecutor,created}; }
    finally { workers.Worker=old;delete require.cache[target];if(cached)require.cache[target]=cached; }
}

describe('Worker lifecycle fault controls', function() {
    it('surfaces constructor and message-transfer failures', async function() {
        for(const key of ['construct','post']) {
            const {factory}=controlledPool({[key]:true});const executor=factory();
            await assert.rejects(executor.evaluate('1','{}'),new RegExp(key+' failure'));
            await executor.close();
        }
    });
    it('retires error/exit workers and permits a subsequent healthy job', async function() {
        for(const event of ['error','exit']) {
            const {factory,created}=controlledPool({});const executor=factory({workers:1});
            const failed=assert.rejects(executor.evaluate('1','{}'),/failed|exited/);
            created[0].emit(event,event==='error'?Error('worker failed'):3);await failed;
            const next=executor.evaluate('2','{}');created[1].reply('2');assert.strictEqual(await next,'2');
            await executor.close();
        }
    });
    it('ignores unmatched/duplicate replies and stale cancellation callbacks', async function() {
        const {factory,created}=controlledPool({});const executor=factory();let onAbort;
        const signal={aborted:false,addEventListener(_,fn){onAbort=fn},removeEventListener(){}};
        const pending=executor.evaluate('1','{}',{signal});
        created[0].emit('message',{id:-1,value:'unmatched'});created[0].reply();
        assert.strictEqual(await pending,'1');created[0].reply();onAbort();
        await assert.rejects(executor.evaluate('1','{}',{signal:{aborted:true}}),/cancelled/);
        await executor.close();
    });
    it('keeps error listeners while cancellation and close await the same exit', async function() {
        const behavior={pendingTerminate:true};const {factory,created}=controlledPool(behavior);const executor=factory({workers:1});let onAbort;
        const signal={aborted:false,addEventListener(_,fn){onAbort=fn},removeEventListener(){}};
        const pending=assert.rejects(executor.evaluate('1','{}',{signal}),/cancelled/);onAbort();
        const closing=executor.close();
        // EventEmitter would throw here if retire removed the error listener.
        created[0].emit('error',Error('late termination error'));
        created[0].finish();await Promise.all([pending,closing]);
    });
    it('does not double-settle a task when worker termination itself rejects', async function() {
        const {factory,created}=controlledPool({rejectTerminate:true});const executor=factory();let onAbort;
        const signal={aborted:false,addEventListener(_,fn){onAbort=fn},removeEventListener(){}};
        const pending=assert.rejects(executor.evaluate('1','{}',{signal}),/cancelled/);onAbort();await pending;
        await new Promise(resolve=>setImmediate(resolve));
        assert.strictEqual(created.length,1);await executor.close();
    });
    it('handles termination rejection on error, exit and transfer-failure paths', async function() {
        for(const event of ['error','exit','post']) {
            const {factory,created}=controlledPool({rejectTerminate:true,post:event==='post'});const executor=factory();
            const pending=assert.rejects(executor.evaluate('1','{}'),/failed|exited|post failure/);
            if(event!=='post')created[0].emit(event,event==='error'?Error('worker failed'):3);
            await pending;await new Promise(resolve=>setImmediate(resolve));await executor.close();
        }
    });
    it('closes both running and queued work without waiting for a queue timeout', async function() {
        const {factory}=controlledPool({});const executor=factory({workers:1});
        const active=assert.rejects(executor.evaluate('1','{}'),/closed/);
        const queued=assert.rejects(executor.evaluate('2','{}'),/closed/);
        await executor.close();await Promise.all([active,queued]);
    });
});
