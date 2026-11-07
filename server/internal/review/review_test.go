package review

import (
	"encoding/json"
	"strings"
	"testing"
)

const schema = `{"type":"object","properties":{
	"story":{"type":"integer","minimum":0,"maximum":10},
	"art":{"type":"integer","minimum":0,"maximum":10}},"additionalProperties":false}`

func TestCheckSchema(t *testing.T) {
	for name, tc := range map[string]struct {
		schema string
		ok     bool
	}{
		"critères corrects":    {schema, true},
		"sans critère":         {`{"type":"object","properties":{}}`, true},
		"vide":                 {``, true},
		"mauvaise échelle":     {`{"type":"object","properties":{"a":{"type":"integer","minimum":0,"maximum":5}}}`, false},
		"sans bornes":          {`{"type":"object","properties":{"a":{"type":"integer"}}}`, false},
		"texte":                {`{"type":"object","properties":{"a":{"type":"string"}}}`, false},
		"nombre à virgule":     {`{"type":"object","properties":{"a":{"type":"number","minimum":0,"maximum":10}}}`, false},
		"propriété mal formée": {`{"type":"object","properties":{"a":true}}`, false},
	} {
		if err := CheckSchema(json.RawMessage(tc.schema)); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestRate(t *testing.T) {
	got, err := Rate(json.RawMessage(schema), json.RawMessage(`{"story":8,"art":7}`))
	if err != nil || got != 7.5 {
		t.Fatalf("Rate = %v, %v", got, err)
	}
	three := `{"type":"object","properties":{"a":{},"b":{},"c":{}}}`
	if got, _ := Rate(json.RawMessage(three), json.RawMessage(`{"a":10,"b":10,"c":9}`)); got != 9.7 {
		t.Errorf("arrondi à une décimale : %v", got)
	}
	for name, criteria := range map[string]string{
		"critère manquant": `{"story":8}`,
		"critère inconnu":  `{"story":8,"art":7,"sound":1}`,
		"remplacé":         `{"story":8,"sound":1}`,
		"hors échelle":     `{"story":11,"art":7}`,
		"négatif":          `{"story":-1,"art":7}`,
		"décimale":         `{"story":7.5,"art":7}`,
		"pas un nombre":    `{"story":"8","art":7}`,
		"pas un objet":     `[8,7]`,
		"vide":             `{}`,
	} {
		if _, err := Rate(json.RawMessage(schema), json.RawMessage(criteria)); err == nil {
			t.Errorf("%s : accepté", name)
		}
	}
	if _, err := Rate(json.RawMessage(`{"type":"object","properties":{}}`), json.RawMessage(`{}`)); err == nil {
		t.Error("un type sans critère refuse le mode détaillé")
	}
}

func TestLimits(t *testing.T) {
	if !ValidLang("fr") || ValidLang("es") || ValidLang("") {
		t.Error("ValidLang")
	}
	for _, v := range []string{"private", "members", "anonymous", "public"} {
		if !ValidVisibility(v) {
			t.Errorf("%s refusé", v)
		}
	}
	if ValidVisibility("rating_only") || ValidVisibility("") {
		t.Error("ValidVisibility")
	}
	if CheckContextLabel(map[string]string{"fr": "Joué sur", "en": "Played on"}) != nil ||
		CheckContextLabel(map[string]string{"es": "Jugado en"}) == nil ||
		CheckContextLabel(map[string]string{"fr": strings.Repeat("a", 61)}) == nil {
		t.Error("CheckContextLabel")
	}
	if CheckBody(strings.Repeat("é", MaxBody)) != nil || CheckBody(strings.Repeat("é", MaxBody+1)) == nil {
		t.Error("CheckBody compte les caractères")
	}
}
