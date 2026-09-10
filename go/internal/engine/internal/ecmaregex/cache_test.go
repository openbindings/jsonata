package ecmaregex

import (
	"fmt"
	"testing"
)

func TestCacheCardinality(t *testing.T) {
	for i := 0; i < 1000; i++ {
		if _, err := Compile(fmt.Sprintf("a%d", i), ""); err != nil {
			t.Fatal(err)
		}
	}
	cache.Lock()
	defer cache.Unlock()
	if len(cache.values) != 32 || len(cache.keys) != 32 {
		t.Fatalf("unbounded cache %d/%d", len(cache.values), len(cache.keys))
	}
}
