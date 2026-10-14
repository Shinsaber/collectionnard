package bnf

import "testing"

func TestARK(t *testing.T) {
	// Check characters of real catalogue records.
	for number, ark := range map[string]string{
		"34338439": "ark:/12148/cb343384394",
		"12336978": "ark:/12148/cb12336978f",
		"14534411": "ark:/12148/cb14534411x",
		"35600400": "ark:/12148/cb356004006",
		"44353554": "ark:/12148/cb44353554m",
	} {
		if got := ARK(number); got != ark {
			t.Errorf("ARK(%s) = %s, want %s", number, got, ark)
		}
		if n, ok := RecordNumber(ark); !ok || n != number {
			t.Errorf("RecordNumber(%s) = %s, %v", ark, n, ok)
		}
	}
	for _, bad := range []string{"", "1234", "abcdefgh", "3433843x"} {
		if got := ARK(bad); got != "" {
			t.Errorf("ARK(%q) = %q, want empty", bad, got)
		}
	}
	for _, bad := range []string{"ark:/12148/cb34338439x", "ark:/12148/cb3433843", "ark:/99999/cb343384394", "cb343384394"} {
		if _, ok := RecordNumber(bad); ok {
			t.Errorf("RecordNumber(%q) accepted", bad)
		}
	}
}

func TestNormalizeISBN(t *testing.T) {
	for in, want := range map[string]string{
		"978-2-87695-205-8": "9782876952058",
		"9782876952058":     "9782876952058",
		"2-8769-5205-X":     "9782876952058", // ISBN-10 → 13
		"287695205x":        "9782876952058",
		"9782876952059":     "", // checksum
		"2-8769-5205-1":     "",
		"1234":              "",
		"9772876952058":     "", // EAN of a periodical, not a book
	} {
		if got := NormalizeISBN(in); got != want {
			t.Errorf("NormalizeISBN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValues(t *testing.T) {
	for in, want := range map[string]int{
		"1 vol. (190 p.)":              190,
		"[90] p.":                      90,
		"Non paginé [ca 182 p.]":       182,
		"1 vol. (297 p.-[2] p. dépl.)": 297,
		"207 p.":                       207,
	} {
		if got, ok := PageCount(in); !ok || got != want {
			t.Errorf("PageCount(%q) = %d, %v", in, got, ok)
		}
	}
	if _, ok := PageCount("1 vol."); ok {
		t.Error("PageCount without pages")
	}
	if p, ok := PriceEUR("6,90 EUR"); !ok || p != 6.9 {
		t.Errorf("PriceEUR = %v, %v", p, ok)
	}
	for _, s := range []string{"38 F", "42 F", "impr. 2009", ""} {
		if _, ok := PriceEUR(s); ok {
			t.Errorf("PriceEUR(%q) accepted", s)
		}
	}
	if s, e := Years("19931005g1993    m  y0frey50      ba"); s != 1993 || e != 0 {
		t.Errorf("Years = %d, %d", s, e)
	}
	if s, e := Years("20131114a19..2010m  y0frey50      ba"); s != 0 || e != 2010 {
		t.Errorf("Years = %d, %d", s, e)
	}
	if n, ok := FirstNumber("T. 12"); !ok || n != 12 {
		t.Errorf("FirstNumber = %d, %v", n, ok)
	}
	if got := Quote(`one "piece"* \x`); got != `"one piece x"` {
		t.Errorf("Quote = %s", got)
	}
}
