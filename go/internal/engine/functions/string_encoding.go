package functions

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/openbindings/jsonata/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata/go/internal/engine/internal/parser"
)

// ── $eval ─────────────────────────────────────────────────────────────────────

func makeFnEval() evaluator.EnvAwareBuiltin {
	const maxEvalDepth = 5
	return func(args []any, focus any, env *evaluator.Environment) (any, error) {
		if len(args) == 0 {
			return nil, &evaluator.JSONataError{Code: "D3006", Message: "$eval: requires at least 1 argument"}
		}
		if args[0] == nil {
			return nil, nil
		}
		expr, ok := args[0].(string)
		if !ok {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$eval: argument must be a string"}
		}
		if err := env.IncrEvalDepth(maxEvalDepth); err != nil {
			return nil, err
		}
		defer env.DecrEvalDepth()
		p := parser.NewParser(expr)
		ast, parseErr := p.Parse()
		if parseErr != nil {
			return nil, &evaluator.JSONataError{Code: "D3120", Message: fmt.Sprintf("$eval: invalid expression: %v", parseErr)}
		}
		ast, processErr := parser.ProcessAST(ast)
		if processErr != nil {
			return nil, &evaluator.JSONataError{Code: "D3120", Message: fmt.Sprintf("$eval: invalid expression: %v", processErr)}
		}
		ctx := focus
		if len(args) >= 2 && args[1] != nil {
			ctx = args[1]
		}
		childEnv := evaluator.NewChildEnvironment(env)
		result, evalErr := evaluator.Eval(ast, ctx, childEnv)
		if evalErr != nil {
			je := &evaluator.JSONataError{}
			if errors.As(evalErr, &je) {
				if je.Code == "T1006" || je.Code == "T1005" {
					return nil, &evaluator.JSONataError{Code: "D3121", Message: fmt.Sprintf("$eval: %v", evalErr)}
				}
			}
			return nil, evalErr
		}
		return result, nil
	}
}

// ── $base64encode / $base64decode ─────────────────────────────────────────────

func fnBase64Encode(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$base64encode: argument must be a string"}
	}
	bytes := make([]byte, 0, len(s))
	for _, character := range s {
		if character > 255 {
			return nil, &evaluator.JSONataError{Code: "D3137", Message: "$base64encode: characters must be in the byte range"}
		}
		bytes = append(bytes, byte(character))
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func fnBase64Decode(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$base64decode: argument must be a string"}
	}
	// Match the standard forgiving-base64 input convention (ASCII whitespace
	// and omitted final padding), without admitting the different URL alphabet.
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			return -1
		}
		return r
	}, s)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		b, err = base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			return nil, &evaluator.JSONataError{Code: "D3137", Message: fmt.Sprintf("$base64decode: invalid base64 string: %v", err)}
		}
	}
	if !utf8.Valid(b) {
		return nil, &evaluator.JSONataError{Code: "D3137", Message: "$base64decode: invalid UTF-8 text"}
	}
	return string(b), nil
}

// ── $encodeUrl / $encodeUrlComponent / $decodeUrl / $decodeUrlComponent ───────

const encodeURISafe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'();/?:@&=+$,#"

const encodeURIComponentSafe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()"

func fnEncodeURL(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$encodeUrl: argument must be a string"}
	}
	if hasLoneSurrogate(s) {
		return nil, &evaluator.JSONataError{Code: "D3140", Message: "$encodeUrl: string contains illegal character"}
	}
	return encodeWithSafeChars(s, encodeURISafe), nil
}

func fnEncodeURLComponent(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$encodeUrlComponent: argument must be a string"}
	}
	if hasLoneSurrogate(s) {
		return nil, &evaluator.JSONataError{Code: "D3140", Message: "$encodeUrlComponent: string contains illegal character"}
	}
	return encodeWithSafeChars(s, encodeURIComponentSafe), nil
}

func hasLoneSurrogate(s string) bool {
	// A literal U+FFFD is valid Unicode, not evidence of a lone surrogate.
	// Go strings cannot encode a surrogate in well-formed UTF-8.
	return !utf8.ValidString(s)
}

func encodeWithSafeChars(s, safe string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(safe, r) {
			b.WriteRune(r)
		} else {
			encoded := url.QueryEscape(string(r))
			encoded = strings.ReplaceAll(encoded, "+", "%20")
			b.WriteString(encoded)
		}
	}
	return b.String()
}

func fnDecodeURL(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$decodeUrl: argument must be a string"}
	}
	return decodeURLValue(s, true)
}

func fnDecodeURLComponent(args []any, _ any) (any, error) {
	if len(args) == 0 || args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$decodeUrlComponent: argument must be a string"}
	}
	return decodeURLValue(s, false)
}

func decodeURLValue(s string, preserveReserved bool) (any, error) {
	decoded, err := url.PathUnescape(s)
	if err != nil || !utf8.ValidString(decoded) {
		return nil, &evaluator.JSONataError{Code: "D3140", Message: "malformed URI or invalid UTF-8"}
	}
	if !preserveReserved {
		return decoded, nil
	}
	// decodeURI, unlike decodeURIComponent, leaves reserved ASCII escapes
	// unchanged, including the original spelling of each percent escape.
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '%' {
			out.WriteByte(s[i])
			i++
			continue
		}
		b, _ := strconv.ParseUint(s[i+1:i+3], 16, 8) // validated by PathUnescape
		if strings.ContainsRune(";/?:@&=+$,#", rune(b)) {
			out.WriteString(s[i : i+3])
		} else {
			out.WriteByte(byte(b))
		}
		i += 3
	}
	return out.String(), nil
}
