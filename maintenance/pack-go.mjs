// Create a local Go module-proxy artifact, without publishing or modifying Go's
// shared module cache. Use a fresh output directory and isolated consumer cache.
import fs from 'node:fs';
import path from 'node:path';
import {execFileSync} from 'node:child_process';
import {createHash} from 'node:crypto';
const root=path.resolve(path.dirname(new URL(import.meta.url).pathname),'..');
const output=path.resolve(process.argv[2]);
if(fs.existsSync(output))throw Error('Fresh artifact destination required');
const module='github.com/openbindings/jsonata-runtime/go',version='v0.0.0-dev';
const prefix=module+'@'+version,stage=path.join(output,'stage',prefix),proxy=path.join(output,'proxy',module,'@v');
fs.mkdirSync(stage,{recursive:true});fs.mkdirSync(proxy,{recursive:true});
const files=[];
function copy(dir,relative='') { for(const entry of fs.readdirSync(dir,{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name))){
 const name=path.join(relative,entry.name),from=path.join(dir,entry.name),to=path.join(stage,name);
 if(entry.name==='.git'||entry.name==='node_modules'||entry.name==='.DS_Store')continue;
 if(entry.isDirectory()){fs.mkdirSync(to,{recursive:true});copy(from,name);}
 else if(entry.isFile()){fs.copyFileSync(from,to);fs.chmodSync(to,0o644);fs.utimesSync(to,new Date('2026-09-10T00:00:00Z'),new Date('2026-09-10T00:00:00Z'));files.push(name);}
 else throw Error('Unsupported artifact member '+from);
}}
copy(path.join(root,'go'));
const archive=path.join(proxy,version+'.zip');
execFileSync('zip',['-X','-q',archive,'-@'],{cwd:path.join(output,'stage'),input:files.map(n=>prefix+'/'+n).join('\n')+'\n'});
fs.copyFileSync(path.join(root,'go/go.mod'),path.join(proxy,version+'.mod'));
fs.writeFileSync(path.join(proxy,version+'.info'),JSON.stringify({Version:version,Time:'2026-09-10T00:00:00Z'})+'\n');
fs.writeFileSync(path.join(proxy,'list'),version+'\n');
const record={module,version,proxy:path.join(output,'proxy'),archive,files:files.length,sha256:createHash('sha256').update(fs.readFileSync(archive)).digest('hex'),purpose:'Local private candidate artifact, not a release or source replacement'};
fs.writeFileSync(path.join(output,'ARTIFACT.json'),JSON.stringify(record,null,2)+'\n');console.log(JSON.stringify(record));
