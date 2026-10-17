package bdmanga

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// SearchWorks searches manga and comic works on Wikidata by title.
func (c *Connector) SearchWorks(ctx context.Context, text string) ([]model.ExternalWorkHit, error) {
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	classes := workClasses
	if c.profile == profileNovels {
		classes = novelClasses
	}
	ids, err := c.WD.Search(ctx, text, classes, maxWorkHits)
	if err != nil {
		return nil, err
	}
	ents, err := c.WD.Entities(ctx, ids, fetchLangs, true)
	if err != nil {
		return nil, err
	}
	hits := []model.ExternalWorkHit{}
	byType := map[uuid.UUID][]string{}
	for _, id := range ids {
		e := ents[id]
		if e == nil {
			continue
		}
		typeName := c.workType(e)
		if typeName == "" {
			continue
		}
		h := model.ExternalWorkHit{
			WikidataID:  id,
			Label:       e.Label("fr", "en", "mul", "ja"),
			Description: e.Descriptions["fr"],
			ItemTypeID:  d.typeID(typeName),
			SourceURL:   wikidata.EntityURL(id),
		}
		if h.Description == "" {
			h.Description = e.Descriptions["en"]
		}
		yearProps := []string{pStart, pInception, pPublication}
		if c.profile == profileNovels {
			yearProps = []string{pPublication, pInception}
		}
		if y, ok := e.Year(yearProps...); ok {
			h.StartYear = &y
		}
		hits = append(hits, h)
		byType[h.ItemTypeID] = append(byType[h.ItemTypeID], id)
	}
	existing, err := c.existing(ctx, byType, []string{"external_ids", "wikidata"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].WikidataID]
	}
	return hits, nil
}

// SearchEditions searches French editions on the BnF, by title or by the ISBN
// of one of their volumes (multi-volume sets ; novels: see searchNovelEditions).
func (c *Connector) SearchEditions(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error) {
	if c.profile == profileNovels {
		return c.searchNovelEditions(ctx, q)
	}
	title, isbn := q.Title, q.ISBN
	if q.Work != "" {
		return nil, InputError("this connector searches editions by title or ISBN")
	}
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var sets []bnf.Record
	matched := map[string]*model.ExternalVolumeHit{} // set number → volume carrying the ISBN
	if isbn != "" {
		vols, _, err := c.BnF.Search(ctx, "bib.fuzzyIsbn all "+bnf.Quote(isbn), maxISBNVolumeHits)
		if err != nil {
			return nil, err
		}
		var numbers []string
		for _, v := range vols {
			for _, f := range setLinks(v) {
				n := f.Get("0")
				if bnf.ARK(n) == "" || matched[n] != nil {
					continue
				}
				vh := &model.ExternalVolumeHit{BnfArk: v.ARK, Title: v.Get("200", "a"), ISBN: bnf.NormalizeISBN(isbn)}
				if num, ok := volumeNumber(v, n); ok {
					vh.Number = &num
				}
				if t := v.Get("200", "i"); t != "" {
					vh.Title = t
				}
				matched[n] = vh
				numbers = append(numbers, n)
			}
		}
		if len(numbers) > 0 {
			q := "bib.recordid any " + bnf.Quote(joinSpace(numbers))
			if sets, _, err = c.BnF.Search(ctx, q, len(numbers)); err != nil {
				return nil, err
			}
		}
	} else {
		recs, _, err := c.BnF.Search(ctx, "bib.serialtitle all "+bnf.Quote(title), maxEditionHits)
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if r.IsText() {
				sets = append(sets, r)
			}
		}
	}

	hits := []model.ExternalEditionHit{}
	var arks []string
	for i := range sets {
		r := &sets[i]
		if r.ARK == "" {
			continue
		}
		h := model.ExternalEditionHit{
			BnfArk: r.ARK, Title: editionTitle(r), Publishers: publishers(r), SourceURL: bnf.RecordURL(r.ARK),
		}
		if h.Publishers == nil {
			h.Publishers = []string{}
		}
		if start, _ := bnf.Years(r.Get("100", "a")); start > 0 {
			h.StartYear = &start
		}
		if n, ok := bnf.RecordNumber(r.ARK); ok {
			h.MatchedVolume = matched[n]
		}
		hits = append(hits, h)
		arks = append(arks, r.ARK)
	}
	editionType := d.typeID(typeEdition)
	existing, err := c.existing(ctx, map[uuid.UUID][]string{editionType: arks}, []string{"external_ids", "bnf_ark"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].BnfArk]
	}
	return hits, nil
}

// existing maps external identifiers to the id of the catalogue item carrying them.
func (c *Connector) existing(ctx context.Context, byType map[uuid.UUID][]string, path []string) (map[string]*uuid.UUID, error) {
	out := map[string]*uuid.UUID{}
	for typeID, values := range byType {
		items, err := c.Catalog.FindItemsByAttribute(ctx, typeID, path, values, false)
		if err != nil {
			return nil, err
		}
		for i := range items {
			v := stringAt(parseObject(items[i].Attributes), path)
			if v != "" && out[v] == nil {
				id := items[i].ID
				out[v] = &id
			}
		}
	}
	return out, nil
}

func joinSpace(s []string) string { return strings.Join(s, " ") }
