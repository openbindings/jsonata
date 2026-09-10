// Package syntax validates JSONata grammar without loading the evaluator or
// registering the standard library. Dynamic failures are evaluation outcomes.
package syntax

import enginesyntax "github.com/openbindings/jsonata-runtime/go/internal/engine/syntax"

func Validate(expression string) error { return enginesyntax.Validate(expression) }
