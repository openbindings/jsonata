// Package syntax validates JSONata expressions without importing the evaluator
// or registering its standard library. Parsing does not evaluate expressions,
// resolve function names, or impose arithmetic work limits on numeric literals.
package syntax

import "github.com/openbindings/jsonata/go/internal/engine/internal/parser"

// Validate checks grammar and static language constraints. Dynamic errors and
// undefined results are evaluation outcomes, not syntax errors.
func Validate(expression string) error {
	ast, err := parser.NewParser(expression).Parse()
	if err != nil {
		return err
	}
	_, err = parser.ProcessAST(ast)
	return err
}
