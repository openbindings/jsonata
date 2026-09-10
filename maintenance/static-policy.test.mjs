import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {test} from 'node:test';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
test('reviewed policy retains raw failures and rejects changed findings or scanner failures',()=>{
 const temporary=fs.mkdtempSync(path.join(os.tmpdir(),'jsonata-static-policy-'));
 const executable=path.join(temporary,'linter.cjs');
 fs.writeFileSync(executable,`#!/usr/bin/env node
const fs=require('fs');
if(process.argv[2]==='version'){console.log('golangci-lint version 2.12.2');process.exit(0);}
if(process.env.JSONATA_TEST_SCANNER==='failure')process.exit(2);
const root=${JSON.stringify(root)};
const kind=process.argv.includes('../maintenance/gosec.yaml')?'security':'lint';
const ledger=JSON.parse(fs.readFileSync(root+'/maintenance/STATIC_FINDINGS.json'));
const issues=ledger.findings.filter(f=>f.kind===kind).map(f=>({FromLinter:f.rule,Pos:{Filename:root+'/'+f.file,Line:f.line,Column:f.column},Text:f.text,SourceLines:f.source}));
if(process.env.JSONATA_TEST_SCANNER==='changed')issues.push(issues[0]);
const target=process.argv.find(v=>v.startsWith('--output.json.path=')).split('=').slice(1).join('=');
fs.writeFileSync(target,JSON.stringify({Issues:issues}));
process.exit(1);
`);
 fs.chmodSync(executable,0o755);
 for(const [lane,mode,expected]of [['reviewed-static','normal',0],['static','normal',1],['reviewed-static','changed',1],['reviewed-static','failure',1]]){
  const output=path.join(temporary,lane+'-'+mode);
  const result=spawnSync(process.execPath,['maintenance/run.mjs',lane],{cwd:root,env:{...process.env,GOLANGCI_LINT:executable,JSONATA_MAINTENANCE_OUTPUT:output,JSONATA_TEST_SCANNER:mode},encoding:'utf8',timeout:30000});
  assert.equal(result.status,expected,result.stdout+result.stderr);
  const report=JSON.parse(fs.readFileSync(path.join(output,'RESULT.json')));
  if(mode==='normal'){
   assert.equal(report.rawFindings,true);
   for(const kind of ['lint','security']){
    assert.equal(report.records.find(r=>r.name==='raw-'+kind).code,1);
    assert.equal(report.records.find(r=>r.name==='reviewed-'+kind).code,0);
   }
  }
  assert.equal(report.status,expected===0?'PASS_REVIEWED_FINDINGS':'FAIL_OR_RAW_FINDINGS');
 }
});
