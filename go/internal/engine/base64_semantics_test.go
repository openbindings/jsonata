package gnata_test

import (
	"context"
	"encoding/base64"
	"testing"

	gnata "github.com/openbindings/jsonata-runtime/go/internal/engine"
	"github.com/openbindings/jsonata-runtime/go/internal/engine/internal/jstring"
)

func TestBase64ByteAndUTF8Domains(t *testing.T) {
	encode, err := gnata.Compile(`$base64encode($)`)
	if err != nil {
		t.Fatal(err)
	}
	decode, err := gnata.Compile(`$base64decode($)`)
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 256; n++ {
		v, err := encode.Eval(context.Background(), string(rune(n)))
		want := base64.StdEncoding.EncodeToString([]byte{byte(n)})
		if err != nil || v != want {
			t.Fatalf("byte %x: %#v / %v, want %s", n, v, err, want)
		}
	}
	for _, s := range []string{"", "ASCII", "é", "e\u0301", "😀", "\u0000", "\ufefftext", "漢字"} {
		v, err := decode.Eval(context.Background(), base64.StdEncoding.EncodeToString([]byte(s)))
		if err != nil || v != s {
			t.Fatalf("text %q: %#v / %v", s, v, err)
		}
	}
	for _, s := range []string{" w6k\n", "w6k="} {
		v, err := decode.Eval(context.Background(), s)
		if err != nil || v != "é" {
			t.Fatalf("whitespace/padding: %#v / %v", v, err)
		}
	}
	for _, s := range []string{"Ā", "😀", jstring.CodeUnit(0xd800), jstring.CodeUnit(0xdfff)} {
		if _, err := encode.Eval(context.Background(), s); err == nil {
			t.Fatalf("admitted non-byte %q", s)
		}
	}
	for _, s := range []string{"a", "?", "6Q==", "wK8=", "7aCA", "9JCAgA==", "_w==", "===="} {
		if _, err := decode.Eval(context.Background(), s); err == nil {
			t.Fatalf("admitted malformed base64/UTF-8 %q", s)
		}
	}
}
