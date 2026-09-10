// Download only pinned generator inputs; verify checksums before use.
import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const destination=path.resolve(process.argv[2]);fs.mkdirSync(destination,{recursive:true});
const unicode=path.join(destination,'unicode');fs.mkdirSync(unicode,{recursive:true});
const sources=JSON.parse(fs.readFileSync(path.join(root,'javascript/unicode/SOURCE.json'))).sources;
for(const [name,[url,hash]] of Object.entries(sources)){
  const file=path.join(unicode,name);
  let bytes;
  if(fs.existsSync(file))bytes=fs.readFileSync(file);
  else{const response=await fetch(url);assert(response.ok,url);bytes=Buffer.from(await response.arrayBuffer());}
  assert.equal(createHash('sha256').update(bytes).digest('hex'),hash,'Changed upstream input '+name);
  if(!fs.existsSync(file))fs.writeFileSync(file,bytes);
}
const pinned=JSON.parse(fs.readFileSync(path.join(root,'go/internal/engine/internal/thirdparty/regonaut/UPSTREAM.json')));
const module=JSON.parse(execFileSync('go',['mod','download','-json',pinned.module+'@'+pinned.version],{cwd:path.join(root,'go'),env:{...process.env,GOWORK:'off'},encoding:'utf8'}));
assert.equal(module.Sum,pinned.sum);assert(module.Dir);
const result={unicode,regex:module.Dir};
fs.writeFileSync(path.join(destination,'INPUTS.json'),JSON.stringify(result,null,2)+'\n');
console.log(JSON.stringify(result));
