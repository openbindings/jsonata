// Current graph exposure gate, deliberately separate from a clean npm audit.
import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import {spawnSync, execFileSync} from 'node:child_process';
const root=path.resolve(path.dirname(new URL(import.meta.url).pathname),'..');
const output=path.resolve(process.argv[2]);
fs.mkdirSync(output,{recursive:true});
const js=path.join(root,'javascript');
function audit(name,args){
  const result=spawnSync('npm',['audit','--json',...args],{cwd:js,encoding:'utf8',timeout:120000,maxBuffer:8<<20});
  fs.writeFileSync(path.join(output,name+'.json'),JSON.stringify(result,null,2)+'\n');
  assert([0,1].includes(result.status),name+': scanner execution failed');
  const parsed=JSON.parse(result.stdout);assert(!parsed.error,name+': registry audit unavailable');return parsed;
}
const production=audit('production',['--omit=dev']);
assert.equal(production.metadata.vulnerabilities.total,0);
const development=audit('development',[]);
const expected=['browserify-sign','create-ecdh','crypto-browserify','elliptic'];
assert.deepEqual(Object.keys(development.vulnerabilities).sort(),expected);
for(const [name,finding] of Object.entries(development.vulnerabilities)){
  assert.equal(finding.severity,'low',name);
  assert.deepEqual(finding.nodes,['node_modules/'+name]);
}
const direct=development.vulnerabilities.elliptic.via;
assert.equal(direct.length,1);
assert.equal(direct[0].url,'https://github.com/advisories/GHSA-848j-6mx2-7j84');
assert.equal(direct[0].range,'<=6.6.1');
const trace=path.join(output,'build-loaded-modules.json');
execFileSync(process.execPath,['--require',path.join(root,'maintenance/trace-build.cjs'),'scripts/build-runtime.cjs'],{
  cwd:js,env:{...process.env,JSONATA_BUILD_TRACE:trace},stdio:'inherit',timeout:120000
});
const loaded=JSON.parse(fs.readFileSync(trace));
const graphs={};
for(const entry of ['src/jsonata.js','src/executor-public.js']){
  const list=execFileSync(process.execPath,['node_modules/browserify/bin/cmd.js',entry,'--list'],{cwd:js,encoding:'utf8'}).trim().split('\n');
  graphs[entry]=list.map(file=>path.relative(js,file));
}
for(const [surface,files] of Object.entries({build:loaded,...graphs})){
  for(const name of expected)assert(!files.some(file=>file.includes('node_modules/'+name+'/')),surface+' loads '+name);
}
const result={status:'PASS_CURRENT_REACHABILITY',rawDevelopmentAudit:'RED',productionAudit:'CLEAN',advisory:direct[0].url,
  affectedDevelopmentPackages:expected,buildLoadsAffectedCode:false,bundlesContainAffectedCode:false,graphs,
  limitation:'Current maintained runtime/build graph only; not blanket acceptance of development dependencies or future cryptographic use.'};
fs.writeFileSync(path.join(output,'EXPOSURE.json'),JSON.stringify(result,null,2)+'\n');
console.log(JSON.stringify({...result,graphs:undefined}));
