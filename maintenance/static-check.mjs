import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
export const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
export function normalize(issue,kind){
  const file=path.isAbsolute(issue.Pos.Filename)?path.relative(root,issue.Pos.Filename):'go/internal/engine/'+issue.Pos.Filename;
  assert(!file.startsWith('../')&&!path.isAbsolute(file));
  return {kind,rule:issue.FromLinter,file,line:issue.Pos.Line,column:issue.Pos.Column,text:issue.Text,source:issue.SourceLines};
}
export function verify(issues,kind,ledger,read=filename=>fs.readFileSync(path.join(root,filename))){
  const actual=issues.map(issue=>normalize(issue,kind));
  const expected=ledger.findings.filter(finding=>finding.kind===kind);
  const key=finding=>JSON.stringify(Object.fromEntries(['kind','rule','file','line','column','text','source'].map(k=>[k,finding[k]])));
  assert.deepEqual(actual.map(key).sort(),expected.map(key).sort(),'New, removed, changed or increased findings require review');
  for(const finding of expected){
    assert(finding.reason&&finding.evidence,'Missing specific review');
    assert.equal(createHash('sha256').update(read(finding.file)).digest('hex'),finding.fileSHA256,'Changed source invalidates review: '+finding.file);
  }
  return {status:'PASS_REVIEWED_FINDINGS',kind,count:actual.length,rawLint:'NOT_CLEAN',policy:'Evidence comparison only; not a required-CI waiver'};
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  const [kind,report]=process.argv.slice(2);
  assert(['lint','security'].includes(kind));
  const ledger=JSON.parse(fs.readFileSync(path.join(root,'maintenance/STATIC_FINDINGS.json')));
  const issues=JSON.parse(fs.readFileSync(report)).Issues;
  assert(Array.isArray(issues),'Missing raw report');
  console.log(JSON.stringify(verify(issues,kind,ledger)));
}
