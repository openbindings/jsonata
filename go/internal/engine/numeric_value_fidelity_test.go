package gnata_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

func TestNumericValueFidelityEntryPaths(t *testing.T) {
	for _, tt := range []struct {
		expr, input string
		want        any
	}{
		{`$formatInteger(n,"#,##0")`, `{"n":9007199254740993}`, "9,007,199,254,740,993"},
		{`$formatInteger(n,"0")`, `{"n":1000000000000000000000000000000000000000000000000001}`, "1000000000000000000000000000000000000000000000000001"},
		{`$formatInteger(n,"0;o")`, `{"n":9007199254740993}`, "9007199254740993rd"},
		{`$parseInteger("9007199254740993","0") = 9007199254740993`, `{}`, true},
		{`$parseInteger($formatInteger(n,"w"),"w") = n`, `{"n":1000000000000000000000000000000000000000000000000001}`, true},
		{`$parseInteger($formatInteger(n,"A"),"A") = n`, `{"n":9007199254740993}`, true},
		{`$parseInteger($formatInteger(n,"Ww;o"),"Ww;o") = n`, `{"n":9007199254740993}`, true},
		{`$formatInteger(-2.00000000000000000000001,"0")`, `{}`, "-3"},
		{`(n^(<$))[0] = 9007199254740992`, `{"n":[9007199254740993,9007199254740992]}`, true},
		{`$string($power(2,0.1))`, `{}`, "1.071773462536293164213006325023342"},
		{`$string($power(1e-1000,0.125))`, `{}`, "1e-125"},
		{`$substring("abcd",1.99999999999999999999999,1.99999999999999999999999)`, `{}`, "b"},
		{`$substring("abcd",-1e-1000,2)`, `{}`, "ab"},
		{`$substring("abcd",-1e1000,1e1000)`, `{}`, "abcd"},
		{`$substring("abcd",1e1000)`, `{}`, ""},
		{`$pad("a",2.99999999999999999999999,".")`, `{}`, "a."},
		{`$pad("a",-2.99999999999999999999999,".")`, `{}`, ".a"},
		{`$fromMillis(1.99999999999999999999999)`, `{}`, "1970-01-01T00:00:00.001Z"},
		{`$fromMillis(-0.99999999999999999999999)`, `{}`, "1970-01-01T00:00:00.000Z"},
		{`$formatBase(9007199254740993,16)`, `{}`, "20000000000001"},
		{`$formatBase(2.5)`, `{}`, "2"},
		{`$formatBase(-2.5)`, `{}`, "-2"},
		{`$formatBase(99.5,2.5)`, `{}`, "1100100"},
		{`[10,20][-1e-1000]`, `{}`, float64(20)},
		{`[10,20][0.99999999999999999999999999999]`, `{}`, float64(10)},
		{`$exists([10,20][1e1000])`, `{}`, false},
		{`$string([9007199254740992..9007199254740994])`, `{}`, "[9007199254740992,9007199254740993,9007199254740994]"},
		{`$string($sqrt(2))`, `{}`, "1.414213562373095048801688724209698"},
		{`$string($sqrt(1e-1000))`, `{}`, "1e-500"},
		{`$string($sqrt(15241578753238836750495351562536198787501905199875019052100))`, `{}`, "1.2345678901234567890123456789e+29"},
		{`$formatNumber(n,"#,##0.0")`, `{"n":9007199254740993.25}`, "9,007,199,254,740,993.2"},
		{`$formatNumber(n,"0.000e0")`, `{"n":1.23456789e-1000}`, "1.235e-1000"},
		{`$formatNumber(n,"0.00%")`, `{"n":0.1000000000000000000000000000000000001}`, "10.00%"},
		{`$formatNumber(n,"0.00e0")`, `{"n":9.999}`, "10.00e0"},
		{`$formatNumber(n,"0.0;[0.0]")`, `{"n":-9007199254740993.25}`, "[9007199254740993.2]"},
		{`n = 9007199254740992`, `{"n":9007199254740993.0}`, false},
		{`n = 0`, `{"n":1e-1000}`, false},
		{`$boolean(n)`, `{"n":1e-1000}`, true},
		{`$not(n)`, `{"n":1e-1000}`, false},
		{`$string($number(n))`, `{"n":"0x20000000000001"}`, "9007199254740993"},
		{`$string($sum(n))`, `{"n":[9007199254740993,0.1,0.2]}`, "9007199254740993.3"},
		{`$string($average(n))`, `{"n":[9007199254740993,9007199254740995]}`, "9007199254740994"},
		{`$max(n) = 9007199254740993`, `{"n":[9007199254740992,9007199254740993]}`, true},
		{`$min(n) = 1e-1001`, `{"n":[1e-1000,1e-1001]}`, true},
		{`$string($floor(n))`, `{"n":9007199254740993.75}`, "9007199254740993"},
		{`$string($ceil(n))`, `{"n":9007199254740993.25}`, "9007199254740994"},
		{`$string($abs(n))`, `{"n":-9007199254740993}`, "9007199254740993"},
		{`$string($round(n))`, `{"n":9007199254740993.5}`, "9007199254740994"},
		{`$string($round(n,1))`, `{"n":9007199254740993.25}`, "9007199254740993.2"},
		{`$sort(n)[0] = 9007199254740992`, `{"n":[9007199254740993,9007199254740992]}`, true},
		{`$count($distinct(n)) = 2`, `{"n":[9007199254740993,9007199254740992,9007199254740993.0]}`, true},
		{`n in [9007199254740992]`, `{"n":9007199254740993}`, false},
		{`$eval("9007199254740993 = 9007199254740992")`, `{}`, false},
		{`0.1 + 0.2 = 0.3`, `{}`, true},
		{`$string((1/3)*3)`, `{}`, "0.9999999999999999999999999999999999"},
		{`$type(1e-1000)`, `{}`, "number"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			e, err := gnata.Compile(tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			data, err := gnata.DecodeJSON([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			var mapped map[string]json.RawMessage
			if err = json.Unmarshal([]byte(tt.input), &mapped); err != nil {
				t.Fatal(err)
			}
			paths := []func() (any, error){
				func() (any, error) { return e.Eval(context.Background(), data) },
				func() (any, error) { return e.EvalBytes(context.Background(), json.RawMessage(tt.input)) },
				func() (any, error) { return e.EvalMap(context.Background(), mapped) },
				func() (any, error) { return e.EvalBytesWithVars(context.Background(), json.RawMessage(tt.input), nil) },
			}
			for i, path := range paths {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					v, err := path()
					if err != nil || v != tt.want {
						t.Errorf("got %v (%T), error %v; want %v", v, v, err, tt.want)
					}
				})
			}
		})
	}
}
