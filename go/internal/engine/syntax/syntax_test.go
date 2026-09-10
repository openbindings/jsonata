package syntax_test

import (
	"testing"

	"github.com/openbindings/jsonata/go/internal/engine/syntax"
)

func TestLanguageMembershipWithoutEvaluation(t *testing.T) {
	for _, expr := range []string{`$`, `unknown.path`, `$unknown()`, `$flatten([])`, `$error("runtime")`, `1/0`, `1e9999999999999999999999999`, `**`, `a.**`, `$power(2,8)`, `($r := (/a/); $r("a"))`, `[$match("a", /a/)]`, `$ ~> |$|{"ok":true}|`, `0 ?? 1`, `null ?? 1`, `0 ?: 1`} {
		if err := syntax.Validate(expr); err != nil {
			t.Errorf("valid syntax %s: %v", expr, err)
		}
	}
	for _, expr := range []string{``, `2 ** 8`, `$ | $ | {"ok":true} |`, `1 | 2`, `1e+`, `01`, `1 +`, `/a/gg`, `/[a/`, `function($x){`} {
		if err := syntax.Validate(expr); err == nil {
			t.Errorf("invalid syntax accepted: %s", expr)
		}
	}
}
