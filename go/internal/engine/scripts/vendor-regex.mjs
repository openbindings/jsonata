// Mechanical private relocation; no runtime dependency on a workstation path.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const source=process.argv[2];
if(!source || !source.endsWith('/github.com/auvred/regonaut@v0.0.1'))throw Error('Pinned regonaut v0.0.1 module directory required');
const target=new URL('../internal/thirdparty/regonaut/',import.meta.url);
const hashes={}; const sha=b=>crypto.createHash('sha256').update(b).digest('hex');
const pinned={
 'regonaut.go':'d2b14f0536fee6cbc3e003c657166309fde55f3b4b89f57fdc2c8aca2c1076c5',
 'unicode_generated.go':'c83a29a57bceb0fd64d0b818e8191e67c0f7352a480958c1aca61f71e28f8529',
 'LICENSE':'ffaeebcd807e4743e606a8d9014b11b08be925a619770e20682c8757d5144bca',
 'README.md':'51b2aabd4c739063d6065832b12e7c074e21957a6c620ba140dd35ca189696df'
};
function replaceOnce(s,a,b){if(s.split(a).length!==2)throw Error('Upstream patch anchor changed: '+a);return s.replace(a,b)}
fs.mkdirSync(target,{recursive:true});
for(const name of ['regonaut.go','unicode_generated.go','LICENSE','README.md']){
 const bytes=fs.readFileSync(path.join(source,name));let output=bytes.toString();
 if(sha(bytes)!==pinned[name])throw Error('Pinned upstream source hash changed: '+name);
 if(name==='regonaut.go'){
  output=replaceOnce(output,'type compiler struct {','type compiler struct {\n\tboundedDepth int');
  output=replaceOnce(output,'func (c *compiler) emit(v func(vm *machine)) int {','func (c *compiler) emit(v func(vm *machine)) int {\n\tc.checkCompileBudget()');
  output=replaceOnce(output,'func (c *compiler) compileDisjunction() error {','func (c *compiler) compileDisjunction() error {\n\tc.boundedDepth++; defer func(){ c.boundedDepth-- }()\n\tc.checkCompileBudget()');
  output=replaceOnce(output,'func (c *compiler) compileTerm() error {','func (c *compiler) compileTerm() error {\n\tdefer c.checkCompileBudget()');
  output=replaceOnce(output,'type machine struct {','type machine struct {\n\tbudget *executionBudget');
  output=replaceOnce(output,'func (vm *machine) pushBacktrackingFrame(pc int) {','func (vm *machine) pushBacktrackingFrame(pc int) {\n\tvm.checkFrameBudget()');
  output=replaceOnce(output,'\t\tvm.byteCode[vm.pc](vm)','\t\tvm.checkExecutionBudget()\n\t\tvm.byteCode[vm.pc](vm)');
  const formatted=spawnSync('gofmt',[],{input:output,encoding:'utf8'});
  if(formatted.status!==0)throw Error(formatted.stderr);
  output=formatted.stdout;
 }
 hashes[name]={source:sha(bytes),relocated:sha(output)};
 const destination=new URL(name,target);
 if(fs.existsSync(destination)&&fs.readFileSync(destination,'utf8')!==output)throw Error('Refusing to overwrite edited vendor file: '+name);
 fs.writeFileSync(destination,output);
}
const record={module:'github.com/auvred/regonaut',version:'v0.0.1',sum:'h1:308+62qAZlIJ9Uq8R9KeLscLFBu/gENl1ZlgdlwWM1E=',license:'MIT',changes:'Seven instrumentation anchors only; bounded.go supplies private resource controls. Regex grammar and matching rules unchanged.',hashes};
const manifest=new URL('UPSTREAM.json',target);const rendered=JSON.stringify(record,null,2)+'\n';
if(fs.existsSync(manifest)&&fs.readFileSync(manifest,'utf8')!==rendered)throw Error('Manifest changed');
fs.writeFileSync(manifest,rendered);console.log(JSON.stringify(record));
