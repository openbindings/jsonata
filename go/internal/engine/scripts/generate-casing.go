//go:build ignore

// Reproduce data.go using only Go and checksum-verified Unicode inputs.
// Run from the repository root: go run scripts/generate-casing.go INPUT_DIRECTORY
// An optional second argument selects a verification output directory.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func number(s string) int {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 16, 32)
	check(err)
	return int(n)
}

func scalars(s string) string {
	var out strings.Builder
	for _, v := range strings.Fields(s) {
		out.WriteRune(rune(number(v)))
	}
	return out.String()
}

func main() {
	if len(os.Args) < 2 {
		panic("supply the pinned Unicode input directory")
	}
	dir := "internal/unicodecase"
	destination := dir
	if len(os.Args) > 2 {
		destination = os.Args[2]
	}
	metadata, err := os.ReadFile(filepath.Join(dir, "SOURCE.json"))
	check(err)
	var source struct {
		Unicode string
		Sources map[string][2]string
	}
	check(json.Unmarshal(metadata, &source))
	if source.Unicode != "16.0.0" {
		panic("review the casing algorithm before changing Unicode versions")
	}
	texts := map[string]string{}
	for name, identity := range source.Sources {
		b, err := os.ReadFile(filepath.Join(os.Args[1], name))
		check(err)
		digest := sha256.Sum256(b)
		if hex.EncodeToString(digest[:]) != identity[1] {
			panic("Unicode input mismatch: " + name)
		}
		texts[name] = string(b)
	}
	upper, lower := map[int]string{}, map[int]string{}
	var decimalZeros []int
	for _, line := range strings.Split(texts["UnicodeData.txt"], "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, ";")
		cp := number(f[0])
		if f[2] == "Nd" && f[6] == "0" {
			decimalZeros = append(decimalZeros, cp)
		}
		if f[12] != "" {
			upper[cp] = scalars(f[12])
		}
		if f[13] != "" {
			lower[cp] = scalars(f[13])
		}
	}
	contexts := 0
	for _, line := range strings.Split(texts["SpecialCasing.txt"], "\n") {
		body := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if body == "" {
			continue
		}
		f := strings.Split(body, ";")
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		cp := number(f[0])
		condition := strings.Fields(f[4])
		if len(condition) > 0 {
			if condition[0] == "tr" || condition[0] == "az" || condition[0] == "lt" {
				continue
			}
			if f[4] != "Final_Sigma" || cp != 0x3a3 || scalars(f[1]) != "ς" {
				panic("unimplemented default casing context")
			}
			contexts++
			continue
		}
		lower[cp], upper[cp] = scalars(f[1]), scalars(f[3])
	}
	if contexts != 1 {
		panic("default contextual rules changed")
	}
	ranges := map[string][][2]int{"Cased": {}, "Case_Ignorable": {}}
	for _, line := range strings.Split(texts["DerivedCoreProperties.txt"], "\n") {
		body := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if body == "" {
			continue
		}
		f := strings.Split(body, ";")
		property := strings.TrimSpace(f[1])
		if _, ok := ranges[property]; !ok {
			continue
		}
		bounds := strings.Split(strings.TrimSpace(f[0]), "..")
		a := number(bounds[0])
		b := a
		if len(bounds) == 2 {
			b = number(bounds[1])
		}
		ranges[property] = append(ranges[property], [2]int{a, b})
	}
	var out strings.Builder
	out.WriteString("// Code generated from pinned Unicode 16.0.0 data; DO NOT EDIT.\n// Copyright Unicode, Inc. See LICENSE and SOURCE.json.\npackage unicodecase\nconst Version = \"16.0.0\"\n")
	for _, table := range []struct {
		name   string
		values map[int]string
	}{{"upperMapping", upper}, {"lowerMapping", lower}} {
		fmt.Fprintf(&out, "var %s = map[rune]string{\n", table.name)
		keys := make([]int, 0, len(table.values))
		for cp := range table.values {
			keys = append(keys, cp)
		}
		sort.Ints(keys)
		for _, cp := range keys {
			// JSON emits all valid letter scalars literally, including Unicode 16
			// characters not classified as printable by the generating Go host.
			quoted, err := json.Marshal(table.values[cp])
			check(err)
			fmt.Fprintf(&out, "0x%x:%s,\n", cp, quoted)
		}
		out.WriteString("}\n")
	}
	for _, table := range []struct{ name, property string }{{"casedRanges", "Cased"}, {"ignorableRanges", "Case_Ignorable"}} {
		fmt.Fprintf(&out, "var %s = [][2]rune{\n", table.name)
		for _, r := range ranges[table.property] {
			fmt.Fprintf(&out, "{0x%x,0x%x},\n", r[0], r[1])
		}
		out.WriteString("}\n")
	}
	out.WriteString("var decimalZeros = []rune{\n")
	for _, cp := range decimalZeros {
		fmt.Fprintf(&out, "0x%x,\n", cp)
	}
	out.WriteString("}\n")
	generated, err := format.Source([]byte(out.String()))
	check(err)
	check(os.MkdirAll(destination, 0o755))
	check(os.WriteFile(filepath.Join(destination, "data.go"), generated, 0o644))
	check(os.WriteFile(filepath.Join(destination, "SOURCE.json"), metadata, 0o644))
	check(os.WriteFile(filepath.Join(destination, "LICENSE"), []byte(texts["LICENSE"]), 0o644))
	fmt.Printf("Unicode %s: %d upper, %d lower, %d cased ranges, %d ignorable ranges; SHA256 %x\n", source.Unicode, len(upper), len(lower), len(ranges["Cased"]), len(ranges["Case_Ignorable"]), sha256.Sum256(generated))
}
