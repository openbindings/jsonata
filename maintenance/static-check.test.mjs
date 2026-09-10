import fs from 'node:fs';
import assert from 'node:assert/strict';
import {test} from 'node:test';
import {root,verify} from './static-check.mjs';
test('Reviewed findings cannot hide new, increased, missing or changed findings or sources',()=>{
  const ledger=JSON.parse(fs.readFileSync(root+'/maintenance/STATIC_FINDINGS.json'));
  const issues=ledger.findings.filter(f=>f.kind==='security').map(f=>({FromLinter:f.rule,Pos:{Filename:root+'/'+f.file,Line:f.line,Column:f.column},Text:f.text,SourceLines:f.source}));
  assert.equal(verify(issues,'security',ledger).status,'PASS_REVIEWED_FINDINGS');
  assert.throws(()=>verify([...issues,issues[0]],'security',ledger));
  assert.throws(()=>verify(issues.slice(1),'security',ledger));
  const changed=structuredClone(issues);changed[0].Text+=' changed';
  assert.throws(()=>verify(changed,'security',ledger));
  assert.throws(()=>verify(issues,'security',ledger,()=>Buffer.from('changed source')));
});
