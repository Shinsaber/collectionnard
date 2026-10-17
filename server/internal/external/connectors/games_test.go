package connectors_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

// Réponses réelles enregistrées le 2026-10-17 (testdata/game_*, wikidata_*_games*.json).
const (
	mk8Deluxe   = "Q28321447"
	mk8         = "Q13427106"
	nintendo    = "Q8093"
	switchQID   = "Q19610114"
	mk8Switch   = "ark:/12148/cb453824912" // notice complète : « 1 cartouche (Nintendo Switch) »
	mk8Brief    = "ark:/12148/cb488158895" // notice brève : EAN, « 1 cartouche informatique », sans plateforme
	mk8WiiU     = "ark:/12148/cb43877942t" // « 1 disque optique numérique (Wii U) », EAN sans son zéro initial
	mk8WiiUCode = "045496333133"
)

func TestGamesSearch(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.GamesConnector(connectorstest.NewCatalog(t))
	if conn.ID() != "games" || conn.Domain() != "games" || conn.EditionCode() != "barcode" || !conn.EditionsByWork() {
		t.Errorf("id %s, domain %s", conn.ID(), conn.Domain())
	}
	ctx := context.Background()
	hits, err := conn.SearchWorks(ctx, "Mario Kart 8 Deluxe")
	if err != nil || len(hits) != 2 || hits[0].ID != mk8Deluxe || hits[0].ItemTypeID != connectorstest.TypeID(t, "Game") ||
		hits[0].StartYear == nil || *hits[0].StartYear != 2017 {
		t.Fatalf("works: %v %+v", err, hits)
	}

	// Éditions du jeu : titrées comme lui (pas le Mario Kart 8 de la Wii U), livres écartés.
	eds, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Work: mk8Deluxe})
	if err != nil {
		t.Fatal(err)
	}
	formats := map[string]string{}
	for _, h := range eds {
		formats[h.ID] = h.Format
	}
	if formats[mk8Switch] != "Cartouche · Nintendo Switch" || formats[mk8Brief] != "Cartouche" || formats[mk8WiiU] != "" {
		t.Errorf("editions: %v", formats)
	}

	byEAN, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Barcode: mk8WiiUCode})
	if err != nil || len(byEAN) != 1 || byEAN[0].ID != mk8WiiU || byEAN[0].Barcode != mk8WiiUCode || byEAN[0].Format != "Disque · Wii U" {
		t.Errorf("by EAN (zero restored): %v %+v", err, byEAN)
	}
	var input external.InputError
	if _, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{ISBN: "9782723433358"}); !errors.As(err, &input) {
		t.Errorf("ISBN: %v", err)
	}
}

func TestGamesPreview(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.GamesConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Work: mk8Deluxe, Editions: []string{mk8Switch, mk8Brief}})
	if err != nil {
		t.Fatal(err)
	}
	game := nodeByKey(t, plan, "wikidata:"+mk8Deluxe)
	a := attrs(t, game.Attributes)
	if jsonPath(a, "title", "fr") != "Mario Kart 8 Deluxe" || a["release_year"] != 2017.0 || a["pegi"] != "3" {
		t.Errorf("game: %v", a)
	}
	if m, _ := a["game_modes"].([]any); !slices.Contains(m, any("single_player")) || !slices.Contains(m, any("multiplayer")) {
		t.Errorf("modes: %v", a["game_modes"])
	}

	// La plateforme est rattachée à son constructeur ; le jeu y est.
	sw := nodeByKey(t, plan, "wikidata:"+switchQID)
	if attrs(t, sw.Attributes)["name"] != "Nintendo Switch" || relation(plan, game.Key, sw.Key) == nil {
		t.Errorf("platform: %+v", sw)
	}
	if r := relation(plan, sw.Key, "wikidata:"+nintendo); r == nil || !slices.Equal(jsonValue(t, r.Attributes, "roles").([]any), []any{"manufacturer"}) {
		t.Errorf("manufacturer: %+v", r)
	}
	if r := relation(plan, game.Key, "wikidata:"+nintendo); r == nil || !slices.Contains(jsonValue(t, r.Attributes, "roles").([]any), any("publisher")) {
		t.Errorf("publisher of the game: %+v", r)
	}

	// Cartouche Switch : plateforme lue dans la notice ; « Nintendo co. » = le Nintendo de Wikidata.
	cart := nodeByKey(t, plan, "bnf:cb453824912")
	ca := attrs(t, cart.Attributes)
	if ca["format"] != "cartridge" || ca["release_year"] != 2017.0 || relation(plan, cart.Key, sw.Key) == nil {
		t.Errorf("cartridge: %v", ca)
	}
	if relation(plan, cart.Key, "wikidata:"+nintendo) == nil {
		t.Error("BnF company not merged with the Wikidata one")
	}
	for _, n := range plan.Nodes {
		if n.Key == "publisher:nintendo co." {
			t.Error("duplicate company « Nintendo co. »")
		}
	}
	// Notice brève sans plateforme : celle, unique, du jeu ; son EAN.
	brief := nodeByKey(t, plan, "bnf:cb488158895")
	if ba := attrs(t, brief.Attributes); ba["ean"] != "045496420246" || relation(plan, brief.Key, sw.Key) == nil {
		t.Errorf("brief record: %v", ba)
	}

	its, rts := connectorstest.Domain(t)
	for _, n := range plan.Nodes {
		i := slices.IndexFunc(its, func(it model.ItemType) bool { return it.ID == n.ItemTypeID })
		if err := validator.ValidateAttributes(its[i].AttributesSchema, n.Attributes); err != nil {
			t.Errorf("node %s invalid: %v", n.Key, err)
		}
	}
	for _, r := range plan.Relations {
		i := slices.IndexFunc(rts, func(rt model.RelationType) bool { return rt.ID == r.RelationTypeID })
		if err := validator.ValidateAttributes(rts[i].AttributesSchema, r.Attributes); err != nil {
			t.Errorf("relation %s → %s invalid: %v", r.From, r.To, err)
		}
	}
}

// A release chosen alone finds its game (Wikidata, same title) and its platform.
func TestGamesReleaseAlone(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.GamesConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Editions: []string{mk8WiiU}})
	if err != nil {
		t.Fatal(err)
	}
	disc := nodeByKey(t, plan, "bnf:cb43877942t")
	if disc.Parent != "wikidata:"+mk8 || attrs(t, disc.Attributes)["ean"] != mk8WiiUCode || attrs(t, disc.Attributes)["format"] != "optical_disc" {
		t.Errorf("disc: %+v", disc)
	}
	if relation(plan, disc.Key, "wikidata:Q56942") == nil {
		t.Error("Wii U platform of the disc missing")
	}
}
