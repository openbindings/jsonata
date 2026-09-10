'use strict';
const assert = require('assert');
const {EventEmitter} = require('events');
const workers = require('worker_threads');

// Exercise the actual private worker entry in the instrumented parent process.
// Real threads separately qualify isolation, cancellation and engine execution;
// these deliberate dependency failures cover the message protocol's fallbacks.
describe('Private worker message protocol', function() {
    it('returns exact text and serializable errors, including non-Error throws', async function() {
        const entry=require.resolve('../src/json-worker');
        const dependency=require('../src/json-executor'),original=dependency.createJSONExecutor;
        const saved={parentPort:workers.parentPort,workerData:workers.workerData,cached:require.cache[entry]};
        const parent=new EventEmitter(),received=[];
        parent.postMessage=message=>received.push(message);
        let problem;
        try {
            workers.parentPort=parent;workers.workerData={timeout:1234};
            dependency.createJSONExecutor=options=>{
                assert.strictEqual(options.timeout,1234);
                return {async evaluate(expression,input,options){
                    assert.strictEqual(expression,'$x');assert.strictEqual(input,'{}');
                    assert.strictEqual(options.bindingsJSON,'{"x":9007199254740993}');
                    if(problem!==undefined)throw problem;
                    return '9007199254740993';
                }};
            };
            delete require.cache[entry];require(entry);
            const handler=parent.listeners('message')[0];
            for(const failure of [undefined,new TypeError('typed failure'),{code:'U_TEST'},null,'failure']) {
                problem=failure;
                await handler({id:received.length,expression:'$x',inputJSON:'{}',bindingsJSON:'{"x":9007199254740993}'});
            }
            assert.deepStrictEqual(received,[
                {id:0,value:'9007199254740993'},
                {id:1,error:{name:'TypeError',message:'typed failure',code:undefined}},
                {id:2,error:{name:'Error',message:'JSONata evaluation failed',code:'U_TEST'}},
                {id:3,error:{name:'Error',message:'JSONata evaluation failed',code:null}},
                {id:4,error:{name:'Error',message:'JSONata evaluation failed',code:undefined}}
            ]);
        } finally {
            dependency.createJSONExecutor=original;workers.parentPort=saved.parentPort;workers.workerData=saved.workerData;
            delete require.cache[entry];if(saved.cached)require.cache[entry]=saved.cached;
        }
    });
});
