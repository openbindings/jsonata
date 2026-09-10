'use strict';
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert');
const {cases}=require('./documentation-expectations.json');
assert.strictEqual(cases.length,3);
for(const c of cases) {
    const bytes=fs.readFileSync(path.join(__dirname,'test-suite/groups',c.file));
    assert.strictEqual(crypto.createHash('sha256').update(bytes).digest('hex'),c.sourceSHA256);
    assert.strictEqual(JSON.parse(bytes)[c.index].expr,c.expression);
    assert(c.authority && c.reason);
}
module.exports=function(file,index,spec) {
    if(process.env.JSONATA_REFERENCE_EXPECTATIONS==='1')return;
    const c=cases.find(c=>c.file===file && c.index===index);
    if(!c)return;
    for(const key of ['result','error','code','undefinedResult'])delete spec[key];
    spec.result=JSON.parse(c.resultJSON);
};
