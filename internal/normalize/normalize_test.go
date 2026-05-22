package normalize

import "testing"

func TestFoldLowercases(t *testing.T) {
	if got := Fold("README"); got != "readme" {
		t.Fatalf("Fold(README) = %q, want %q", got, "readme")
	}
}

func TestSmartCaseQueryAllLower(t *testing.T) {
	lookup, scoreQ, cs := SmartCaseQuery("foo")
	if lookup != "foo" || scoreQ != "foo" || cs {
		t.Fatalf("SmartCaseQuery(foo) = (%q,%q,%v), want (foo,foo,false)", lookup, scoreQ, cs)
	}
}

func TestSmartCaseQueryMixedCase(t *testing.T) {
	lookup, scoreQ, cs := SmartCaseQuery("Foo")
	if lookup != "foo" || scoreQ != "Foo" || !cs {
		t.Fatalf("SmartCaseQuery(Foo) = (%q,%q,%v), want (foo,Foo,true)", lookup, scoreQ, cs)
	}
}

func TestNFCComposedVsDecomposed(t *testing.T) {
	// "é" composed: U+00E9
	composed := "café"
	// "é" decomposed: "e" + U+0301 (combining acute)
	decomposed := "café"
	if Fold(composed) != Fold(decomposed) {
		t.Fatalf("expected NFC fold to equalize composed/decomposed: %q vs %q",
			Fold(composed), Fold(decomposed))
	}
}
