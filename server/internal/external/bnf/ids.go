package bnf

import (
	"regexp"
	"strconv"
	"strings"
)

// arkPrefix starts every ARK of the BnF catalogue (NAAN 12148, "cb" records).
const arkPrefix = "ark:/12148/cb"

// noidDigits is the NOID alphabet used by the ARK check character.
const noidDigits = "0123456789bcdfghjkmnpqrstvwxz"

var arkRE = regexp.MustCompile(`^ark:/12148/cb([0-9]{8,9})([0-9bcdfghjkmnpqrstvwxz])$`)

// checkChar computes the NOID check character of "cb" + record number.
func checkChar(name string) byte {
	sum := 0
	for i, c := range name {
		if p := strings.IndexRune(noidDigits, c); p >= 0 {
			sum += (i + 1) * p
		}
	}
	return noidDigits[sum%len(noidDigits)]
}

// ARK returns the ARK of a catalogue record number ("34338439", as found in
// the $0 / $3 link subfields), or "" when it is not a record number.
func ARK(recordNumber string) string {
	recordNumber = strings.TrimSpace(recordNumber)
	if len(recordNumber) < 8 || len(recordNumber) > 9 {
		return ""
	}
	if _, err := strconv.Atoi(recordNumber); err != nil {
		return ""
	}
	name := "cb" + recordNumber
	return arkPrefix + recordNumber + string(checkChar(name))
}

// RecordNumber returns the record number of a catalogue ARK after checking its
// check character ("ark:/12148/cb343384394" → "34338439").
func RecordNumber(ark string) (string, bool) {
	m := arkRE.FindStringSubmatch(ark)
	if m == nil || checkChar("cb"+m[1]) != m[2][0] {
		return "", false
	}
	return m[1], true
}

// ShortARK is the ARK without its "ark:/12148/" prefix, used in node keys.
func ShortARK(ark string) string { return strings.TrimPrefix(ark, "ark:/12148/") }

// RecordURL is the public page of a record.
func RecordURL(ark string) string { return "https://catalogue.bnf.fr/" + ark }

// ── ISBN ────────────────────────────────────────────────────────────

// NormalizeISBN returns the ISBN-13 (digits only) of an ISBN-10, ISBN-13 or EAN,
// with or without hyphens, or "" when the checksum is wrong.
func NormalizeISBN(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == 'X' || r == 'x':
			b.WriteRune('X')
		}
	}
	d := b.String()
	switch len(d) {
	case 13:
		if (strings.HasPrefix(d, "978") || strings.HasPrefix(d, "979")) && !strings.Contains(d, "X") &&
			isbn13Check(d[:12]) == d[12] {
			return d
		}
	case 10:
		if isbn10Valid(d) {
			core := "978" + d[:9]
			return core + string(isbn13Check(core))
		}
	}
	return ""
}

func isbn13Check(twelve string) byte {
	sum := 0
	for i, c := range twelve {
		n := int(c - '0')
		if i%2 == 1 {
			n *= 3
		}
		sum += n
	}
	return byte('0' + (10-sum%10)%10)
}

func isbn10Valid(d string) bool {
	sum := 0
	for i, c := range d {
		var n int
		switch {
		case c == 'X' && i == 9:
			n = 10
		case c >= '0' && c <= '9':
			n = int(c - '0')
		default:
			return false
		}
		sum += (10 - i) * n
	}
	return sum%11 == 0
}

// ── Values ──────────────────────────────────────────────────────────

var (
	pagesRE  = regexp.MustCompile(`\[?(?:ca )?(\d+)\]?\s*p\.`)
	priceRE  = regexp.MustCompile(`^(\d+(?:[.,]\d{1,2})?)\s*(?:EUR|€)`)
	numberRE = regexp.MustCompile(`\d+`)
	yearRE   = regexp.MustCompile(`^[12][0-9]{3}$`)
)

// PageCount reads the number of pages of a 215$a extent ("1 vol. (190 p.)",
// "[90] p.", "Non paginé [ca 182 p.]").
func PageCount(extent string) (int, bool) {
	m := pagesRE.FindStringSubmatch(extent)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil && n > 0
}

// PriceEUR reads a 010$d price in euros ("6,90 EUR"); prices in francs are ignored.
func PriceEUR(s string) (float64, bool) {
	m := priceRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
	return f, err == nil
}

// FirstNumber returns the first integer of a volume designation ("12", "T. 3").
func FirstNumber(s string) (int, bool) {
	m := numberRE.FindString(s)
	if m == "" {
		return 0, false
	}
	n, err := strconv.Atoi(m)
	return n, err == nil
}

// Years reads the publication dates of the coded field 100$a (positions 9-12
// and 13-16); blanks, "9999" and unknown digits ("199u") are skipped.
func Years(coded string) (start, end int) {
	year := func(from int) int {
		if len(coded) < from+4 {
			return 0
		}
		s := coded[from : from+4]
		if !yearRE.MatchString(s) || s == "9999" {
			return 0
		}
		y, _ := strconv.Atoi(s)
		return y
	}
	return year(9), year(13)
}

// Language maps a UNIMARC language code (ISO 639-2/B) to its ISO 639-1 code.
func Language(code string) string {
	switch code {
	case "fre":
		return "fr"
	case "jpn":
		return "ja"
	case "eng":
		return "en"
	case "kor":
		return "ko"
	case "chi":
		return "zh"
	case "spa":
		return "es"
	case "ger":
		return "de"
	case "ita":
		return "it"
	case "dut":
		return "nl"
	case "por":
		return "pt"
	}
	return ""
}
