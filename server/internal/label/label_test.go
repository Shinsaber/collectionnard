package label

import (
	"encoding/json"
	"testing"
)

func TestRender(t *testing.T) {
	cases := []struct {
		name, schema, attrs, lang, want string
	}{
		{"template simple", `{"x-label":"{title}"}`, `{"title":"Breaking Bad"}`, "fr", "Breaking Bad"},
		{"template composé", `{"x-label":"{last_name} {first_name}"}`, `{"first_name":"Akira","last_name":"Toriyama"}`, "fr", "Toriyama Akira"},
		{"template avec texte et nombre", `{"x-label":"Tome {number} — {title}"}`, `{"number":12,"title":"Freezer"}`, "fr", "Tome 12 — Freezer"},
		{"placeholder manquant → espaces réduits", `{"x-label":"{number}. {title}"}`, `{"title":"Pilot"}`, "fr", ". Pilot"},
		{"i18n selon la langue", `{"x-label":"{title}"}`, `{"title":{"fr":"Le Pilote","en":"Pilot"}}`, "en", "Pilot"},
		{"i18n repli fr", `{"x-label":"{title}"}`, `{"title":{"fr":"Le Pilote"}}`, "de", "Le Pilote"},
		{"fallback anglais", `{"type":"object"}`, `{"name":"AMC"}`, "fr", "AMC"},
		{"fallback français historique", `{}`, `{"titre":"Breaking Bad"}`, "fr", "Breaking Bad"},
		{"fallback nombre", ``, `{"numero":3}`, "fr", "3"},
		{"rien d'utilisable", `{}`, `{"foo":"bar"}`, "fr", ""},
		{"attributs invalides", `{}`, `not json`, "fr", ""},
		{"tableau", `{"x-label":"{roles}"}`, `{"roles":["writer","artist"]}`, "fr", "writer, artist"},
		{"nombre décimal", `{"x-label":"{price} €"}`, `{"price":12.5}`, "fr", "12.5 €"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Render(json.RawMessage(tc.schema), json.RawMessage(tc.attrs), tc.lang)
			if got != tc.want {
				t.Errorf("Render() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTemplate(t *testing.T) {
	if got := Template(json.RawMessage(`{"x-label":"  {title} "}`)); got != "{title}" {
		t.Errorf("Template() = %q", got)
	}
	if got := Template(json.RawMessage(`{"type":"object"}`)); got != "" {
		t.Errorf("Template() = %q, want empty", got)
	}
	if got := Template(nil); got != "" {
		t.Errorf("Template(nil) = %q", got)
	}
}
