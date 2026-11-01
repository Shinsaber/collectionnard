package connectors

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// editionSearch is what differs between the connectors that list BnF records of a
// kind of edition (video releases of a film, releases of a game, box sets of a
// series); searchEditions is the shared skeleton. Audit Q-004: the same seventy
// lines were copied in films.go, games.go and series.go.
type editionSearch struct {
	// refused is the error when the query carries an ISBN: what the connector searches by.
	refused string
	// typeName is the ItemType of the editions, to match them against the catalogue.
	typeName string
	// byWork turns the Wikidata entity of the work into BnF records and, optionally,
	// the folded titles a record must bear to belong to that work (title searches
	// bring back homonyms).
	byWork func(ctx context.Context, e *wikidata.Entity) (recs []bnf.Record, titles []string, err error)
	// barcodeQuery and titleQuery build the CQL of the other two searches.
	barcodeQuery func(code string) string
	titleQuery   func(title string) (cql string, maxHits int)
	// accept keeps a record of the right kind (and, when titles are set, of the work).
	accept func(r *bnf.Record, titles []string) bool
	// hit describes an accepted record.
	hit func(r *bnf.Record) model.ExternalEditionHit
	// less orders the hits.
	less func(a, b model.ExternalEditionHit) bool
}

// searchEditions lists editions by work (Wikidata QID), barcode or title, keeps
// one hit per ARK, sorts them and marks those already in the catalogue.
func (c *Connector) searchEditions(ctx context.Context, q model.ExternalEditionQuery, s editionSearch) ([]model.ExternalEditionHit, error) {
	if q.ISBN != "" {
		return nil, InputError(s.refused)
	}
	if q.Work != "" && !wikidata.IsQID(q.Work) {
		return nil, InputError("work must be a Wikidata identifier (Q…)")
	}
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var recs []bnf.Record
	var titles []string
	switch {
	case q.Work != "":
		ents, err := c.WD.Entities(ctx, []string{q.Work}, fetchLangs, true)
		if err != nil {
			return nil, err
		}
		e := ents[q.Work]
		if e == nil {
			return nil, notFound("wikidata entity %s", q.Work)
		}
		if recs, titles, err = s.byWork(ctx, e); err != nil {
			return nil, err
		}
	case q.Barcode != "":
		if recs, _, err = c.BnF.Search(ctx, s.barcodeQuery(q.Barcode), maxEditionHits); err != nil {
			return nil, err
		}
	default:
		cql, maxHits := s.titleQuery(q.Title)
		if recs, _, err = c.BnF.Search(ctx, cql, maxHits); err != nil {
			return nil, err
		}
	}
	hits := []model.ExternalEditionHit{}
	var arks []string
	for i := range recs {
		r := &recs[i]
		if r.ARK == "" || slicesContains(arks, r.ARK) || !s.accept(r, titles) {
			continue
		}
		hits = append(hits, s.hit(r))
		arks = append(arks, r.ARK)
	}
	sort.SliceStable(hits, func(i, j int) bool { return s.less(hits[i], hits[j]) })
	existing, err := c.existing(ctx, map[uuid.UUID][]string{d.typeID(s.typeName): arks}, []string{"external_ids", "bnf_ark"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].ID]
	}
	return hits, nil
}

// yearOf is the year of a hit, 0 when unknown.
func yearOf(h model.ExternalEditionHit) int {
	if h.StartYear != nil {
		return *h.StartYear
	}
	return 0
}

// newestFirst orders hits by year, most recent first (DVD and Blu-ray before the VHS).
func newestFirst(a, b model.ExternalEditionHit) bool { return yearOf(a) > yearOf(b) }

// eanCQL searches a record by its EAN.
func eanCQL(code string) string { return "bib.ean all " + bnf.Quote(code) }
