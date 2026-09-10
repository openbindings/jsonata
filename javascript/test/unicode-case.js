'use strict';
const assert=require('assert');
const jsonata=require('../src/jsonata');
const casing=require('../src/unicode-case');

describe('Pinned default Unicode casing',function(){
    it('uses Unicode 16 without locale tailoring or normalization',function(){
        assert.strictEqual(casing.version,'16.0.0');
        for(const [source,upper,lower]of [
            ['','', ''],['aBc','ABC','abc'],['Straße','STRASSE','straße'],
            ['İIıi','İIII','i\u0307iıi'],['é e\u0301','É E\u0301','é e\u0301'],
            ['\u{10d50}','\u{10d50}','\u{10d70}'],['\u{10d70}','\u{10d50}','\u{10d70}'],
            ['\ud800A\udfff','\ud800A\udfff','\ud800a\udfff']
        ]){assert.strictEqual(casing.upper(source),upper);assert.strictEqual(casing.lower(source),lower);}
    });
    it('uses original-string, possessive final-sigma contexts',async function(){
        for(const [source,expected]of [
            ['Σ','σ'],['ΟΣ','ος'],['ΟΣΑ','οσα'],['\u0345Σ','\u0345σ'],
            ['AΣ\u0345','aς\u0345'],['A\u0345Σ','a\u0345ς'],['AΣ\u0345B','aσ\u0345b'],
            ['A\ud800Σ','a\ud800σ'],['AΣ\ud800B','aς\ud800b'],
            ['\u{10d50}Σ','\u{10d70}ς'],['AΣ\u{10d50}','aσ\u{10d70}']
        ]){assert.strictEqual(await jsonata('$lowercase($)').evaluate(source),expected);}
    });
});
