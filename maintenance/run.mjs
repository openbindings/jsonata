// One small repository-owned entry point. Reports each gate independently;
// it never updates findings, publishes, or treats raw lint as a clean check.
import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const lane=process.argv[2];
const lanes=['go','javascript','shared','resource','packages','generated','dependency','static','reviewed-static'];
assert(lanes.includes(lane),'Usage: node maintenance/run.mjs '+lanes.join('|'));
const parent=path.join(root,'.maintenance-output');fs.mkdirSync(parent,{recursive:true});
const output=process.env.JSONATA_MAINTENANCE_OUTPUT?path.resolve(process.env.JSONATA_MAINTENANCE_OUTPUT):fs.mkdtempSync(path.join(parent,lane+'-'));
assert(!fs.existsSync(path.join(output,'RESULT.json')),'Use a fresh evidence destination');
fs.mkdirSync(output,{recursive:true});
const records=[];let failed=false,rawFindings=false;
function run(name,cwd,command,args,allowed=[0]){
  const start=Date.now();const result=spawnSync(command,args,{cwd:path.join(root,cwd),env:{...process.env,GOWORK:'off'},encoding:'utf8',timeout:300000,maxBuffer:32<<20});
  const record={name,command,args,cwd,code:result.status,signal:result.signal,error:result.error?.message,elapsedMs:Date.now()-start,stdout:result.stdout,stderr:result.stderr};
  fs.writeFileSync(path.join(output,name+'.json'),JSON.stringify(record,null,2)+'\n');records.push({name,code:result.status});
  console.log(JSON.stringify({name,code:result.status}));
  if(!allowed.includes(result.status)){failed=true;console.error((result.stdout+result.stderr).slice(-6000));throw Error('Gate failed: '+name);}
  return result;
}
try{
  if(lane==='go')run('native-go','go','go',['test','-race','./...']);
  if(lane==='javascript'){
    run('native-js','javascript','npm',['test']);
    run('docs','javascript','npm',['run','doc']);
  }
  if(lane==='shared'){
    run('boundary-vectors','.','node',['maintenance/boundary-cases.mjs','--check']);
    run('parser-patch','.','node',['--test','maintenance/parser-patch.test.cjs']);
    run('inventory','.','node',['maintenance/inventory.mjs']);
    run('vectors','.','node',['maintenance/qualify.mjs']);
    run('maintenance-guards','.','node',['--test','maintenance/static-check.test.mjs']);
    run('static-policy','.','node',['--test','maintenance/static-policy.test.mjs']);
  }
  if(lane==='resource'){
    run('go-resource','go','go',['test','-race','./...','-run','Cancel|Timeout|Budget|Isolation|Checkpoint|Resource|Lifecycle','-count=3']);
    run('js-resource','javascript','node',['node_modules/mocha/bin/mocha.js','test/evaluation-cancellation.js','test/json-executor.js','test/boundary-corrections.js','test/worker-faults.js','test/worker-protocol.js','test/public-entry.js']);
  }
  if(lane==='packages'){
    run('build-js','javascript','npm',['run','build']);
    run('npm-consumer','.','node',['maintenance/consumer-js.mjs']);
    const destination=path.join(output,'go-artifact');
    run('go-archive','.','node',['maintenance/pack-go.mjs',destination]);
    run('go-consumer','.','node',['maintenance/consumer-go.mjs',path.join(destination,'ARTIFACT.json')]);
  }
  if(lane==='generated'){
    let unicode=process.env.JSONATA_UNICODE_INPUT,regex=process.env.JSONATA_REGEX_INPUT;
    if(!unicode&&!regex){
      const destination=path.join(output,'inputs');
      run('generator-inputs','.','node',['maintenance/generated-inputs.mjs',destination]);
      ({unicode,regex}=JSON.parse(fs.readFileSync(path.join(destination,'INPUTS.json'))));
    }
    assert(unicode&&regex,'Supply both generator input paths, or neither to fetch checksum-verified inputs');
    run('generated','.','node',['maintenance/verify-generated.mjs',unicode,regex]);
  }
  if(lane==='dependency')run('exposure','.','node',['maintenance/audit-exposure.mjs',path.join(output,'audit')]);
  if(lane==='static'||lane==='reviewed-static'){
    const tool=process.env.GOLANGCI_LINT??'golangci-lint';
    const version=run('linter-version','go',tool,['version']);assert.match(version.stdout,/version 2\.12\.2\b/);
    for(const [kind,config] of [['lint','internal/engine/.golangci.yaml'],['security','../maintenance/gosec.yaml']]){
      const report=path.join(output,kind+'.json');
      const args=['run','--config',config,'--fix=false','--max-issues-per-linter=0','--max-same-issues=0','--output.json.path='+report,'--output.text.path=/dev/null','./...'];
      if(kind==='security')args.splice(-1,0,'--uniq-by-line=false','--path-mode=abs');
      const raw=run('raw-'+kind,'go',tool,args,[0,1]);
      run('reviewed-'+kind,'.','node',['maintenance/static-check.mjs',kind,report]);
      if(raw.status!==0){
        rawFindings=true;
        if(lane==='static')failed=true; // Raw mode stays nonzero; reviewed mode requires exact ledger checks.
      }
    }
  }
}catch(error){failed=true;console.error(error.message);}
const report={lane,status:failed?'FAIL_OR_RAW_FINDINGS':lane==='reviewed-static'?'PASS_REVIEWED_FINDINGS':'PASS',rawFindings,records,output,publication:false};
fs.writeFileSync(path.join(output,'RESULT.json'),JSON.stringify(report,null,2)+'\n');
console.log(JSON.stringify(report));process.exitCode=failed?1:0;
