package evaluator

import (
	"context"
	"strings"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/ecmaregex"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/jstring"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
	re "github.com/openbindings/jsonata-runtime/go/internal/engine/internal/thirdparty/regonaut"
)

// RegexValue is a callable, never a JSON object with magic member names.
type RegexValue struct{ Pattern, Flags string }

// Regex stores only an immutable program. Input, cursor and budgets belong to
// each call, so concurrent/reentrant use cannot mutate another match's state.
type (
	Regex struct{ program *ecmaregex.Program }
	Match struct {
		Index, Length int
		units         []uint16
		program       *ecmaregex.Program
		ctx           context.Context
		groups        []re.GroupUtf16
	}
)

type Group struct {
	Index, Length int
	Captured      bool
	value         string
}

func (g *Group) String() string  { return g.value }
func (m *Match) String() string  { return jstring.FromUnits(m.units[m.Index : m.Index+m.Length]) }
func (m *Match) GroupCount() int { return len(m.groups) }
func (m *Match) GroupByNumber(i int) *Group {
	if i < 0 || i >= len(m.groups) || m.groups[i].Start < 0 {
		return &Group{}
	}
	g := m.groups[i]
	return &Group{g.Start, g.End - g.Start, true, jstring.FromUnits(g.Data())}
}

func findRegex(p *ecmaregex.Program, ctx context.Context, units []uint16, start int) (*Match, error) {
	m, err := p.Find(ctx, units, start)
	if err != nil || m == nil {
		return nil, err
	}
	g := m.Groups[0]
	return &Match{g.Start, g.End - g.Start, units, p, ctx, m.Groups}, nil
}

func (r *Regex) FindStringMatchContext(ctx context.Context, s string, start int) (*Match, error) {
	return findRegex(r.program, ctx, jstring.Units(s), start)
}

func (r *Regex) FindStringMatch(s string) (*Match, error) {
	return r.FindStringMatchContext(context.Background(), s, 0)
}

func (r *Regex) MatchString(s string) (bool, error) {
	m, err := r.FindStringMatch(s)
	return m != nil, err
}

func (m *Match) FindNextMatch() (*Match, error) {
	end := m.Index + m.Length
	if end >= len(m.units) {
		return nil, nil
	}
	next, err := findRegex(m.program, m.ctx, m.units, end)
	if err != nil {
		return nil, err
	}
	if next != nil && next.Length == 0 {
		return nil, &JSONataError{Code: "D1004", Message: "regex matched a zero-length string while advancing"}
	}
	return next, nil
}

func CachedCompileRegex(pattern, flags string) (*Regex, error) {
	p, err := ecmaregex.Compile(pattern, flags)
	if err != nil {
		return nil, err
	}
	return &Regex{p}, nil
}
func CompileLiteralRegex(s string) (*Regex, error) { return CachedCompileRegex(ecmaregex.Quote(s), "") }

func (m *Match) Value() any {
	groups := make([]any, 0, m.GroupCount()-1)
	for i := 1; i < m.GroupCount(); i++ {
		g := m.GroupByNumber(i)
		if g.Captured {
			groups = append(groups, g.String())
		} else {
			groups = append(groups, nil)
		}
	}
	value := NewOrderedMap()
	value.Set("match", m.String())
	value.Set("start", float64(m.Index))
	value.Set("end", float64(m.Index+m.Length))
	value.Set("groups", NewArray(groups))
	value.Set("next", EnvAwareBuiltin(func(_ []any, _ any, env *Environment) (any, error) {
		if err := env.Context().Err(); err != nil {
			return nil, err
		}
		next, err := m.FindNextMatch()
		if err != nil || next == nil {
			return nil, err
		}
		return next.Value(), nil
	}))
	return value
}

func applyRegexTest(input any, value *RegexValue, start any, env *Environment) (any, error) {
	s, ok := input.(string)
	if !ok {
		return nil, &JSONataError{Code: "T0410", Message: "matcher requires a string"}
	}
	index := 0
	if start != nil {
		if !IsNumeric(start) {
			return nil, &JSONataError{Code: "T0410", Message: "matcher start must be a number"}
		}
		index = numeric.ClippedTrunc(start, 0, len(jstring.Units(s))+1)
	}
	r, err := CachedCompileRegex(value.Pattern, value.Flags)
	if err != nil {
		return nil, err
	}
	m, err := r.FindStringMatchContext(env.Context(), s, index)
	if err != nil || m == nil {
		return nil, err
	}
	return m.Value(), nil
}

func evalRegex(raw string) *RegexValue {
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		return &RegexValue{raw[:i], raw[i+1:]}
	}
	return &RegexValue{Pattern: raw}
}
