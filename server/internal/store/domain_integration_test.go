//go:build integration

package store_test

import (
	"encoding/json"
	"net/url"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/dbmigrate"
	"github.com/shinsaber/colec/internal/domains"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// freshStore creates an empty migrated database (dropped at the end of the test).
func freshStore(t *testing.T, name string) *store.PgStore {
	t.Helper()
	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(baseURL)
	u.Path = "/" + name
	if err := dbmigrate.Migrate(ctx, u.String(), migrationsDir); err != nil {
		t.Fatal(err)
	}
	st, err := store.NewPgStore(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		st.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	return st
}

func outcomes(rep *model.DomainInstallReport) map[string]int {
	out := map[string]int{}
	for _, e := range rep.Entries {
		out[e.Outcome]++
	}
	return out
}

func TestInstallDomain(t *testing.T) {
	st := freshStore(t, "colec_test_domains")
	books, err := domains.Get("books")
	if err != nil || books == nil {
		t.Fatal(err)
	}
	comic := uuid.MustParse("b0000000-0000-0000-0000-000000000013")

	// Simulation : rien d'écrit.
	rep, err := st.InstallDomain(ctx, books, true)
	if err != nil || rep.Applied || len(rep.Errors) != 0 {
		t.Fatalf("dry run: %v %+v", err, rep)
	}
	if _, err := st.GetItemType(ctx, comic); err == nil {
		t.Fatal("dry run wrote the types")
	}

	// Première installation (tv absent : adapted_as laissée de côté).
	rep, err = st.InstallDomain(ctx, books, false)
	if err != nil || !rep.Applied || len(rep.Errors) != 0 {
		t.Fatalf("install: %v %+v", err, rep)
	}
	if o := outcomes(rep); o[model.DomainCreated] != 15 || o[model.DomainSkipped] != 1 {
		t.Errorf("first install: %v", o)
	}
	// Réinstallation : rien à changer.
	if rep, _ = st.InstallDomain(ctx, books, false); outcomes(rep)[model.DomainUnchanged] != 15 {
		t.Errorf("reinstall: %v", outcomes(rep))
	}
	// tv installé ensuite : adapted_as est ajoutée au domaine books.
	if _, err := st.InstallDomains(ctx, "tv"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetRelationType(ctx, uuid.MustParse("c0000000-0000-0000-0000-000000000016")); err != nil {
		t.Errorf("adapted_as after tv: %v", err)
	}

	// Modifications locales : renommage et champ ajouté.
	ct, _ := st.GetItemType(ctx, comic)
	var schema map[string]any
	_ = json.Unmarshal(ct.AttributesSchema, &schema)
	schema["properties"].(map[string]any)["shelf"] = map[string]any{"type": "string"}
	raw, _ := json.Marshal(schema)
	name := "bd"
	if _, err := st.UpdateItemType(ctx, comic, model.ItemTypeUpdate{Name: &name, AttributesSchema: raw}); err != nil {
		t.Fatal(err)
	}
	list, _ := st.ListDomains(ctx)
	for _, d := range list {
		if d.ID == "books" && (d.Status != model.DomainUpToDate || !d.ModifiedLocally) {
			t.Errorf("books state: %+v", d)
		}
		if d.ID == "collections" && d.Status != model.DomainNotInstalled {
			t.Errorf("collections state: %+v", d)
		}
	}

	// Version 2 du paquet : un champ ajouté à Comic, le nom changé aussi.
	v2 := *books
	v2.PackageVersion = books.PackageVersion + 1
	v2.ItemTypes = slices.Clone(books.ItemTypes)
	for i, it := range v2.ItemTypes {
		if it.ID != comic {
			continue
		}
		var s map[string]any
		_ = json.Unmarshal(it.AttributesSchema, &s)
		s["properties"].(map[string]any)["isbn"] = map[string]any{"type": "string"}
		v2.ItemTypes[i].AttributesSchema, _ = json.Marshal(s)
		v2.ItemTypes[i].Name = "Comics"
	}
	rep, err = st.InstallDomain(ctx, &v2, false)
	if err != nil || !rep.Applied {
		t.Fatalf("v2: %v %+v", err, rep)
	}
	for _, e := range rep.Entries {
		if e.ID == comic && (e.Outcome != model.DomainMerged || !slices.Equal(e.Conflicts, []string{"name"})) {
			t.Errorf("Comic v2: %+v", e)
		}
	}
	got, _ := st.GetItemType(ctx, comic)
	var merged map[string]any
	_ = json.Unmarshal(got.AttributesSchema, &merged)
	props := merged["properties"].(map[string]any)
	if got.Name != "bd" || props["shelf"] == nil || props["isbn"] == nil || props["title"] == nil {
		t.Errorf("merged Comic: name %q, properties %v", got.Name, props)
	}
	if rep.FromVersion == nil || *rep.FromVersion != books.PackageVersion || rep.ToVersion != books.PackageVersion+1 {
		t.Errorf("versions: %v → %d", rep.FromVersion, rep.ToVersion)
	}
}
