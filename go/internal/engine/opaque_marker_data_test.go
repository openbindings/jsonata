package gnata_test

import (
	"context"
	"encoding/json"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

func TestInternalMarkerNamesAreOrdinaryData(t *testing.T) {
	for _, raw := range []string{`{"isLosslessNumber":true,"value":"9007199254740993","ordinary":7}`, `{"_jsonata_function":true,"ordinary":7}`, `{"_jsonata_lambda":true,"ordinary":7}`, `{"_jsonata_function":true,"_jsonata_lambda":true,"isLosslessNumber":true,"value":"0.1","ordinary":7}`} {
		var want any
		if err := json.Unmarshal([]byte(raw), &want); err != nil {
			t.Fatal(err)
		}
		for _, source := range []string{`$`, `$clone($)`, `$eval("$")`, raw} {
			expr, err := gnata.Compile(source)
			if err != nil {
				t.Fatal(err)
			}
			got, err := expr.EvalBytes(context.Background(), []byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			if !gnata.DeepEqual(got, want) {
				t.Fatalf("%s %s changed: %#v", source, raw, got)
			}
			if _, err := gnata.JSONValue(got); err != nil {
				t.Fatal(err)
			}
		}
	}
}
