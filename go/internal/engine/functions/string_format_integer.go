package functions

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/evaluator"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/numeric"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/unicodecase"
)

// ── $formatBase ───────────────────────────────────────────────────────────────

func fnFormatBase(args []any, focus any, env *evaluator.Environment) (any, error) {
	if len(args) < 1 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$formatBase: requires at least 1 argument"}
	}
	numArg := args[0]
	if numArg == nil {
		numArg = focus
	}
	if numArg == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(numArg) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$formatBase: argument 1 must be a number"}
	}
	base := 10
	if len(args) >= 2 && args[1] != nil {
		if !evaluator.IsNumeric(args[1]) {
			return nil, &evaluator.JSONataError{Code: "T0410", Message: "$formatBase: argument 2 must be a number"}
		}
		bf, err := env.NumberLimits().Calculate("round", args[1], nil, 0)
		if err != nil {
			return nil, err
		}
		b, ok := numeric.TruncInt64(bf)
		if !ok || b < 2 || b > 36 {
			return nil, &evaluator.JSONataError{Code: "D3100", Message: "$formatBase: base must be between 2 and 36"}
		}
		base = int(b)
	}
	if base < 2 || base > 36 {
		return nil, &evaluator.JSONataError{Code: "D3100", Message: "$formatBase: base must be between 2 and 36"}
	}
	rounded, err := env.NumberLimits().Calculate("round", numArg, nil, 0)
	if err != nil {
		return nil, err
	}
	n, err := env.NumberLimits().Integer(rounded)
	if err != nil {
		return nil, err
	}
	return n.Text(base), nil
}

// ── $formatInteger ────────────────────────────────────────────────────────────

func fnFormatInteger(args []any, _ any, env *evaluator.Environment) (any, error) {
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$formatInteger: requires 2 arguments"}
	}
	if args[0] == nil {
		return nil, nil
	}
	if !evaluator.IsNumeric(args[0]) {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$formatInteger: argument 1 must be a number"}
	}
	picture, ok := args[1].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$formatInteger: argument 2 must be a string"}
	}
	// The reference's integer-format convention is floor; preserve the assigned
	// value until that explicit integer operation.
	value, err := env.NumberLimits().Calculate("floor", args[0], nil, 0)
	if err != nil {
		return nil, err
	}
	n, err := env.NumberLimits().Integer(value)
	if err != nil {
		return nil, err
	}
	return formatIntegerWithPicture(n, picture, env.NumberLimits())
}

func formatIntegerWithPicture(n *big.Int, picture string, limits numeric.Limits) (string, error) {
	formatToken, modifier := splitPictureModifier(picture)

	negative, absN := n.Sign() < 0, new(big.Int).Abs(n)

	var result string
	switch formatToken {
	case "w":
		if result = bigIntToWords(absN); modifier == "o" {
			result = applyOrdinalWord(result)
		}
	case "W":
		if result = strings.ToUpper(bigIntToWords(absN)); modifier == "o" {
			result = strings.ToUpper(applyOrdinalWord(bigIntToWords(absN)))
		}
	case "Ww":
		if result = toTitleCase(bigIntToWords(absN)); modifier == "o" {
			result = toTitleCase(applyOrdinalWord(bigIntToWords(absN)))
		}
	case "i":
		if !absN.IsInt64() || absN.Int64()/1000 > int64(limits.MaxDigits) {
			return "", &numeric.Error{Kind: "work-limit", Detail: "Roman numeral exceeds output budget"}
		}
		result = toRoman(absN.Int64(), false)
	case "I":
		if !absN.IsInt64() || absN.Int64()/1000 > int64(limits.MaxDigits) {
			return "", &numeric.Error{Kind: "work-limit", Detail: "Roman numeral exceeds output budget"}
		}
		result = toRoman(absN.Int64(), true)
	default:
		runes := []rune(formatToken)
		if len(runes) == 1 {
			ch := runes[0]
			if ch >= 'a' && ch <= 'z' {
				result = bigIntToAlphabetic(absN, 'a')
				break
			}
			if ch >= 'A' && ch <= 'Z' && ch != 'W' && ch != 'I' {
				result = bigIntToAlphabetic(absN, 'A')
				break
			}
		}
		if !slices.ContainsFunc(runes, func(c rune) bool {
			return c == '#' || (c >= '0' && c <= '9') || unicodeDigitZero(c) != 0
		}) {
			return "", &evaluator.JSONataError{Code: "D3130", Message: fmt.Sprintf("$formatInteger: unsupported picture string %q", formatToken)}
		}
		var err error
		result, err = formatIntegerDecimal(absN, formatToken)
		if err != nil {
			return "", err
		}
		if modifier == "o" {
			result += ordinalSuffix(new(big.Int).Mod(absN, big.NewInt(100)).Int64())
		}
		if negative {
			result = "-" + result
		}
		return result, nil
	}

	if negative {
		result = "-" + result
	}
	return result, nil
}

func splitPictureModifier(picture string) (token, modifier string) {
	if token, modifier, ok := strings.Cut(picture, ";"); ok {
		return token, modifier
	}
	return picture, "c"
}

func formatIntegerDecimal(n *big.Int, picture string) (string, error) {
	runes := []rune(picture)

	zeroRune := '0'
	foundFamily := false
	for _, c := range runes {
		if c >= '0' && c <= '9' {
			if foundFamily && zeroRune != '0' {
				return "", &evaluator.JSONataError{Code: "D3131", Message: "$formatInteger: mixed digit families in picture"}
			}
			zeroRune = '0'
			foundFamily = true
			continue
		}
		if z := unicodeDigitZero(c); z != 0 {
			if foundFamily && zeroRune != z {
				return "", &evaluator.JSONataError{Code: "D3131", Message: "$formatInteger: mixed digit families in picture"}
			}
			if foundFamily && zeroRune == '0' {
				return "", &evaluator.JSONataError{Code: "D3131", Message: "$formatInteger: mixed digit families in picture"}
			}
			zeroRune = z
			foundFamily = true
		}
	}

	mandatoryCount := 0
	totalDigits := 0
	for _, c := range runes {
		if c == '#' {
			totalDigits++
		} else if (c >= '0' && c <= '9') || isUnicodeDigit(c) {
			mandatoryCount++
			totalDigits++
		}
	}
	if totalDigits == 0 {
		return "", &evaluator.JSONataError{Code: "D3131", Message: "$formatInteger: no digit placeholders in picture"}
	}

	type grpInfo struct {
		sep  rune
		posR int
	}
	var grpInfos []grpInfo
	digitFromRight := 0
	for i := len(runes) - 1; i >= 0; i-- {
		c := runes[i]
		if c == '#' || (c >= '0' && c <= '9') || isUnicodeDigit(c) {
			digitFromRight++
		} else if digitFromRight > 0 {
			grpInfos = append(grpInfos, grpInfo{c, digitFromRight})
		}
	}

	digits := n.String()
	if len(digits) < mandatoryCount {
		for len(digits) < mandatoryCount {
			digits = "0" + digits
		}
	}

	type grpAnon = struct {
		sep  rune
		posR int
	}
	var grpAnons []grpAnon
	for _, g := range grpInfos {
		grpAnons = append(grpAnons, grpAnon{g.sep, g.posR})
	}

	if len(grpAnons) > 0 && digits != "" {
		digits = applyIntegerGrouping(digits, grpAnons)
	}

	if zeroRune != '0' {
		digits = applyDigitFamilyRune(digits, zeroRune)
	}

	return digits, nil
}

func applyIntegerGrouping(digits string, grps []struct {
	sep  rune
	posR int
},
) string {
	if len(grps) == 0 {
		return digits
	}

	posSet := map[int]rune{}
	for _, g := range grps {
		posSet[g.posR] = g.sep
	}

	allSameSep := true
	if len(grps) > 1 {
		for i := 1; i < len(grps); i++ {
			if grps[i].sep != grps[0].sep {
				allSameSep = false
				break
			}
		}
	}

	isRegular := true
	if len(grps) > 1 {
		gap0 := grps[0].posR
		for i := 1; i < len(grps); i++ {
			if grps[i].posR-grps[i-1].posR != gap0 {
				isRegular = false
				break
			}
		}
	}

	if allSameSep && isRegular && len(grps) > 0 {
		interval := grps[0].posR
		rightmostSep := grps[0].sep
		maxLen := len(digits) + 1
		for pos := interval; pos <= maxLen; pos += interval {
			if _, exists := posSet[pos]; !exists {
				posSet[pos] = rightmostSep
			}
		}
	}

	runes := []rune(digits)
	var result []rune
	for i, ch := range runes {
		posFromRight := len(runes) - i
		if sep, ok := posSet[posFromRight]; ok && i > 0 {
			result = append(result, sep)
		}
		result = append(result, ch)
	}
	return string(result)
}

func applyDigitFamilyRune(s string, zero rune) string {
	var sb strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			sb.WriteRune(zero + c - '0')
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

func unicodeDigitZero(c rune) rune {
	return unicodecase.DigitZero(c)
}

func isUnicodeDigit(c rune) bool {
	return unicodeDigitZero(c) != 0
}

func ordinalSuffix(n int64) string {
	abs := max(n, -n)
	mod100 := abs % 100
	mod10 := abs % 10
	if mod100 >= 11 && mod100 <= 13 {
		return "th"
	}
	switch mod10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func applyOrdinalWord(word string) string {
	ordinals := map[string]string{
		"one": "first", "two": "second", "three": "third", "four": "fourth",
		"five": "fifth", "six": "sixth", "seven": "seventh", "eight": "eighth",
		"nine": "ninth", "ten": "tenth", "eleven": "eleventh", "twelve": "twelfth",
		"thirteen": "thirteenth", "fourteen": "fourteenth", "fifteen": "fifteenth",
		"sixteen": "sixteenth", "seventeen": "seventeenth", "eighteen": "eighteenth",
		"nineteen": "nineteenth", "twenty": "twentieth", "thirty": "thirtieth",
		"forty": "fortieth", "fifty": "fiftieth", "sixty": "sixtieth",
		"seventy": "seventieth", "eighty": "eightieth", "ninety": "ninetieth",
		"hundred": "hundredth", "thousand": "thousandth", "million": "millionth",
		"billion": "billionth", "trillion": "trillionth",
	}
	last := ""
	sep := ""
	prefix := ""
	for i := len(word) - 1; i >= 0; i-- {
		if word[i] == ' ' || word[i] == '-' {
			sep = string(word[i])
			prefix = word[:i]
			last = word[i+1:]
			break
		}
		if i == 0 {
			last = word
			prefix = ""
			sep = ""
		}
	}
	if ord, ok := ordinals[last]; ok {
		return prefix + sep + ord
	}
	if strings.HasSuffix(last, "y") {
		return prefix + sep + last[:len(last)-1] + "ieth"
	}
	return prefix + sep + last + "th"
}

func intToWords(n int64) string {
	if n == 0 {
		return "zero"
	}
	if n < 0 {
		return "minus " + intToWords(-n)
	}

	ones := []string{
		"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen",
	}
	tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

	var belowThousand func(int64) string
	belowThousand = func(n int64) string {
		if n == 0 {
			return ""
		}
		if n < 20 {
			return ones[n]
		}
		if n < 100 {
			if n%10 == 0 {
				return tens[n/10]
			}
			return tens[n/10] + "-" + ones[n%10]
		}
		rem := n % 100
		if rem == 0 {
			return ones[n/100] + " hundred"
		}
		return ones[n/100] + " hundred and " + belowThousand(rem)
	}

	type scale struct {
		name string
		val  int64
	}

	baseScales := []scale{
		{"trillion", 1_000_000_000_000},
		{"billion", 1_000_000_000},
		{"million", 1_000_000},
		{"thousand", 1_000},
	}

	var toWords func(int64) string
	toWords = func(n int64) string {
		if n == 0 {
			return ""
		}
		if n < 1000 {
			return belowThousand(n)
		}

		for _, sc := range baseScales {
			if n < sc.val {
				continue
			}
			q := n / sc.val
			rem := n % sc.val
			qWord := toWords(q)
			result := qWord + " " + sc.name
			if rem > 0 {
				remWord := toWords(rem)
				if rem < 100 {
					result += " and " + remWord
				} else {
					result += ", " + remWord
				}
			}
			return result
		}
		return belowThousand(n)
	}

	result := toWords(n)
	return result
}

func toTitleCase(s string) string {
	lowercase := map[string]bool{"and": true, "or": true, "of": true, "the": true}
	var sb strings.Builder
	capitalizeNext := true
	wordBuf := strings.Builder{}
	flush := func() {
		if wordBuf.Len() == 0 {
			return
		}
		word := wordBuf.String()
		lower := strings.ToLower(word)
		if capitalizeNext || !lowercase[lower] {
			if lower != "" {
				sb.WriteString(strings.ToUpper(lower[:1]) + lower[1:])
			}
		} else {
			sb.WriteString(lower)
		}
		capitalizeNext = false
		wordBuf.Reset()
	}
	for _, c := range s {
		if c == ' ' || c == ',' || c == '-' {
			flush()
			sb.WriteRune(c)
			capitalizeNext = (c == '-')
		} else {
			wordBuf.WriteRune(c)
		}
	}
	flush()
	return sb.String()
}

func toRoman(n int64, upper bool) string {
	if n <= 0 {
		return ""
	}
	vals := []int64{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var sb strings.Builder
	for i, v := range vals {
		for n >= v {
			sb.WriteString(syms[i])
			n -= v
		}
	}
	result := sb.String()
	if !upper {
		return strings.ToLower(result)
	}
	return result
}

// ── $parseInteger ─────────────────────────────────────────────────────────────

func fnParseInteger(args []any, _ any, env *evaluator.Environment) (any, error) {
	if len(args) < 2 {
		return nil, &evaluator.JSONataError{Code: "D3006", Message: "$parseInteger: requires 2 arguments"}
	}
	if args[0] == nil {
		return nil, nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$parseInteger: argument 1 must be a string"}
	}
	picture, ok := args[1].(string)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "T0410", Message: "$parseInteger: argument 2 must be a string"}
	}
	n, err := parseIntegerWithPicture(s, picture, env.NumberLimits())
	if err != nil {
		return nil, err
	}
	value := numeric.FromText(n.String())
	if _, err := env.NumberLimits().Integer(value); err != nil {
		return nil, err
	}
	return value, nil
}

func parseIntegerWithPicture(s, picture string, limits numeric.Limits) (*big.Int, error) {
	formatToken, _ := splitPictureModifier(picture)
	switch formatToken {
	case "w", "W", "Ww":
		return wordsToBigInt(strings.ToLower(s), limits)
	case "i", "I":
		// A Roman input's value is at most 1000 times its in-memory length.
		n, err := fromRoman(strings.ToUpper(s))
		return big.NewInt(n), err
	default:
		runes := []rune(formatToken)
		if len(runes) == 1 && ((runes[0] >= 'a' && runes[0] <= 'z') || (runes[0] >= 'A' && runes[0] <= 'Z')) {
			n := new(big.Int)
			for _, c := range strings.ToLower(s) {
				if c < 'a' || c > 'z' {
					return nil, &evaluator.JSONataError{Code: "D3137", Message: "invalid alphabetic integer"}
				}
				n.Mul(n, big.NewInt(26)).Add(n, big.NewInt(int64(c-'a'+1)))
				if len(n.String()) > int(limits.MaxDigits) {
					return nil, &numeric.Error{Kind: "work-limit", Detail: "parsed integer exceeds digit budget"}
				}
			}
			return n, nil
		}
	}
	zeroRune := '0'
	hasMandatory := false
	for _, c := range formatToken {
		if c >= '0' && c <= '9' {
			zeroRune = '0'
			hasMandatory = true
		} else if z := unicodeDigitZero(c); z != 0 {
			zeroRune = z
			hasMandatory = true
		}
	}
	if !hasMandatory {
		return nil, &evaluator.JSONataError{Code: "D3130", Message: "integer picture requires a mandatory digit"}
	}
	var digits strings.Builder
	for _, c := range s {
		switch {
		case c == '-' || (c >= '0' && c <= '9'):
			digits.WriteRune(c)
		case zeroRune != '0' && c >= zeroRune && c <= zeroRune+9:
			digits.WriteRune('0' + c - zeroRune)
		}
	}
	if digits.Len() > int(limits.MaxDigits)+1 {
		return nil, &numeric.Error{Kind: "work-limit", Detail: "parsed integer exceeds digit budget"}
	}
	n, ok := new(big.Int).SetString(digits.String(), 10)
	if !ok {
		return nil, &evaluator.JSONataError{Code: "D3137", Message: "cannot parse integer"}
	}
	return n, nil
}

func deOrdinalise(s string) string {
	irregulars := map[string]string{
		"first": "one", "second": "two", "third": "three", "fourth": "four",
		"fifth": "five", "sixth": "six", "seventh": "seven", "eighth": "eight",
		"ninth": "nine", "tenth": "ten", "eleventh": "eleven", "twelfth": "twelve",
		"thirteenth": "thirteen", "fourteenth": "fourteen", "fifteenth": "fifteen",
		"sixteenth": "sixteen", "seventeenth": "seventeen", "eighteenth": "eighteen",
		"nineteenth": "nineteen", "twentieth": "twenty", "thirtieth": "thirty",
		"fortieth": "forty", "fiftieth": "fifty", "sixtieth": "sixty",
		"seventieth": "seventy", "eightieth": "eighty", "ninetieth": "ninety",
		"hundredth": "hundred", "thousandth": "thousand", "millionth": "million",
		"billionth": "billion", "trillionth": "trillion",
		"zeroth": "zero",
	}
	lastIdx := -1
	lastSep := ' '
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ' ' || s[i] == '-' {
			lastIdx = i
			lastSep = rune(s[i])
			break
		}
	}
	var prefix, lastWord string
	if lastIdx >= 0 {
		prefix = s[:lastIdx]
		lastWord = s[lastIdx+1:]
	} else {
		lastWord = s
	}
	if cardinal, ok := irregulars[lastWord]; ok {
		if lastIdx >= 0 {
			return prefix + string(lastSep) + cardinal
		}
		return cardinal
	}
	return s
}

func wordsToBigInt(s string, limits numeric.Limits) (*big.Int, error) {
	s = deOrdinalise(s)
	values := map[string]int64{
		"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9,
		"ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19,
		"twenty": 20, "thirty": 30, "forty": 40, "fifty": 50, "sixty": 60, "seventy": 70, "eighty": 80, "ninety": 90,
		"hundred": 100, "thousand": 1000, "million": 1000000, "billion": 1000000000, "trillion": 1000000000000,
	}
	type segment struct {
		value *big.Int
		scale int64
	}
	segments := []segment{}
	current := new(big.Int)
	for _, word := range strings.Fields(strings.NewReplacer("-", " ", ",", " ").Replace(s)) {
		if word == "and" {
			continue
		}
		v, ok := values[word]
		if !ok {
			return nil, &evaluator.JSONataError{Code: "D3137", Message: "unknown integer word " + word}
		}
		switch {
		case v < 100:
			current.Add(current, big.NewInt(v))
		case v == 100:
			current.Mul(current, big.NewInt(v))
		default:
			// A larger following magnitude applies to the whole preceding
			// smaller-magnitude phrase: "nine thousand and seven trillion".
			for len(segments) > 0 && segments[len(segments)-1].scale <= v {
				current.Add(current, segments[len(segments)-1].value)
				segments = segments[:len(segments)-1]
			}
			current.Mul(current, big.NewInt(v))
			if len(current.String()) > int(limits.MaxDigits) {
				return nil, &numeric.Error{Kind: "work-limit", Detail: "parsed integer exceeds digit budget"}
			}
			segments = append(segments, segment{current, v})
			current = new(big.Int)
		}
	}
	for _, v := range segments {
		current.Add(current, v.value)
	}
	return current, nil
}

func fromRoman(s string) (int64, error) {
	vals := map[rune]int64{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	runes := []rune(s)
	var total int64
	for i, c := range runes {
		v, ok := vals[c]
		if !ok {
			return 0, &evaluator.JSONataError{Code: "D3137", Message: fmt.Sprintf("$parseInteger: invalid Roman numeral %q", string(c))}
		}
		if i+1 < len(runes) {
			if next, ok2 := vals[runes[i+1]]; ok2 && next > v {
				total -= v
				continue
			}
		}
		total += v
	}
	return total, nil
}

func bigIntToAlphabetic(value *big.Int, base rune) string {
	n := new(big.Int).Set(value)
	rem := new(big.Int)
	chars := []rune{}
	for n.Sign() > 0 {
		n.Sub(n, big.NewInt(1))
		n.QuoRem(n, big.NewInt(26), rem)
		chars = append(chars, base+rune(rem.Int64()))
	}
	slices.Reverse(chars)
	return string(chars)
}

func bigIntToWords(n *big.Int) string {
	if n.Cmp(big.NewInt(1000)) < 0 {
		return intToWords(n.Int64())
	}
	scales := []struct {
		name  string
		value int64
	}{
		{"trillion", 1000000000000}, {"billion", 1000000000}, {"million", 1000000}, {"thousand", 1000},
	}
	for _, sc := range scales {
		if n.Cmp(big.NewInt(sc.value)) < 0 {
			continue
		}
		q, r := new(big.Int), new(big.Int)
		q.QuoRem(n, big.NewInt(sc.value), r)
		words := bigIntToWords(q) + " " + sc.name
		if r.Sign() > 0 {
			if r.Cmp(big.NewInt(100)) < 0 {
				words += " and "
			} else {
				words += ", "
			}
			words += bigIntToWords(r)
		}
		return words
	}
	panic("unreachable integer-word scale")
}
