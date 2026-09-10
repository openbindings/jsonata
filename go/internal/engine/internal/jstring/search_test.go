package jstring_test

import (
	"context"
	"errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/openbindings/jsonata/go/internal/engine/internal/jstring"
)

func TestUnitCarriageAndSearch(t *testing.T) {
	random := rand.New(rand.NewSource(90719))
	for i := 0; i < 5000; i++ {
		units := make([]uint16, random.Intn(96))
		for j := range units {
			units[j] = uint16(random.Intn(65536))
		}
		if got := jstring.Units(jstring.FromUnits(units)); !reflect.DeepEqual(got, units) {
			t.Fatalf("unit carriage %x -> %x", units, got)
		}
		start := random.Intn(len(units) + 1)
		pattern := make([]uint16, random.Intn(12))
		for j := range pattern {
			pattern[j] = uint16(random.Intn(8))
		}
		if i%2 == 0 && len(units) > 0 {
			a := random.Intn(len(units))
			pattern = units[a : a+random.Intn(len(units)-a+1)]
		}
		want := -1
		for n := start; n+len(pattern) <= len(units); n++ {
			if reflect.DeepEqual(units[n:n+len(pattern)], pattern) {
				want = n
				break
			}
		}
		got, err := jstring.IndexUnits(context.Background(), units, pattern, start)
		if err != nil || got != want {
			t.Fatalf("index got %d, want %d; %v", got, want, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := jstring.IndexUnits(ctx, []uint16{'a'}, []uint16{'a'}, 0); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
