// Enumerate cases the actual upstream runners discover; verify all referenced
// expressions/datasets and require every standalone expression to be registered.
import fs from 'node:fs';
import path from 'node:path';
import {createHash} from 'node:crypto';
const root=path.resolve(path.dirname(new URL(import.meta.url).pathname),'..');
const reports=[];
function checkImports(dir){for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
 const file=path.join(dir,entry.name);if(entry.isDirectory()){checkImports(file);continue;}
 if(entry.name.endsWith('.go')&&/"github\.com\/(?:recolabs\/gnata|openbindings\/openbindings-(?:go|ts))(?:\/|\")/.test(fs.readFileSync(file,'utf8')))throw Error('Forbidden runtime/backend import '+file);
}}
checkImports(path.join(root,'go'));
for(const file of fs.readdirSync(path.join(root,'contract/cases')).filter(n=>n.startsWith('official-'))){
 const canonical=fs.readFileSync(path.join(root,'contract/cases',file));
 for(const target of ['go/internal/engine/testdata','javascript/test'])if(!canonical.equals(fs.readFileSync(path.join(root,target,file))))throw Error('Shared corpus mirror drift: '+target+'/'+file);
}
for(const target of ['go/IMPLEMENTATION.md','javascript/IMPLEMENTATION.md'])if(!fs.readFileSync(path.join(root,'contract/IMPLEMENTATION.md')).equals(fs.readFileSync(path.join(root,target))))throw Error('Packaged contract drift: '+target);
for(const [language,base] of [['go','go/internal/engine/testdata'],['javascript','javascript/test/test-suite']]) {
 const directory=path.join(root,base),groups=path.join(directory,'groups'),entries=[],references=new Set(),expressions=[];
 function visit(dir){for(const file of fs.readdirSync(dir,{withFileTypes:true})){
  const name=path.join(dir,file.name);if(file.isDirectory()){visit(name);continue;}
  if(file.name.endsWith('.jsonata'))expressions.push(name);
  if(!file.name.endsWith('.json'))continue;
  if(path.relative(groups,name).split(path.sep).length!==2)throw Error('Fixture outside native runner discovery depth '+name);
  const value=JSON.parse(fs.readFileSync(name));
  for(const [index,c] of (Array.isArray(value)?value:[value]).entries()){
   if(typeof c.expr!=='string'&&typeof c['expr-file']!=='string')throw Error('Missing expression '+name+'/'+index);
   const dependencies=[];
   if(c['expr-file'])dependencies.push(path.join(path.dirname(name),c['expr-file']));
   if(c.dataset)dependencies.push(path.join(directory,'datasets',c.dataset+'.json'));
   for(const dep of dependencies){if(!fs.existsSync(dep))throw Error('Missing fixture dependency '+dep);references.add(dep);}
   entries.push({file:path.relative(directory,name),index,dependencies:dependencies.map(p=>({file:path.relative(directory,p),sha256:createHash('sha256').update(fs.readFileSync(p)).digest('hex')}))});
  }
 }}
 visit(groups);
 for(const file of expressions)if(!references.has(file))throw Error('Unregistered expression '+file);
 reports.push({language,cases:entries.length,expressionFiles:expressions.length,inventorySHA256:createHash('sha256').update(JSON.stringify(entries)).digest('hex')});
}
console.log(JSON.stringify({status:'PASS',reports},null,2));
