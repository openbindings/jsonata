package gnata_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

func applyDocumentationExpectation(t *testing.T, file string, index int, tc *testCase) {
	t.Helper()
	if os.Getenv("JSONATA_REFERENCE_EXPECTATIONS") == "1" {
		return
	}
	bytes, err := os.ReadFile("documentation-expectations.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			File, Expression, SourceSHA256, ResultJSON, Authority, Reason string
			Index                                                         int
		}
	}
	if err := json.Unmarshal(bytes, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 3 {
		t.Fatal("unreviewed documentation overlay count")
	}
	for _, c := range fixture.Cases {
		if c.File != file || c.Index != index {
			continue
		}
		source, err := os.ReadFile(filepath.Join("testdata/groups", file))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(source)) != c.SourceSHA256 || tc.Expr != c.Expression || c.Authority == "" || c.Reason == "" {
			t.Fatal("unreviewed documentation overlay source")
		}
		tc.Code, tc.Error, tc.UndefinedResult = "", nil, false
		tc.Result, err = gnata.DecodeJSON([]byte(c.ResultJSON))
		if err != nil {
			t.Fatal(err)
		}
	}
}
