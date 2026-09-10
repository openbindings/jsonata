package gnata_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	gnata "github.com/openbindings/jsonata/go/internal/engine"
)

// This is a separate official-policy lane, never an assertion that the stronger
// number domain is a requirement on all JSONata/OpenBindings implementations.
// Set JSONATA_REFERENCE_EXPECTATIONS=1 to observe the unchanged reference lane.
type officialExpectation struct {
	Sources    map[string]string `json:"sources"`
	ResultJSON *string           `json:"resultJSON"`
	Code       string            `json:"code"`
	Reason     string            `json:"reason"`
}

var officialExpectations struct {
	sync.Once
	cases map[string]officialExpectation
	err   error
}

func loadOfficialExpectations() {
	raw, err := os.ReadFile("official-expectation-overlays.json")
	if err != nil {
		officialExpectations.err = err
		return
	}
	var overlay struct {
		Schema int                            `json:"schema"`
		Cases  map[string]officialExpectation `json:"cases"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		officialExpectations.err = err
		return
	}
	if overlay.Schema != 1 || len(overlay.Cases) != 20 {
		officialExpectations.err = fmt.Errorf("unreviewed expectation-overlay schema/count")
		return
	}
	for id, delta := range overlay.Cases {
		if delta.Reason == "" || (delta.ResultJSON == nil) == (delta.Code == "") {
			officialExpectations.err = fmt.Errorf("invalid declared delta %s", id)
			return
		}
		if _, ok := delta.Sources["groups/"+id]; !ok {
			officialExpectations.err = fmt.Errorf("delta missing original source %s", id)
			return
		}
		for name, hash := range delta.Sources {
			raw, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				officialExpectations.err = fmt.Errorf("cannot read source for delta %s: %s: %w", id, name, err)
				return
			}
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != hash {
				officialExpectations.err = fmt.Errorf("source changed for delta %s: %s", id, name)
				return
			}
		}
	}
	officialExpectations.cases = overlay.Cases
}

func applyOfficialExpectation(t *testing.T, id string, tc *testCase) {
	t.Helper()
	if os.Getenv("JSONATA_REFERENCE_EXPECTATIONS") == "1" {
		return
	}
	officialExpectations.Do(loadOfficialExpectations)
	if officialExpectations.err != nil {
		t.Fatal(officialExpectations.err)
	}
	if delta, ok := officialExpectations.cases[id]; ok {
		tc.Code, tc.Error, tc.UndefinedResult = delta.Code, nil, false
		if delta.ResultJSON != nil {
			var err error
			tc.Result, err = gnata.DecodeJSON([]byte(*delta.ResultJSON))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestPublicDeepEqualExactNumbers(t *testing.T) {
	if gnata.DeepEqual(json.Number("9007199254740993"), float64(9007199254740992)) {
		t.Fatal("public comparison rounded adjacent identifiers")
	}
	if !gnata.DeepEqual(json.Number("9007199254740993.0"), json.Number("9007199254740993")) {
		t.Fatal("public comparison confused spelling with numerical value")
	}
}
