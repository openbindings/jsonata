// Real module artifact consumer: no source replacements or sibling imports.
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {execFileSync,spawnSync} from 'node:child_process';
const artifact=JSON.parse(fs.readFileSync(path.resolve(process.argv[2])));
const temporary=fs.mkdtempSync(path.join(os.tmpdir(),'jsonata-independent-go-'));
const prerequisites=process.env.JSONATA_PUBLIC_MODULE_PROXY??'https://proxy.golang.org';
const env={...process.env,GOWORK:'off',GOTOOLCHAIN:'local',GOMODCACHE:path.join(temporary,'module-cache'),GOSUMDB:'off',GOFLAGS:'-buildvcs=false',GOPROXY:'file://'+artifact.proxy+','+prerequisites};
const logs=[];
const run=(args,cwd=temporary)=>{const output=execFileSync('go',args,{cwd,env,encoding:'utf8',timeout:120000,maxBuffer:16<<20});logs.push({args,output});return output;};
fs.writeFileSync(path.join(temporary,'go.mod'),'module consumer.example/jsonata\n\ngo 1.25.6\n\nrequire '+artifact.module+' '+artifact.version+'\n');
const expression = JSON.stringify('id = 9007199254740993 ? {"id":id,"sum":0.1+0.2} : null');
const input = JSON.stringify('{"id":9007199254740993}');
fs.writeFileSync(path.join(temporary,'main.go'),`package main
import("context";"errors";"fmt";"strings"; jsonata "github.com/openbindings/jsonata/go")
func main(){
 e,err:=jsonata.New(jsonata.Options{});if err!=nil{panic(err)}
 output,err:=e.Evaluate(context.Background(),${expression},[]byte(${input}),nil)
 if err!=nil||!strings.Contains(string(output),"9007199254740993")||!strings.Contains(string(output),"0.3"){panic(fmt.Sprintf("%s %v",output,err))}
 if _,err=e.Evaluate(context.Background(),"missing",[]byte("null"),nil);!errors.Is(err,jsonata.ErrUndefined){panic("undefined")}
 if _,err=e.Evaluate(context.Background(),"function(){1}",[]byte("null"),nil);err==nil{panic("function result")}
 ctx,cancel:=context.WithCancel(context.Background());cancel();if _,err=e.Evaluate(ctx,"$",[]byte("null"),nil);!errors.Is(err,context.Canceled){panic("cancellation")}
 fmt.Println(string(output))
}
`);
run(['mod','tidy']);run(['run','.']);
assert(!fs.readFileSync(path.join(temporary,'go.mod'),'utf8').includes('replace'));
const modules=run(['list','-m','-json','all']);
assert(!modules.includes('openbindings-go'));assert(modules.includes(temporary+'/module-cache/github.com/openbindings/jsonata/go@'));
fs.mkdirSync(path.join(temporary,'syntax'));
fs.writeFileSync(path.join(temporary,'syntax/main.go'),'package main\nimport "github.com/openbindings/jsonata/go/syntax"\nfunc main(){if syntax.Validate("1+1")!=nil{panic("syntax")}}\n');
const syntaxDeps=run(['list','-deps','./syntax']).trim().split('\n');
assert(!syntaxDeps.some(n=>n===artifact.module+'/internal/engine'||n.includes('/engine/internal/evaluator')||n.includes('/engine/functions')));
run(['run','./syntax']);
fs.mkdirSync(path.join(temporary,'private'));
fs.writeFileSync(path.join(temporary,'private/main.go'),'package main\nimport _ "github.com/openbindings/jsonata/go/internal/engine"\nfunc main(){}\n');
const denied=spawnSync('go',['build','./private'],{cwd:temporary,env,encoding:'utf8'});assert.notEqual(denied.status,0);assert.match(denied.stderr,/use of internal package/);
const record={status:'PASS',temporary,artifact,workspace:'off',sourceReplacements:false,sdkDependencies:false,syntaxLoadsEvaluator:false,privateBackendImportDenied:true,logs};
fs.writeFileSync(path.join(temporary,'RESULTS.json'),JSON.stringify(record,null,2)+'\n');console.log(JSON.stringify({...record,logs:undefined}));
