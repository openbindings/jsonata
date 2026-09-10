'use strict';
// Mechanically derive default full casing from Unicode's data, not from either
// evaluator's answers. Contextual Final_Sigma remains in the small runtime rule.
// Usage: node scripts/generate-casing.cjs INPUT_DIRECTORY [GO_REPOSITORY]
const fs=require('fs'),path=require('path'),{createHash}=require('crypto'),{execFileSync}=require('child_process');
const root=path.resolve(__dirname,'..'),inputs=process.argv[2],goRoot=process.argv[3];
if(!inputs)throw Error('Supply a cache directory for pinned Unicode inputs');
const sources={
    'UnicodeData.txt':['https://www.unicode.org/Public/16.0.0/ucd/UnicodeData.txt','ff58e5823bd095166564a006e47d111130813dcf8bf234ef79fa51a870edb48f'],
    'SpecialCasing.txt':['https://www.unicode.org/Public/16.0.0/ucd/SpecialCasing.txt','8d5de354eef79f2395a54c9c7dcebbaf3d30fc962d0f85611ea97aa973a0c451'],
    'DerivedCoreProperties.txt':['https://www.unicode.org/Public/16.0.0/ucd/DerivedCoreProperties.txt','39d35161f2954497f69e08bdb9e701493f476a3d30222de20028feda36c1dabd'],
    'LICENSE':['https://www.unicode.org/license.txt','e7a93b009565cfce55919a381437ac4db883e9da2126fa28b91d12732bc53d96']
};
const hash=bytes=>createHash('sha256').update(bytes).digest('hex');
(async()=>{
    fs.mkdirSync(inputs,{recursive:true});const text={};
    for(const [name,[url,sha256]]of Object.entries(sources)){
        const target=path.join(inputs,name);let bytes;
        if(fs.existsSync(target))bytes=fs.readFileSync(target);
        else{const response=await fetch(url);if(!response.ok)throw Error(response.status+' '+url);bytes=Buffer.from(await response.arrayBuffer());}
        if(hash(bytes)!==sha256)throw Error('Pinned Unicode source changed: '+name);
        if(!fs.existsSync(target))fs.writeFileSync(target,bytes);text[name]=bytes.toString('utf8');
    }
    const upper={},lower={},cased=[],ignorable=[],decimalZeros=[];const conditional=[];
    const decode=field=>field.trim().split(/\s+/).map(s=>parseInt(s,16));
    for(const line of text['UnicodeData.txt'].split('\n')){
        if(!line)continue;const fields=line.split(';'),cp=parseInt(fields[0],16);
        if(fields[12])upper[cp]=String.fromCodePoint(parseInt(fields[12],16));
        if(fields[13])lower[cp]=String.fromCodePoint(parseInt(fields[13],16));
        if(fields[2]==='Nd' && fields[6]==='0')decimalZeros.push(cp);
    }
    for(const line of text['SpecialCasing.txt'].split('\n')){
        const body=line.split('#')[0].trim();if(!body)continue;
        const fields=body.split(';').map(s=>s.trim()),cp=parseInt(fields[0],16),condition=fields[4];
        if(condition){if(!/^(tr|az|lt)(\s|$)/i.test(condition)){if(condition!=='Final_Sigma'||cp!==0x3a3)throw Error('Unimplemented default casing context: '+body);conditional.push({cp,condition,lower:decode(fields[1])});}continue;}
        lower[cp]=String.fromCodePoint(...decode(fields[1]));upper[cp]=String.fromCodePoint(...decode(fields[3]));
    }
    if(conditional.length!==1||conditional[0].lower.join(',')!=='962')throw Error('Default contextual rule changed');
    for(const line of text['DerivedCoreProperties.txt'].split('\n')){
        const body=line.split('#')[0].trim();if(!body)continue;const [range,property]=body.split(';').map(s=>s.trim());
        const target=property==='Cased'?cased:property==='Case_Ignorable'?ignorable:undefined;if(!target)continue;
        const bounds=range.split('..').map(s=>parseInt(s,16));target.push([bounds[0],bounds[1]===undefined?bounds[0]:bounds[1]]);
    }
    const data={version:'16.0.0',upper,lower,cased,ignorable,decimalZeros};
    const outputs={};const write=(file,bytes)=>{fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,bytes);outputs[path.basename(file)]=hash(bytes)};
    write(path.join(root,'src/unicode-data.json'),JSON.stringify(data)+'\n');
    const metadata=JSON.stringify({unicode:'16.0.0',sources,algorithm:'Unicode 16.0 section 3.13 default full upper/lower; locale-independent Final_Sigma, no normalization'},null,2)+'\n';
    write(path.join(root,'unicode/LICENSE'),text.LICENSE);write(path.join(root,'unicode/SOURCE.json'),metadata);
    if(goRoot){
        const dir=path.join(goRoot,'internal/unicodecase');
        const mappings=(name,table)=>'var '+name+' = map[rune]string{\n'+Object.entries(table).map(([cp,value])=>'0x'+Number(cp).toString(16)+':'+JSON.stringify(value)+',').join('\n')+'\n}\n';
        const ranges=(name,values)=>'var '+name+' = [][2]rune{\n'+values.map(([a,b])=>'{0x'+a.toString(16)+',0x'+b.toString(16)+'},').join('\n')+'\n}\n';
        const source='// Code generated from pinned Unicode 16.0.0 data; DO NOT EDIT.\n// Copyright Unicode, Inc. See LICENSE and SOURCE.json.\npackage unicodecase\nconst Version = "16.0.0"\n'+mappings('upperMapping',upper)+mappings('lowerMapping',lower)+ranges('casedRanges',cased)+ranges('ignorableRanges',ignorable)+'var decimalZeros = []rune{\n'+decimalZeros.map(cp=>'0x'+cp.toString(16)+',').join('\n')+'\n}\n';
        const formatted=execFileSync('gofmt',[],{input:source,encoding:'utf8'});
        write(path.join(dir,'data.go'),formatted);write(path.join(dir,'LICENSE'),text.LICENSE);write(path.join(dir,'SOURCE.json'),metadata);
    }
    console.log(JSON.stringify({sourceVersion:data.version,mappings:{upper:Object.keys(upper).length,lower:Object.keys(lower).length},ranges:{cased:cased.length,ignorable:ignorable.length},outputs}));
})().catch(error=>{console.error(error);process.exitCode=1});
