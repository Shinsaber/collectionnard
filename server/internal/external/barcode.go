package external

import "strings"

// NormalizeBarcode returns the digits of a barcode (EAN-8, UPC-A, EAN-13 or
// GTIN-14, spaces and dashes ignored) when its check digit is valid, else "".
func NormalizeBarcode(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-':
		default:
			return ""
		}
	}
	d := b.String()
	switch len(d) {
	case 8, 12, 13, 14:
	default:
		return ""
	}
	// GTIN check digit: from the right, weights 3, 1, 3… on the digits before it.
	sum := 0
	for i := len(d) - 2; i >= 0; i-- {
		n := int(d[i] - '0')
		if (len(d)-2-i)%2 == 0 {
			n *= 3
		}
		sum += n
	}
	if byte('0'+(10-sum%10)%10) != d[len(d)-1] {
		return ""
	}
	return d
}
