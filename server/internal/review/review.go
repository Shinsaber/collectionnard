// Package review holds the rules of a review (I-0, docs/phases/interactions.md):
// the shape of a ReviewType schema, the computed rating, and the fixed limits.
package review

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/shinsaber/colec/internal/privacy"
)

const (
	// MinRatings is the number of notes from which an average is shown.
	MinRatings = 5
	// MaxBody is the longest text of a review, in characters.
	MaxBody = 10000
	// MaxRating is the fixed scale of every criterion and overall rating.
	MaxRating = 10
)

// Languages of a review: those of the interface. To keep aligned with
// web/src/i18n/langs.ts.
var Languages = []string{"fr", "en"}

// ValidLang tells whether lang is a language of the interface.
func ValidLang(lang string) bool { return slices.Contains(Languages, lang) }

// ValidVisibility tells whether v is a visibility of a review.
func ValidVisibility(v string) bool {
	return v == privacy.Anonymous || privacy.Valid(v)
}

// Round keeps one decimal, like the column reviews.rating.
func Round(v float64) float64 { return math.Round(v*10) / 10 }

type criterionSchema struct {
	Type    any      `json:"type"`
	Minimum *float64 `json:"minimum"`
	Maximum *float64 `json:"maximum"`
}

// CheckSchema refuses a ReviewType schema whose criteria are not all integers
// from 0 to 10: the scale stays common and the averages comparable.
func CheckSchema(schema json.RawMessage) error {
	if len(bytes.TrimSpace(schema)) == 0 {
		return nil
	}
	var root struct {
		Properties map[string]criterionSchema `json:"properties"`
	}
	if err := json.Unmarshal(schema, &root); err != nil {
		return errors.New("every criterion must be an integer from 0 to 10")
	}
	for key, c := range root.Properties {
		if c.Type != "integer" || c.Minimum == nil || *c.Minimum != 0 || c.Maximum == nil || *c.Maximum != MaxRating {
			return fmt.Errorf("criterion %q must be an integer from 0 to %d", key, MaxRating)
		}
	}
	return nil
}

// Rate returns the rating of a detailed review: the average of its criteria.
// Every criterion of the schema is required, and nothing else is accepted.
func Rate(schema, criteria json.RawMessage) (float64, error) {
	keys := privacy.Properties(schema)
	if len(keys) == 0 {
		return 0, errors.New("this review type has no criteria")
	}
	var values map[string]any
	dec := json.NewDecoder(bytes.NewReader(criteria))
	dec.UseNumber()
	if err := dec.Decode(&values); err != nil {
		return 0, errors.New("criteria must be an object of integers")
	}
	if len(values) != len(keys) {
		return 0, fmt.Errorf("every criterion is required: %s", strings.Join(keys, ", "))
	}
	sum := 0
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			return 0, fmt.Errorf("every criterion is required: %s", strings.Join(keys, ", "))
		}
		num, isNumber := raw.(json.Number)
		if !isNumber {
			return 0, fmt.Errorf("criterion %q must be an integer from 0 to %d", key, MaxRating)
		}
		v, err := num.Int64()
		if err != nil || v < 0 || v > MaxRating {
			return 0, fmt.Errorf("criterion %q must be an integer from 0 to %d", key, MaxRating)
		}
		sum += int(v)
	}
	return Round(float64(sum) / float64(len(keys))), nil
}

// CheckContextLabel accepts a label per language of the interface, short.
func CheckContextLabel(label map[string]string) error {
	for lang, text := range label {
		if !ValidLang(lang) {
			return fmt.Errorf("unknown language %q", lang)
		}
		if utf8.RuneCountInString(text) > 60 {
			return errors.New("label is too long (max 60 characters)")
		}
	}
	return nil
}

// CheckBody bounds the text of a review.
func CheckBody(body string) error {
	if utf8.RuneCountInString(body) > MaxBody {
		return fmt.Errorf("body is too long (max %d characters)", MaxBody)
	}
	return nil
}
