'use strict';
// Rebuild complete browser graphs, including dependencies. Transpiling only
// our src/ leaves modern dependency syntax in the advertised older bundle.
const fs = require('fs');
const path = require('path');
const browserify = require('browserify');
const babel = require('@babel/core');
const terser = require('terser');
const acorn = require('acorn');
const {execFileSync} = require('child_process');
const root = path.resolve(__dirname,'..');
const digest = bytes => require('crypto').createHash('sha256').update(bytes).digest('hex');
function bundle(entry, standalone) {
    return new Promise((resolve,reject) => browserify(path.join(root,entry),{standalone}).bundle((error,value) => error ? reject(error) : resolve(value.toString())));
}
(async () => {
    const hashes = {};
    function write(name,source) { fs.writeFileSync(path.join(root,name),source); hashes[name] = digest(source); }
    for (const [entry,name,global] of [['src/jsonata.js','jsonata','jsonata'], ['src/executor-public.js','json-executor','jsonataExecutor']]) {
        const source = await bundle(entry,global);
        const modern = babel.transformSync(source,{babelrc:false,configFile:false,sourceType:'script',presets:[['@babel/preset-env',{targets:{node:'8'},modules:false}]],comments:true}).code+'\n';
        write(name+'.js',modern);
        const min = await terser.minify(modern,{ecma:2017});
        write(name+'.min.js',min.code+'\n');
        const older = babel.transformSync(source,{babelrc:false,configFile:false,sourceType:'script',presets:[['@babel/preset-env',{targets:{ie:'11'},modules:false}]],comments:true}).code+'\n';
        // Syntax evidence, not proof that an arbitrary ES5 host supplies every
        // required built-in (notably Promise). Keep those claims separate.
        acorn.parse(older,{ecmaVersion:5});
        write(name+'-es5.js',older);
        const minOlder = await terser.minify(older,{ecma:5});
        acorn.parse(minOlder.code,{ecmaVersion:5});
        write(name+'-es5.min.js',minOlder.code+'\n');
    }
    // These optional Node-only entries are deliberately not browserified. Their
    // relative dependencies all exist inside the package, never a workspace.
    for (const name of ['node-executor.js','json-worker.js','execution-options.js']) write(name,fs.readFileSync(path.join(root,'src',name)));
    const listed = execFileSync(process.execPath,[require.resolve('browserify/bin/cmd.js'),'src/json-executor.js','--list'],{cwd:root,encoding:'utf8'}).trim().split('\n');
    const packages = new Map();
    for (const file of listed) {
        if (!file.includes(path.sep+'node_modules'+path.sep)) continue;
        let dir = path.dirname(file);
        while (!fs.existsSync(path.join(dir,'package.json')) || !JSON.parse(fs.readFileSync(path.join(dir,'package.json'))).name) { const parent = path.dirname(dir); if (parent === dir) throw Error('No package owner: '+file); dir = parent; }
        const pkg = JSON.parse(fs.readFileSync(path.join(dir,'package.json')));
        const key = pkg.name+'@'+pkg.version;
        if (packages.has(key)) continue;
        const license = fs.readdirSync(dir).find(name => /^(license|licence|copying)(\.[^.]+)?$/i.test(name));
        if (!license) throw Error('Missing bundled dependency notice: '+key);
        packages.set(key,fs.readFileSync(path.join(dir,license),'utf8'));
    }
    const unicodeNotice = '\n## Unicode 16.0.0 data\n\n'+fs.readFileSync(path.join(root,'unicode/LICENSE'),'utf8')+'\nSource fingerprints:\n\n'+fs.readFileSync(path.join(root,'unicode/SOURCE.json'),'utf8');
    write('THIRD_PARTY_NOTICES.md','# Bundled third-party notices\n\nGenerated from the resolved browser dependency graph.\n\n'+Array.from(packages).sort(([a],[b]) => a.localeCompare(b)).map(([name,license]) => '## '+name+'\n\n'+license+'\n').join('\n')+unicodeNotice);
    fs.writeFileSync(path.join(root,'BUILD.json'),JSON.stringify({purpose:'Private candidate build; no publication',node:process.version,artifacts:hashes},null,2)+'\n');
    console.log(JSON.stringify(hashes,null,2));
})().catch(error => { console.error(error); process.exitCode = 1; });
