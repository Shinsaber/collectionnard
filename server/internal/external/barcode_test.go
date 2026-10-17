package external

import "testing"

func TestNormalizeBarcode(t *testing.T) {
	for in, want := range map[string]string{
		"634904078126":    "634904078126",  // UPC-A (OK Computer, XL 2016)
		"0634904086817":   "0634904086817", // EAN-13
		"4 988006 729254": "4988006729254",
		"96385074":        "96385074", // EAN-8
		"9782723433358":   "9782723433358",
		"634904078127":    "", // wrong check digit
		"12345":           "",
		"63490407812X":    "",
	} {
		if got := NormalizeBarcode(in); got != want {
			t.Errorf("NormalizeBarcode(%q) = %q, want %q", in, got, want)
		}
	}
}
