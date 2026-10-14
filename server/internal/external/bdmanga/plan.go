package bdmanga

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// matchKey is one way to find the existing item of a node.
type matchKey struct {
	path  []string
	value string
	fold  bool // case- and accent-insensitive (publisher names)
}

type node struct {
	key, role, parent, sourceURL, note string
	typeID                             uuid.UUID
	attrs                              map[string]any // from the source, fitted to the schema
	keys                               []matchKey
	selected                           bool

	existing *model.Item
	merged   map[string]any // attributes to write
}

type relation struct {
	typeID   uuid.UUID
	from, to string
	attrs    map[string]any
	existing *model.ItemRelation
	// orderOfTarget: `order` follows the `number` of the target (volume in its
	// edition), also when the number is corrected before the import.
	orderOfTarget bool
}

// planner builds the plan of one request.
type planner struct {
	c     *Connector
	d     *domain
	today string
	nodes []*node
	byKey map[string]*node
	rels  []*relation
	// alias: node merged into another one matching the same item.
	alias map[string]string
}

func (c *Connector) newPlanner(ctx context.Context) (*planner, error) {
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	return &planner{c: c, d: d, today: c.today(), byKey: map[string]*node{}, alias: map[string]string{}}, nil
}

// add registers a node, or returns the node already registered under its key.
func (p *planner) add(n *node) *node {
	if prev := p.byKey[n.key]; prev != nil {
		return prev
	}
	n.attrs = fit(p.d.schemas[n.typeID], n.attrs)
	p.nodes = append(p.nodes, n)
	p.byKey[n.key] = n
	return n
}

// link adds a relation; the same (type, from, to) only fills the empty
// attributes of the first one.
func (p *planner) link(name, from, to string, attrs map[string]any) *relation {
	typeID := p.d.relations[name]
	if attrs == nil {
		attrs = map[string]any{}
	}
	for _, r := range p.rels {
		if r.typeID == typeID && r.from == from && r.to == to {
			r.attrs = fillEmpty(r.attrs, attrs)
			return r
		}
	}
	r := &relation{typeID: typeID, from: from, to: to, attrs: attrs}
	p.rels = append(p.rels, r)
	return r
}

// sources is the provenance entry of a node.
func (p *planner) sources(source string) []any {
	return []any{map[string]any{"source": source, "retrieved_at": p.today}}
}

// ── Building ────────────────────────────────────────────────────────

func (c *Connector) plan(ctx context.Context, req model.ExternalImportRequest) (*planner, error) {
	if req.Work == "" && len(req.Editions) == 0 {
		return nil, InputError("work or editions is required")
	}
	if req.Work != "" && !wikidata.IsQID(req.Work) {
		return nil, InputError("work must be a Wikidata identifier (Q…)")
	}
	if len(req.Editions) > maxEditions {
		return nil, InputError(fmt.Sprintf("at most %d editions", maxEditions))
	}
	for _, ark := range req.Editions {
		if _, ok := bnf.RecordNumber(ark); !ok {
			return nil, InputError("invalid BnF ARK: " + ark)
		}
	}
	p, err := c.newPlanner(ctx)
	if err != nil {
		return nil, err
	}
	workKey := ""
	if req.Work != "" {
		if workKey, err = p.addWork(ctx, req.Work); err != nil {
			return nil, err
		}
	}
	for _, ark := range slices.Compact(slices.Clone(req.Editions)) {
		if err := p.addEdition(ctx, ark, workKey); err != nil {
			return nil, err
		}
	}
	if err := p.matchItems(ctx); err != nil {
		return nil, err
	}
	if err := p.matchRelations(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

// ── Matching ────────────────────────────────────────────────────────

// matchItems finds the existing item of each node (first matching key wins),
// merges nodes that point to the same item and computes the attributes to write.
func (p *planner) matchItems(ctx context.Context) error {
	type group struct {
		typeID uuid.UUID
		path   string
		fold   bool
	}
	found := map[group]map[string]*model.Item{} // exact value → item
	values := map[group][]string{}
	for _, n := range p.nodes {
		for _, k := range n.keys {
			g := group{n.typeID, strings.Join(k.path, "\x00"), k.fold}
			if !slices.Contains(values[g], k.value) {
				values[g] = append(values[g], k.value)
			}
		}
	}
	for g, vals := range values {
		path := strings.Split(g.path, "\x00")
		found[g] = map[string]*model.Item{}
		if g.fold {
			// Accent folding happens in SQL: one query per value.
			for _, v := range vals {
				items, err := p.c.Catalog.FindItemsByAttribute(ctx, g.typeID, path, []string{v}, true)
				if err != nil {
					return err
				}
				if len(items) > 0 {
					found[g][v] = &items[0]
				}
			}
			continue
		}
		items, err := p.c.Catalog.FindItemsByAttribute(ctx, g.typeID, path, vals, false)
		if err != nil {
			return err
		}
		for i := range items {
			v := stringAt(parseObject(items[i].Attributes), path)
			if _, dup := found[g][v]; v != "" && !dup {
				found[g][v] = &items[i]
			}
		}
	}

	owner := map[uuid.UUID]string{} // item id → key of the node that owns it
	kept := p.nodes[:0]
	for _, n := range p.nodes {
		for _, k := range n.keys {
			if it := found[group{n.typeID, strings.Join(k.path, "\x00"), k.fold}][k.value]; it != nil {
				n.existing = it
				break
			}
		}
		if n.existing != nil {
			if first, ok := owner[n.existing.ID]; ok {
				p.alias[n.key] = first
				delete(p.byKey, n.key)
				continue
			}
			owner[n.existing.ID] = n.key
			n.merged = fillEmpty(parseObject(n.existing.Attributes), n.attrs)
		} else {
			n.merged = n.attrs
		}
		kept = append(kept, n)
	}
	p.nodes = kept
	if len(p.alias) > 0 {
		rels := p.rels
		p.rels = nil
		for _, r := range rels {
			nr := p.link(p.relName(r.typeID), p.resolve(r.from), p.resolve(r.to), r.attrs)
			nr.orderOfTarget = nr.orderOfTarget || r.orderOfTarget
		}
	}
	return nil
}

func (p *planner) resolve(key string) string {
	if a, ok := p.alias[key]; ok {
		return a
	}
	return key
}

func (p *planner) relName(id uuid.UUID) string {
	for name, rid := range p.d.relations {
		if rid == id {
			return name
		}
	}
	return ""
}

// matchRelations reuses the existing relations between matched items and
// fills their empty attributes (existing roles are kept as they are).
func (p *planner) matchRelations(ctx context.Context) error {
	var ids []uuid.UUID
	for _, n := range p.nodes {
		if n.existing != nil {
			ids = append(ids, n.existing.ID)
		}
	}
	if len(ids) < 2 {
		return nil
	}
	existing, err := p.c.Catalog.ListRelationsBetween(ctx, ids)
	if err != nil {
		return err
	}
	for _, r := range p.rels {
		from, to := p.byKey[r.from], p.byKey[r.to]
		if from == nil || to == nil || from.existing == nil || to.existing == nil {
			continue
		}
		for i := range existing {
			e := &existing[i]
			if e.RelationTypeID == r.typeID && e.SourceItemID == from.existing.ID && e.TargetItemID == to.existing.ID {
				r.existing = e
				r.attrs = fillEmpty(parseObject(e.Attributes), r.attrs)
				break
			}
		}
	}
	return nil
}

func stringAt(m map[string]any, path []string) string {
	var cur any = m
	for _, k := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[k]
	}
	s, _ := cur.(string)
	return s
}

// ── Output ──────────────────────────────────────────────────────────

func (n *node) status() string {
	switch {
	case n.existing == nil:
		return model.NodeNew
	case sameJSON(parseObject(n.existing.Attributes), n.merged):
		return model.NodeUnchanged
	}
	return model.NodeUpdate
}

func (p *planner) public() *model.ExternalPlan {
	out := &model.ExternalPlan{RetrievedAt: p.today, Nodes: []model.ExternalNode{}, Relations: []model.ExternalRelation{}}
	for _, n := range p.nodes {
		en := model.ExternalNode{
			Key: n.key, Role: n.role, ItemTypeID: n.typeID, Attributes: mustJSON(n.merged),
			Status: n.status(), Selected: n.selected, Parent: p.resolve(n.parent),
			SourceURL: n.sourceURL, Note: n.note,
		}
		if n.existing != nil {
			id := n.existing.ID
			en.ExistingItemID = &id
		}
		out.Nodes = append(out.Nodes, en)
	}
	for _, r := range p.rels {
		er := model.ExternalRelation{RelationTypeID: r.typeID, From: r.from, To: r.to, Attributes: mustJSON(r.attrs)}
		if r.existing != nil {
			id := r.existing.ID
			er.ExistingID = &id
		}
		out.Relations = append(out.Relations, er)
	}
	return out
}

// catalog converts the selected nodes into an import file: selected nodes are
// written (created, or completed when they exist), with the attributes
// corrected in the preview when overrides has their key, and a relation is
// written when each end is selected or already in the catalogue.
func (p *planner) catalog(selected []string, overrides map[string]json.RawMessage) (*model.CatalogExport, map[string]uuid.UUID, error) {
	want := map[string]bool{}
	for _, k := range selected {
		k = p.resolve(k)
		if p.byKey[k] == nil {
			return nil, nil, InputError("unknown node key: " + k)
		}
		want[k] = true
	}
	fixed := map[string]map[string]any{}
	for k, raw := range overrides {
		k = p.resolve(k)
		if !want[k] {
			return nil, nil, InputError("override of a node that is not selected: " + k)
		}
		var attrs map[string]any
		if err := json.Unmarshal(raw, &attrs); err != nil || attrs == nil {
			return nil, nil, InputError("override of " + k + " must be a JSON object")
		}
		fixed[k] = attrs
	}
	cat := &model.CatalogExport{
		Format: model.CatalogFormat, Version: model.ExportVersion,
		ItemTypes: []model.ExportMetaType{}, CollectionTypes: []model.ExportMetaType{},
		RelationTypes: []model.ExportRelationType{}, Items: []model.ExportItem{}, Relations: []model.ExportRelation{},
	}
	ids := map[string]uuid.UUID{}
	for _, n := range p.nodes {
		if n.existing != nil {
			ids[n.key] = n.existing.ID
		}
		if !want[n.key] {
			continue
		}
		attrs := n.merged
		if f, ok := fixed[n.key]; ok {
			attrs = f
		}
		it := model.ExportItem{ItemTypeID: n.typeID, Attributes: mustJSON(attrs)}
		if n.existing != nil {
			it.ID = n.existing.ID
			for _, t := range n.existing.Tags {
				it.Tags = append(it.Tags, t.Name) // the import replaces the tags: keep them
			}
		} else {
			it.ID = uuid.New()
			ids[n.key] = it.ID
		}
		cat.Items = append(cat.Items, it)
	}
	written := map[string]uuid.UUID{}
	for k := range want {
		written[k] = ids[k]
	}
	for _, r := range p.rels {
		from, okFrom := ids[r.from]
		to, okTo := ids[r.to]
		if !okFrom || !okTo || (!want[r.from] && !want[r.to]) {
			continue // an end neither selected nor in the catalogue, or nothing new on either side
		}
		relAttrs := r.attrs
		if n, ok := fixed[r.to]["number"]; ok && r.orderOfTarget {
			relAttrs = map[string]any{}
			for k, v := range r.attrs {
				relAttrs[k] = v
			}
			relAttrs["order"] = n // the corrected number of the volume
		}
		rel := model.ExportRelation{RelationTypeID: r.typeID, SourceItemID: from, TargetItemID: to, Attributes: mustJSON(relAttrs)}
		if r.existing != nil {
			rel.ID = r.existing.ID
		} else {
			rel.ID = uuid.New()
		}
		cat.Relations = append(cat.Relations, rel)
	}
	return cat, written, nil
}

// Preview returns what the sources propose for a selection.
func (c *Connector) Preview(ctx context.Context, req model.ExternalImportRequest) (*model.ExternalPlan, error) {
	p, err := c.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return p.public(), nil
}

// Build rebuilds the plan of a selection (the answers of the sources are
// cached) and converts its selected nodes into an import file, with the item
// id of each written node.
func (c *Connector) Build(ctx context.Context, req model.ExternalImportRequest) (*model.CatalogExport, map[string]uuid.UUID, error) {
	if len(req.Selected) == 0 {
		return nil, nil, InputError("selected is required")
	}
	p, err := c.plan(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return p.catalog(req.Selected, req.Overrides)
}
