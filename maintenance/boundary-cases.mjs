// Deterministic, type-aware boundary witnesses. JSON text with duplicate names
// is constructed as text and is never normalized by an input parser.
import fs from 'node:fs';
import assert from 'node:assert/strict';
const cases=[];
const add=(id,expr,inputJSON,expected,bindingsJSON)=>cases.push({id,expr,inputJSON,expected,...(bindingsJSON===undefined?{}:{bindingsJSON})});
const values=[null,false,true,0,1,'',[],[42],{}, {length:0},{length:1,0:42},[[]],{a:1}];
function same(a,b){if(a===b)return true;if(!a||!b||typeof a!=='object'||typeof b!=='object'||Array.isArray(a)!==Array.isArray(b))return false;const keys=Object.keys(a);return keys.length===Object.keys(b).length&&keys.every(k=>Object.hasOwn(b,k)&&same(a[k],b[k]));}
for(let i=0;i<values.length;i++)for(let j=0;j<values.length;j++)for(const nested of [false,true]){
 const a=nested?[values[i]]:values[i],b=nested?[values[j]]:values[j];
 add(`types-${i}-${j}-${nested}`,'[a = b, a != b]',JSON.stringify({a,b}),{status:'json',json:JSON.stringify([same(a,b),!same(a,b)])});
}
const bad=[
 '{"id":7','{"id":7} garbage','{"id":7} {"id":8}','garbage {"id":7}','{"id":7,}',
 '{"id":7,"x":NaN}','{"id":7,"x":Infinity}','{"id":7,"x":01}','{"id":7,"x":1.}','{"id":7,"x":1e+}',
 '{"id":7,"x":[1,]}','{"id":7,"x":"\\q"}','{"id":7,"x":"\\u-001"}',
 '{"id":7,"id":7}','{"id":7,"id":8}','{"id":7,"\\u0069d":7}',
 '{"id":7,"x":{"n":1,"n":1.0}}','{"id":7,"x":{"n":{},"n":[]}}',
 '{"id":7,"x":{"__proto__":null,"__proto__":null}}',
 '{"id":7,"x":[{"n":1,"n":2}]}','{"id":7,"x":{"\\ud800":1,"\\ud800":2}}'
];
for(let i=0;i<bad.length;i++)for(const [j,expr] of ['id','id = 7','$exists(id)','{"value":id}','42'].entries())add(`invalid-${i}-${j}`,expr,bad[i],{status:'failure'});
for(const [i,bindings] of ['{"x":1,"x":1}','{"x":1,"x":2}','{"x":{"n":1,"n":2}}','{"$":3}','{"$x":3}','{"\\u0024":3}'].entries())add('bindings-'+i,'$$','{"real":1}',{status:'failure'},bindings);
for(const [i,raw] of ['null','false','true','0','1.0','9007199254740993','1e400','1e-400','"\\ud800"','[]','{}','{"$":7}','{"__proto__":{"x":7}}','[{"a":1},{"a":2}]','{"a":1,"A":2}','{"é":1,"é":2}'].entries())add('valid-'+i,'$',raw,{status:'json',json:raw});
add('distinct-containers','$distinct(items)','{"items":[{"length":0},[]]}',{status:'json',json:'[{"length":0},[]]'});
add('numeric-leaves','[a = b, a = c]','{"a":9007199254740993,"b":9007199254740993.0,"c":9007199254740992}',{status:'json',json:'[true,false]'});
add('empty-binding-root','$$','{"real":1}',{status:'json',json:'{"real":1}'},'{"":42}');
add('nested-dollar','$x','null',{status:'json',json:'{"$":7}'},'{"x":{"$":7}}');
add('binding-shadow','($sum := function($v){$v};$sum($x))','null',{status:'json',json:'7'},'{"x":7}');
const output=JSON.stringify({description:'Runtime admission corrections; not general Core or JSONata conformance requirements',cases},null,2)+'\n';
const target=new URL('../contract/cases/boundary-admission-cases.json',import.meta.url);
if(process.argv.includes('--check'))assert.equal(fs.readFileSync(target,'utf8'),output);else fs.writeFileSync(target,output);
console.log(JSON.stringify({status:'PASS',cases:cases.length,generated:true}));
