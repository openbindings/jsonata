import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {createRequire} from 'node:module';
import {pathToFileURL,fileURLToPath} from 'node:url';
import {createHash} from 'node:crypto';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const temporary=fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(),'jsonata-npm-consumer-')));
const artifacts=path.join(temporary,'artifacts');fs.mkdirSync(artifacts);
const pack=JSON.parse(execFileSync('npm',['pack','--ignore-scripts','--json','--pack-destination',artifacts],{cwd:path.join(root,'javascript'),encoding:'utf8'}))[0];
for(const item of pack.files)assert(!/^(src|test|scripts|node_modules)\//.test(item.path)&&!/^jsonata(?:[.-])/.test(item.path),'Private package file: '+item.path);
const archive=path.join(artifacts,pack.filename);
fs.writeFileSync(path.join(temporary,'package.json'),JSON.stringify({private:true,type:'module'}));
execFileSync('npm',['install','--ignore-scripts','--no-audit','--no-fund',archive],{cwd:temporary,stdio:'inherit',timeout:120000});
const require=createRequire(path.join(temporary,'package.json'));
const entry=require.resolve('@openbindings/jsonata');
assert(entry.startsWith(temporary+path.sep));
const cjs=require('@openbindings/jsonata');
const esmConsumer=path.join(temporary,'esm-consumer.mjs');
fs.writeFileSync(esmConsumer,"import * as api from '@openbindings/jsonata'; export default api;\n");
const esm=(await import(pathToFileURL(esmConsumer))).default;
assert.equal(typeof esm.createJSONExecutor,'function');
assert.deepEqual(Object.keys(cjs),['createJSONExecutor']);
for(const api of [cjs,esm]){
  const executor=api.createJSONExecutor();
  assert.equal(await executor.evaluate('id','{"id":9007199254740993}'),'9007199254740993');
  await assert.rejects(executor.evaluate('function(){1}','null'));
  await assert.rejects(executor.evaluate('"\\u-001"','null'));
}
for(const target of ['src/jsonata.js','jsonata.js'])assert.throws(()=>require('@openbindings/jsonata/'+target),{code:'ERR_PACKAGE_PATH_NOT_EXPORTED'});
const worker=require('@openbindings/jsonata/node').createNodeExecutor({workers:1});
try{assert.equal(await worker.evaluate('0.1+0.2','null'),'0.3');}finally{await worker.close();}
execFileSync(process.execPath,[path.join(root,'maintenance/platform-consumer.cjs'),path.dirname(entry)],{stdio:'inherit'});
console.log(JSON.stringify({status:'PASS',temporary,archive,sha256:createHash('sha256').update(fs.readFileSync(archive)).digest('hex'),files:pack.files.length,installedSurface:true,privateExportsDenied:true}));
