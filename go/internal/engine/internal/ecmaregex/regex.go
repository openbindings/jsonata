// Package ecmaregex is the private JSONata regex boundary: exact UTF-16 values,
// i/m flags, immutable compiled programs, bounded cache and synchronous matching.
package ecmaregex

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/jstring"
	re "github.com/openbindings/jsonata-runtime/go/internal/engine/internal/thirdparty/regonaut"
)

type Program struct{ compiled *re.RegExpUtf16 }

var cache = struct {
	sync.Mutex
	keys   []string
	values map[string]*Program
}{values: make(map[string]*Program)}

func Compile(pattern, flags string) (*Program, error) {
	mask := re.FlagAnnexB
	seen := map[rune]bool{}
	for _, flag := range flags {
		if seen[flag] {
			return nil, fmt.Errorf("duplicate regex flag")
		}
		seen[flag] = true
		switch flag {
		case 'g':
		case 'i':
			mask |= re.FlagIgnoreCase
		case 'm':
			mask |= re.FlagMultiline
		default:
			return nil, fmt.Errorf("invalid JSONata regex flag")
		}
	}
	key := flags + ":" + pattern
	cache.Lock()
	v := cache.values[key]
	cache.Unlock()
	if v != nil {
		return v, nil
	}
	compiled, err := re.CompileBoundedUTF16(jstring.Units(pattern), mask)
	if err != nil {
		return nil, err
	}
	v = &Program{compiled}
	cache.Lock()
	defer cache.Unlock()
	if old := cache.values[key]; old != nil {
		return old, nil
	}
	if len(cache.keys) == 32 {
		delete(cache.values, cache.keys[0])
		cache.keys = cache.keys[1:]
	}
	cache.keys = append(cache.keys, key)
	cache.values[key] = v
	return v, nil
}

func (p *Program) Find(ctx context.Context, units []uint16, start int) (*re.MatchUtf16, error) {
	return p.compiled.FindBounded(ctx, units, start, re.DefaultLimits())
}

func Quote(s string) string {
	var b strings.Builder
	for _, part := range jstring.Characters(s) {
		if len(part) == 1 && strings.ContainsRune(`\^$.*+?()[]{}|/`, rune(part[0])) {
			b.WriteByte('\\')
		}
		b.WriteString(part)
	}
	return b.String()
}
