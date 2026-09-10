package gnata_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
)

type testCase struct {
	Expr            string          `json:"expr"`
	ExprFile        string          `json:"expr-file"`
	Data            any             `json:"-"`
	RawData         json.RawMessage `json:"data"`
	Dataset         *string         `json:"dataset"`
	Bindings        map[string]any  `json:"bindings"`
	Result          any             `json:"result"`
	UndefinedResult bool            `json:"undefinedResult"`
	Unordered       bool            `json:"unordered"`
	Code            string          `json:"code"`
	Token           string          `json:"token"`
	Error           map[string]any  `json:"error"`
	TimeLimit       int             `json:"timelimit"`
	Depth           int             `json:"depth"`
}

// Keep expression code units intact before handing the expression to the lexer.
// encoding/json's string conversion replaces lone surrogates in fixture JSON.
// The fixture files and expected error codes remain unchanged.
func (tc *testCase) UnmarshalJSON(raw []byte) error {
	type plainCase testCase
	if err := json.Unmarshal(raw, (*plainCase)(tc)); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if expr, ok := fields["expr"]; ok {
		value, err := gnata.DecodeJSON(expr)
		if err != nil {
			return err
		}
		var valid bool
		tc.Expr, valid = value.(string)
		if !valid {
			return fmt.Errorf("fixture expression is not a string")
		}
	}
	if result, ok := fields["result"]; ok {
		var err error
		tc.Result, err = gnata.DecodeJSON(result)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tc *testCase) decodeData() error {
	if len(tc.RawData) == 0 || string(tc.RawData) == "null" {
		return nil
	}
	var err error
	tc.Data, err = gnata.DecodeJSON(tc.RawData)
	return err
}

var (
	datasetsMu sync.Mutex
	datasets   = map[string]any{}
)

func loadDataset(name string) (any, error) {
	path := filepath.Join("testdata", "datasets", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return gnata.DecodeJSON(data)
}

func getDataset(name string) (any, error) {
	datasetsMu.Lock()
	defer datasetsMu.Unlock()
	if ds, ok := datasets[name]; ok {
		return ds, nil
	}
	ds, err := loadDataset(name)
	if err != nil {
		return nil, err
	}
	datasets[name] = ds
	return ds, nil
}

func TestSuite(t *testing.T) {
	groups, err := os.ReadDir("testdata/groups")
	if err != nil {
		t.Fatalf("testdata/groups not found: %v", err)
		return
	}

	for _, g := range groups {
		if !g.IsDir() {
			continue
		}

		t.Run(g.Name(), func(t *testing.T) {
			t.Parallel()
			cases, err := os.ReadDir(filepath.Join("testdata", "groups", g.Name()))
			if err != nil {
				t.Fatalf("cannot read group %s: %v", g.Name(), err)
			}
			groupDir := filepath.Join("testdata", "groups", g.Name())
			for _, c := range cases {
				if !strings.HasSuffix(c.Name(), ".json") {
					continue
				}

				t.Run(c.Name(), func(t *testing.T) {
					raw, err := os.ReadFile(filepath.Join(groupDir, c.Name()))
					if err != nil {
						t.Fatalf("cannot read case: %v", err)
					}
					// Support both single-object and array-of-objects case files.
					if len(raw) > 0 && raw[0] == '[' {
						var tcs []testCase
						if err := json.Unmarshal(raw, &tcs); err != nil {
							t.Fatalf("cannot parse case array: %v", err)
						}
						for i, tc := range tcs {
							if err := tc.decodeData(); err != nil {
								t.Fatalf("cannot decode data for case %d: %v", i, err)
							}
							t.Run(fmt.Sprintf("%03d", i), func(t *testing.T) {
								applyDocumentationExpectation(t, g.Name()+"/"+c.Name(), i, &tc)
								resolveExprFile(t, &tc, groupDir)
								runTestCase(t, &tc)
							})
						}
						return
					}
					var tc testCase
					if err := json.Unmarshal(raw, &tc); err != nil {
						t.Fatalf("cannot parse case: %v", err)
					}
					if err := tc.decodeData(); err != nil {
						t.Fatalf("cannot decode data: %v", err)
					}
					applyOfficialExpectation(t, g.Name()+"/"+c.Name(), &tc)
					resolveExprFile(t, &tc, groupDir)
					runTestCase(t, &tc)
				})
			}
		})
	}
}

// resolveExprFile loads the expression from the ExprFile field if present.
// Missing fixtures fail qualification; they are not skipped.
func resolveExprFile(t *testing.T, tc *testCase, dir string) {
	t.Helper()
	if tc.ExprFile == "" {
		return
	}
	path := filepath.Join(dir, tc.ExprFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expr-file %s not found: %v", tc.ExprFile, err)
		return
	}
	tc.Expr = string(data)
}

func runTestCase(t *testing.T, tc *testCase) {
	t.Helper()

	// Resolve input data
	var inputData any
	if tc.Dataset != nil {
		ds, err := getDataset(*tc.Dataset)
		if err != nil {
			t.Fatalf("dataset %s not found", *tc.Dataset)
			return
		}
		inputData = ds
	} else {
		inputData = tc.Data
	}

	expr, err := gnata.Compile(tc.Expr)
	if err != nil {
		if tc.Code != "" {
			if !strings.Contains(err.Error(), tc.Code) {
				t.Errorf("compile error code: want %s, got %v", tc.Code, err)
			}
			return
		}
		t.Fatalf("unexpected compile error: %v", err)
		return
	}

	result, err := expr.EvalWithVars(context.Background(), inputData, tc.Bindings)
	if tc.Code != "" || tc.Error != nil {
		if err == nil {
			t.Fatalf("expected error %s, got result %v", tc.Code, result)
		}
		if tc.Code != "" && !strings.Contains(err.Error(), tc.Code) {
			t.Errorf("error code: want %s, got %v", tc.Code, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
		return
	}

	if tc.UndefinedResult {
		if result != nil {
			t.Errorf("expected undefined, got %v", result)
		}
		return
	}

	if tc.Unordered {
		if !deepEqualUnordered(result, tc.Result) {
			t.Errorf("result mismatch (unordered):\n  expr:   %s\n  want:   %v (%T)\n  got:    %v (%T)",
				tc.Expr, tc.Result, tc.Result, result, result)
		}
		return
	}
	if !gnata.DeepEqual(result, tc.Result) {
		t.Errorf("result mismatch:\n  expr:   %s\n  want:   %v (%T)\n  got:    %v (%T)",
			tc.Expr, tc.Result, tc.Result, result, result)
	}
}

// deepEqualUnordered compares two values ignoring array element order.
func deepEqualUnordered(a, b any) bool {
	a, b = gnata.NormalizeValue(a), gnata.NormalizeValue(b)
	switch av := a.(type) {
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		used := make([]bool, len(bv))
		for _, ai := range av {
			found := false
			for j, bi := range bv {
				if !used[j] && gnata.DeepEqual(ai, bi) {
					used[j] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return gnata.DeepEqual(a, b)
	}
}
