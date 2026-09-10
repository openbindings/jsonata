// Shared vectors through supported public execution surfaces. The independent
// oracle uses source-context JSON parsing and BigInt, never either engine's math.
import fs from 'node:fs';
import path from 'node:path';
import {execFileSync} from 'node:child_process';
import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
const root=path.resolve(path.dirname(new URL(import.meta.url).pathname),'..');
const marker=Symbol('number');
function decimal(text){const m=/^(-?)(0|[1-9][0-9]*)(?:\.([0-9]+))?(?:[eE]([+-]?[0-9]+))?$/.exec(text);assert(m);let c=BigInt(m[1]+m[2]+(m[3]||'')),e=BigInt(m[4]||'0')-BigInt((m[3]||'').length);if(c===0n)return '0';while(c%10n===0n){c/=10n;e++;}return c+'e'+e;}
function parse(text){return JSON.parse(text,(_key,value,context)=>{if(typeof value!=='number')return value;assert.equal(typeof context?.source,'string');return {[marker]:decimal(context.source)};});}
function equal(a,b){if(a===b)return true;if(!a||!b||typeof a!=='object'||typeof b!=='object')return false;if(marker in a||marker in b)return a[marker]===b[marker];if(Array.isArray(a)!==Array.isArray(b))return false;const keys=Object.keys(a);return keys.length===Object.keys(b).length&&keys.every(k=>Object.hasOwn(b,k)&&equal(a[k],b[k]));}
assert(equal(parse('1.0'),parse('1e0')));assert(!equal(parse('9007199254740992'),parse('9007199254740993')));assert(!equal(parse('"\\ud800"'),parse('"�"')));assert(!equal(parse('1e-400'),parse('0')));
const directory=path.join(root,'contract/cases'),cases=[];
for(const file of fs.readdirSync(directory).sort()){
 const corpus=JSON.parse(fs.readFileSync(path.join(directory,file)));
 for(const c of corpus.cases){const expected=c.expected??((c.error||c.errorCode)?{status:'failure',code:c.error||c.errorCode}:{status:'json',json:c.resultJSON});cases.push({...c,id:file+'/'+c.id,expected});}
}
assert(cases.length>1200);assert.equal(new Set(cases.map(c=>c.id)).size,cases.length);
const overlays=JSON.parse(fs.readFileSync(path.join(root,'contract/public-boundary-overlays.json'))).cases;
for(const overlay of overlays){const c=cases.find(c=>c.id===overlay.id);assert(c);for(const [key,value] of Object.entries(overlay.source))assert.equal(c[key],value,'changed overlay source '+c.id);c.expected=overlay.expected;}
const output=execFileSync('go',['run','./internal/qualification'],{cwd:path.join(root,'go'),env:{...process.env,GOWORK:'off'},input:cases.map(c=>JSON.stringify(c)).join('\n')+'\n',encoding:'utf8',timeout:120000,maxBuffer:32<<20});
const go=output.trim().split('\n').map(x=>JSON.parse(x));assert.equal(go.length,cases.length);
const require=createRequire(path.join(root,'javascript/package.json'));
const {createJSONExecutor}=require('@openbindings/jsonata');
const {createNodeExecutor}=require('@openbindings/jsonata/node');
const direct=createJSONExecutor({timeout:5000}),worker=createNodeExecutor({timeout:5000,workers:1});
const failures=[];let observations=0;
try{for(let i=0;i<cases.length;i++){
 const c=cases[i];assert.equal(go[i].id,c.id);
 const results=[['go',go[i]]];
 for(const [name,executor] of [['javascript',direct],['node-worker',worker]]){
  try{results.push([name,{status:'json',json:await executor.evaluate(c.expr,c.inputJSON,{bindingsJSON:c.bindingsJSON})}]);}
  catch(error){results.push([name,{status:'failure',error:String(error?.code??'')+' '+String(error?.message??error)}]);}
 }
 for(const [language,actual] of results){observations++;const expected=c.expected;
  // "syntax" in the native regex corpus is a compile-rejection category,
  // not a literal diagnostic code. Native suites prove its compile phase;
  // this combined evaluate boundary requires rejection without prescribing text.
  const ok=actual.status===expected.status&&(expected.status==='json'?equal(parse(actual.json),parse(expected.json)):!expected.code||expected.code==='syntax'||actual.error.includes(expected.code));
  if(!ok)failures.push({id:c.id,language,expected,actual});
 }
}}finally{await worker.close();}
console.log(JSON.stringify({status:failures.length?'FAIL':'PASS',cases:cases.length,observations,publicBoundaryOverlays:overlays.length,scope:'public boundaries and stronger implementation vectors, not general third-party conformance',failures},null,2));
if(failures.length)process.exitCode=1;
