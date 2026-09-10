'use strict';
// Node 8-compatible source for the retained artifact/platform qualification.
var assert = require('assert');
var path = require('path');
var root = process.argv[2];
var variants = ['json-executor.js','json-executor.min.js','json-executor-es5.js','json-executor-es5.min.js'];
var tests = [
    ['9007199254740993 = 9007199254740992',false],
    ['$string(0.1 + 0.2)','0.3'],
    ['$string(9223372036854775807)','9223372036854775807'],
    ['$string($sqrt(4))','2'],
    ['$string($power(16, 0.25))','2'],
    ['$base64encode("\u00e9")','6Q=='],
    ['$base64decode("8J+YgA==")','\ud83d\ude00'],
    ['$uppercase("\ud803\udd70")','\ud803\udd50'],
    ['$lowercase("A\u0345\u03a3")','a\u0345\u03c2'],
    ['$fromMillis(253402300800000)','+010000-01-01T00:00:00.000Z'],
    ['$match("ab12", /(?<=ab)\\d+/).match','12'],
    ['$lookup($eval($string({"value":[[1],[2]]})),"value")',[[1],[2]]],
    ['$string($eval("1e400"))','1e+400']
];
async function main() {
    var observations=[];
    for(var i=0;i<variants.length;i++) {
        var api=require(path.join(root,variants[i]));
        assert.deepEqual(Object.keys(api),['createJSONExecutor']);
        var executor=api.createJSONExecutor();
        for(var j=0;j<tests.length;j++)assert.deepEqual(JSON.parse(await executor.evaluate(tests[j][0],'{}')),tests[j][1]);
        assert.equal(await executor.evaluate('$','{"id":9223372036854775807}'),'{"id":9223372036854775807}');
        var rejected=false;try{await executor.evaluate('{"bad":function(){1}}','{}');}catch(error){rejected=true;}
        assert.equal(rejected,true);observations.push({entry:variants[i],cases:tests.length+2});
    }
    if(Number(process.versions.node.split('.')[0])>=18) {
        var pool=require(path.join(root,'node-executor.js')).createNodeExecutor({workers:1,timeout:10000});
        try {
            assert.equal(await pool.evaluate('0.1+0.2','{}'),'0.3');
            var controller=new AbortController();controller.abort();
            var cancelled=false;try{await pool.evaluate('$','{}',{signal:controller.signal});}catch(error){cancelled=true;}
            assert.equal(cancelled,true);
            assert.equal(await pool.evaluate('id','{"id":9223372036854775807}'),'9223372036854775807');
        }finally{await pool.close();}
        observations.push({entry:'node-executor.js',cases:3});
    }
    console.log(JSON.stringify({node:process.version,root:root,observations:observations}));
}
main().catch(function(error){console.error(error);process.exitCode=1;});
