// Verify pinned generators in disposable output directories; never rewrite the
// candidate's generated sources. Inputs may be cached or independently fetched
// from the URLs recorded by the generators; every source checksum is enforced.
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const [unicodeInput,regexInput]=process.argv.slice(2).map(value=>path.resolve(value));
if(!unicodeInput||!regexInput)throw Error('Usage: node maintenance/verify-generated.mjs UNICODE_INPUT_DIRECTORY REGONAUT_MODULE_DIRECTORY');
const temporary=fs.mkdtempSync(path.join(os.tmpdir(),'jsonata-generated-'));
const engine=path.join(root,'go/internal/engine');
const comparisons=[];
const sha=bytes=>createHash('sha256').update(bytes).digest('hex');
const copy=(source,target)=>{fs.mkdirSync(path.dirname(target),{recursive:true});fs.copyFileSync(source,target);};
const compare=(expected,actual)=>{
 const bytes=fs.readFileSync(expected);
 assert.deepEqual(fs.readFileSync(actual),bytes,path.relative(root,expected));
 comparisons.push({file:path.relative(root,expected),sha256:sha(bytes)});
};
const run=(command,args,cwd)=>execFileSync(command,args,{cwd,encoding:'utf8',timeout:120000});

const js=path.join(temporary,'javascript');
copy(path.join(root,'javascript/scripts/generate-casing.cjs'),path.join(js,'scripts/generate-casing.cjs'));
run(process.execPath,['scripts/generate-casing.cjs',unicodeInput],js);
for(const file of ['src/unicode-data.json','unicode/LICENSE','unicode/SOURCE.json'])compare(path.join(root,'javascript',file),path.join(js,file));

const goOutput=path.join(temporary,'go-unicode');
run('go',['run','scripts/generate-casing.go',unicodeInput,goOutput],engine);
for(const file of ['data.go','LICENSE','SOURCE.json'])compare(path.join(engine,'internal/unicodecase',file),path.join(goOutput,file));

const regex=path.join(temporary,'engine');
copy(path.join(engine,'scripts/vendor-regex.mjs'),path.join(regex,'scripts/vendor-regex.mjs'));
run(process.execPath,['scripts/vendor-regex.mjs',regexInput],regex);
for(const file of ['regonaut.go','unicode_generated.go','LICENSE','README.md','UPSTREAM.json'])compare(path.join(engine,'internal/thirdparty/regonaut',file),path.join(regex,'internal/thirdparty/regonaut',file));

console.log(JSON.stringify({status:'PASS',temporary,comparisons,sourceMutated:false},null,2));
