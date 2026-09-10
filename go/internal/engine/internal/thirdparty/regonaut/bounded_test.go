package regonaut_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	re "github.com/openbindings/jsonata/go/internal/engine/internal/thirdparty/regonaut"
)

func units(s string) []uint16 { return utf16.Encode([]rune(s)) }
func TestBoundedCompile(t *testing.T) {
	for name, pattern := range map[string]string{"units": strings.Repeat("a", 16385), "depth": strings.Repeat("(", 129) + "a" + strings.Repeat(")", 129), "captures": strings.Repeat("(a)", 257), "program": strings.Repeat("(", 18) + "a" + strings.Repeat("){1,2}", 18)} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			_, err := re.CompileBoundedUTF16(units(pattern), re.FlagAnnexB)
			var limit *re.LimitError
			if !errors.As(err, &limit) {
				t.Fatalf("want resource error, got %v", err)
			}
			if time.Since(start) > time.Second {
				t.Fatal("compile limit was not prompt")
			}
		})
	}
}

func TestBoundedMatching(t *testing.T) {
	p, err := re.CompileBoundedUTF16(units("(a+)+$"), re.FlagAnnexB)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"steps", "memory", "cancel", "deadline"} {
		t.Run(kind, func(t *testing.T) {
			limits := re.DefaultLimits()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "steps":
				limits.Steps = 1000
			case "memory":
				limits.LiveCells = 100
			case "cancel":
				cancel()
			case "deadline":
				var c context.CancelFunc
				ctx, c = context.WithTimeout(ctx, time.Millisecond)
				defer c()
			}
			start := time.Now()
			m, err := p.FindBounded(ctx, units(strings.Repeat("a", 28)+"!"), 0, limits)
			if m != nil || err == nil {
				t.Fatalf("budget failure became a result/no-match: %v, %v", m, err)
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if kind == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
			if time.Since(start) > 200*time.Millisecond {
				t.Fatalf("unbounded cancellation latency: %v", time.Since(start))
			}
			t.Logf("%s: %T in %v", kind, err, time.Since(start))
		})
	}
}

func TestBudgetInvarianceAndConcurrency(t *testing.T) {
	pattern := units("(?<=a)(b+)")
	p, err := re.CompileBoundedUTF16(pattern, re.FlagAnnexB)
	if err != nil {
		t.Fatal(err)
	}
	pattern[0] = 'x'
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, steps := range []uint64{100, 10000, 1000000} {
				limits := re.DefaultLimits()
				limits.Steps = steps
				m, err := p.FindBounded(context.Background(), units("xabbby"), 0, limits)
				if err != nil || m == nil || m.Groups[0].Start != 2 || !reflect.DeepEqual(m.Groups[1].Data(), units("bbb")) {
					t.Errorf("budget %d changed result: %v %v", steps, m, err)
				}
			}
		}()
	}
	wg.Wait()
}
