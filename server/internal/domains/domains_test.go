package domains

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

func js(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestMerge3(t *testing.T) {
	base := `{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}},
		"format":{"enum":["poche","album"]}},"required":["title"],"x-order":["title","format"]}`
	cases := []struct {
		name, local, next, want string
		conflicts               []string
	}{
		{"local inchangé : nouvelle version", base,
			`{"name":"Edition","properties":{"title":{"type":"string"}}}`,
			`{"name":"Edition","properties":{"title":{"type":"string"}}}`, nil},
		{"paquet inchangé : version locale",
			`{"name":"Édition locale","properties":{}}`, base,
			`{"name":"Édition locale","properties":{}}`, nil},
		{"ajouts du paquet + modifications locales gardées",
			`{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Intitulé"}},
				"format":{"enum":["poche","album","omnibus"]},"note":{"type":"string"}},"required":["title"],"x-order":["title","note","format"]}`,
			`{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Titre","en":"Title"}},
				"format":{"enum":["poche","album","hardcover"]},"isbn":{"type":"string"}},"required":["title"],"x-order":["title","format","isbn"]}`,
			`{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Intitulé","en":"Title"}},
				"format":{"enum":["poche","album","omnibus","hardcover"]},"note":{"type":"string"},"isbn":{"type":"string"}},
				"required":["title"],"x-order":["title","note","format","isbn"]}`, nil},
		{"supprimé localement : pas réintroduit ; supprimé du paquet : gardé",
			`{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}}},"required":["title"],"x-order":["title"]}`,
			`{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}},"format":{"enum":["poche","album"]}},"x-order":["title","format"]}`,
			`{"name":"Edition","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}}},"required":["title"],"x-order":["title"]}`, nil},
		{"même valeur modifiée des deux côtés : locale gardée, signalée",
			`{"name":"BD","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}},"format":{"enum":["poche","album"]}},"required":["title"],"x-order":["title","format"]}`,
			`{"name":"Comics","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}},"format":{"enum":["poche","album"]}},"required":["title"],"x-order":["title","format"]}`,
			`{"name":"BD","properties":{"title":{"type":"string","x-labels":{"fr":"Titre"}},"format":{"enum":["poche","album"]}},"required":["title"],"x-order":["title","format"]}`,
			[]string{"name"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, conflicts := Merge3(js(t, base), js(t, c.local), js(t, c.next))
			if !reflect.DeepEqual(got, js(t, c.want)) {
				g, _ := json.Marshal(got)
				t.Errorf("got  %s\nwant %s", g, c.want)
			}
			if !reflect.DeepEqual(conflicts, c.conflicts) {
				t.Errorf("conflicts = %v, want %v", conflicts, c.conflicts)
			}
		})
	}

	// Sans copie de référence (type venu d'un seed) : fusion additive.
	got, conflicts := Merge3(nil,
		js(t, `{"name":"BD","properties":{"a":{"type":"string"}},"required":["a"]}`),
		js(t, `{"name":"Comic","properties":{"a":{"type":"string"},"b":{"type":"integer"}},"required":["a"]}`))
	want := js(t, `{"name":"BD","properties":{"a":{"type":"string"},"b":{"type":"integer"}},"required":["a"]}`)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(conflicts, []string{"name"}) {
		t.Errorf("no base: %v %v", got, conflicts)
	}
}

func TestPackages(t *testing.T) {
	all, err := All()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	seen := map[uuid.UUID]string{}
	for _, p := range all {
		ids[p.ID] = true
		if p.Title["fr"] == "" || p.Title["en"] == "" {
			t.Errorf("%s: title fr/en required", p.ID)
		}
		check := func(id uuid.UUID, name string) {
			if other, dup := seen[id]; dup {
				t.Errorf("%s: id %s already used by %s", p.ID, id, other)
			}
			seen[id] = p.ID + "/" + name
		}
		for _, it := range p.ItemTypes {
			check(it.ID, it.Name)
			if err := validator.CheckSchema(it.AttributesSchema); err != nil {
				t.Errorf("%s/%s: %v", p.ID, it.Name, err)
			}
		}
		for _, ct := range p.CollectionTypes {
			check(ct.ID, ct.Name)
		}
		for _, rt := range p.RelationTypes {
			check(rt.ID, rt.Name)
			if err := validator.CheckSchema(rt.AttributesSchema); err != nil {
				t.Errorf("%s/%s: %v", p.ID, rt.Name, err)
			}
			if len(rt.SourceItemTypeIDs) == 0 || len(rt.TargetItemTypeIDs) == 0 {
				t.Errorf("%s/%s: sources and targets required", p.ID, rt.Name)
			}
		}
	}
	for _, id := range []string{"books", "collections", "music", "people", "tv", "video"} {
		if !ids[id] {
			t.Errorf("package %s missing", id)
		}
	}
}

func TestPlanInstall(t *testing.T) {
	books, err := Get("books")
	if err != nil || books == nil {
		t.Fatal(err)
	}
	series := uuid.MustParse("b0000000-0000-0000-0000-000000000001") // tv

	// Personnes et éditeurs : paquet « people », installé avant books (Requires).
	person, publisher := uuid.MustParse("b0000000-0000-0000-0000-000000000016"), uuid.MustParse("b0000000-0000-0000-0000-000000000017")

	// Instance sans livres : tout est créé, sauf adapted_as dont la seule cible, Series (tv), manque.
	plan, err := PlanInstall(books, State{Local: map[uuid.UUID]Snapshot{}, Base: map[uuid.UUID]Snapshot{}, ItemTypes: map[uuid.UUID]bool{person: true, publisher: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range plan.Entries {
		if e.Name == "adapted_as" {
			if e.Outcome != model.DomainSkipped || len(e.MissingTypes) != 1 || e.MissingTypes[0] != series {
				t.Errorf("adapted_as: %s, missing %v", e.Outcome, e.MissingTypes)
			}
			continue
		}
		if e.Outcome != model.DomainCreated {
			t.Errorf("%s: %s", e.Name, e.Outcome)
		}
	}
	if len(plan.Catalog.ItemTypes) != len(books.ItemTypes) || len(plan.Catalog.RelationTypes) != len(books.RelationTypes)-1 {
		t.Errorf("catalog: %d item types, %d relation types", len(plan.Catalog.ItemTypes), len(plan.Catalog.RelationTypes))
	}

	// Réinstallation à l'identique (tv présent) : rien à écrire.
	local := map[uuid.UUID]Snapshot{}
	for id, s := range plan.Bases {
		local[id] = s
	}
	again, err := PlanInstall(books, State{Local: local, Base: plan.Bases, ItemTypes: map[uuid.UUID]bool{series: true, person: true, publisher: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range again.Entries {
		if e.Name == "adapted_as" {
			if e.Outcome != model.DomainCreated {
				t.Errorf("adapted_as: %s (cible Series désormais présente)", e.Outcome)
			}
			continue
		}
		if e.Outcome != model.DomainUnchanged {
			t.Errorf("%s: %s, want unchanged", e.Name, e.Outcome)
		}
	}

	// Type supprimé localement depuis l'installation : pas recréé.
	comic := uuid.MustParse("b0000000-0000-0000-0000-000000000013")
	delete(local, comic)
	skipped, _ := PlanInstall(books, State{Local: local, Base: plan.Bases, ItemTypes: map[uuid.UUID]bool{series: true, person: true, publisher: true}})
	for _, e := range skipped.Entries {
		if e.ID == comic && e.Outcome != model.DomainSkipped {
			t.Errorf("deleted Comic: %s", e.Outcome)
		}
	}
}

// Every relation has a source within its package and the packages it requires;
// a target outside them must belong to another package (optional cross-domain
// end, such as adapted_as → Series).
func TestRequirements(t *testing.T) {
	all, err := All()
	if err != nil {
		t.Fatal(err)
	}
	anywhere := map[uuid.UUID]bool{}
	for _, p := range all {
		for _, it := range p.ItemTypes {
			anywhere[it.ID] = true
		}
	}
	for _, p := range all {
		reqs, err := Requirements(p)
		if err != nil {
			t.Fatal(err)
		}
		own := map[uuid.UUID]bool{}
		for _, q := range append(reqs, p) {
			for _, it := range q.ItemTypes {
				own[it.ID] = true
			}
		}
		for _, rt := range p.RelationTypes {
			if !slices.ContainsFunc(rt.SourceItemTypeIDs, func(id uuid.UUID) bool { return own[id] }) {
				t.Errorf("%s/%s: no source within the package and its requirements %v", p.ID, rt.Name, p.Requires)
			}
			for _, id := range append(slices.Clone(rt.SourceItemTypeIDs), rt.TargetItemTypeIDs...) {
				if !anywhere[id] {
					t.Errorf("%s/%s: item type %s in no package", p.ID, rt.Name, id)
				}
			}
			if !slices.ContainsFunc(rt.TargetItemTypeIDs, func(id uuid.UUID) bool { return own[id] }) && rt.Name != "adapted_as" {
				t.Errorf("%s/%s: no target within the package and its requirements %v", p.ID, rt.Name, p.Requires)
			}
		}
	}
	books, _ := Get("books")
	if reqs, _ := Requirements(books); len(reqs) != 1 || reqs[0].ID != "people" {
		t.Errorf("books requires %v", reqs)
	}
	loop := &model.DomainPackage{ID: "x", Requires: []string{"nope"}}
	if _, err := Requirements(loop); err == nil {
		t.Error("unknown requirement accepted")
	}
}
