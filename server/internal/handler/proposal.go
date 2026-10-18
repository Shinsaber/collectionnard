package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/shinsaber/colec/internal/identifier"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// Proposals and moderation (P4-3b). A member proposes; admins and
// contributors moderate, never their own proposals. A catalog proposal is
// applied through the import, all or nothing, with its author as the author
// of the writes (origin `proposal`).

const (
	maxProposalTitle = 200
	maxProposalText  = 10000
	// maxProposalQuota caps PROPOSAL_QUOTA + accepted proposals; maxProposalsPerDay
	// limits the creations of a member in 24 h.
	maxProposalQuota   = 50
	maxProposalsPerDay = 20
)

func isModerator(r *http.Request) bool {
	return model.CanEditCatalog(middleware.GetRole(r.Context()))
}

func respondProposalState(w http.ResponseWriter) {
	respondError(w, http.StatusConflict, "proposal_state", "the proposal is not in a state allowing this")
}

// proposalStoreError maps ErrConflict (guarded status change) to proposal_state.
func proposalStoreError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrConflict) {
		respondProposalState(w)
		return
	}
	handleStoreError(w, r, err)
}

// checkInput validates a proposal content; kind is the kind of the proposal.
func checkInput(w http.ResponseWriter, kind string, in *model.ProposalInput) bool {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > maxProposalTitle:
		respondError(w, http.StatusBadRequest, "bad_request", "title: 1 to 200 characters")
	case utf8.RuneCountInString(in.Description) > maxProposalText:
		respondError(w, http.StatusBadRequest, "bad_request", "description: at most 10000 characters")
	case kind != model.ProposalCatalog && in.Payload != nil:
		respondError(w, http.StatusBadRequest, "bad_request", "only a catalog proposal carries a payload")
	case kind == model.ProposalReport && in.ItemID == nil:
		respondError(w, http.StatusBadRequest, "bad_request", "a report targets an item (item_id)")
	case kind != model.ProposalCatalog && in.Description == "":
		respondError(w, http.StatusBadRequest, "bad_request", "description is required")
	case in.Payload != nil && !checkPayload(w, in.Payload):
	default:
		return true
	}
	return false
}

// checkPayload: items and relations only (no meta-type), import format.
func checkPayload(w http.ResponseWriter, p *model.CatalogExport) bool {
	if p.Format == "" {
		p.Format, p.Version = model.CatalogFormat, model.ExportVersion
	}
	if len(p.ItemTypes) > 0 || len(p.RelationTypes) > 0 || len(p.CollectionTypes) > 0 {
		respondError(w, http.StatusUnprocessableEntity, "proposal_meta_types",
			"a proposal changes items and relations only")
		return false
	}
	return true
}

// readable: the author, or a moderator for anything but a draft.
func canRead(r *http.Request, p *model.Proposal) bool {
	if p.AuthorID == middleware.GetUserID(r.Context()) {
		return true
	}
	return isModerator(r) && p.Status != model.ProposalDraft
}

func (h *Handler) loadProposal(w http.ResponseWriter, r *http.Request) (*model.Proposal, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	p, err := h.Store.GetProposal(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if !canRead(r, p) {
		notFound(w)
		return nil, false
	}
	return p, true
}

func (h *Handler) ownProposal(w http.ResponseWriter, r *http.Request) (*model.Proposal, bool) {
	p, ok := h.loadProposal(w, r)
	if ok && p.AuthorID != middleware.GetUserID(r.Context()) {
		notFound(w)
		return nil, false
	}
	return p, ok
}

// moderated loads a proposal a moderator may decide on.
func (h *Handler) moderated(w http.ResponseWriter, r *http.Request) (*model.Proposal, bool) {
	if !isModerator(r) {
		respondError(w, http.StatusForbidden, "forbidden", "moderators only")
		return nil, false
	}
	p, ok := h.loadProposal(w, r)
	if !ok {
		return nil, false
	}
	if p.AuthorID == middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusForbidden, "own_proposal", "nobody moderates their own proposal")
		return nil, false
	}
	if p.Status != model.ProposalPending {
		respondProposalState(w)
		return nil, false
	}
	return p, true
}

// dailyQuota limits the creations of a member in 24 h (moderators: no limit).
func (h *Handler) dailyQuota(w http.ResponseWriter, r *http.Request) bool {
	if isModerator(r) {
		return true
	}
	_, _, today, err := h.Store.ProposalQuotaUsage(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return false
	}
	if today >= maxProposalsPerDay {
		respondError(w, http.StatusConflict, "proposal_quota", "too many proposals in 24 hours")
		return false
	}
	return true
}

// --- Auteur ---

func (h *Handler) ListProposals(w http.ResponseWriter, r *http.Request) {
	me := middleware.GetUserID(r.Context())
	f := model.ProposalFilter{Status: r.URL.Query().Get("status")}
	switch r.URL.Query().Get("scope") {
	case "", "mine":
		f.AuthorID = &me
	case "review":
		if !isModerator(r) {
			respondError(w, http.StatusForbidden, "forbidden", "moderators only")
			return
		}
		f.ExcludeAuthor, f.Review = &me, true
	default:
		respondError(w, http.StatusBadRequest, "bad_request", "scope: mine or review")
		return
	}
	pg := parsePagination(r)
	list, total, err := h.Store.ListProposals(r.Context(), f, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

func (h *Handler) CreateProposal(w http.ResponseWriter, r *http.Request) {
	var in model.ProposalInput
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Kind != model.ProposalCatalog && in.Kind != model.ProposalSchemaSuggestion && in.Kind != model.ProposalReport {
		respondError(w, http.StatusBadRequest, "bad_request", "kind: catalog, schema_suggestion or report")
		return
	}
	if !checkInput(w, in.Kind, &in) {
		return
	}
	if !h.dailyQuota(w, r) {
		return
	}
	p, err := h.Store.CreateProposal(r.Context(), middleware.GetUserID(r.Context()), in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if in.Submit {
		h.submit(w, r, p, http.StatusCreated)
		return
	}
	respondJSON(w, http.StatusCreated, p)
}

func (h *Handler) GetProposal(w http.ResponseWriter, r *http.Request) {
	if p, ok := h.loadProposal(w, r); ok {
		respondJSON(w, http.StatusOK, p)
	}
}

func (h *Handler) UpdateProposal(w http.ResponseWriter, r *http.Request) {
	p, ok := h.ownProposal(w, r)
	if !ok {
		return
	}
	var in model.ProposalInput
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !checkInput(w, p.Kind, &in) {
		return
	}
	updated, err := h.Store.UpdateProposal(r.Context(), p.ID, in)
	if err != nil {
		proposalStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) SubmitProposal(w http.ResponseWriter, r *http.Request) {
	if p, ok := h.ownProposal(w, r); ok {
		h.submit(w, r, p, http.StatusOK)
	}
}

// submit checks the quota and the payload (dry-run import), takes the base
// snapshot and moves the proposal to pending.
func (h *Handler) submit(w http.ResponseWriter, r *http.Request, p *model.Proposal, status int) {
	if p.Status != model.ProposalDraft && p.Status != model.ProposalChangesRequested {
		respondProposalState(w)
		return
	}
	if !isModerator(r) {
		waiting, accepted, _, err := h.Store.ProposalQuotaUsage(r.Context(), p.AuthorID)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if p.Status == model.ProposalChangesRequested {
			waiting-- // already counted
		}
		if waiting >= min(h.Config.ProposalQuota+accepted, maxProposalQuota) {
			respondError(w, http.StatusConflict, "proposal_quota", "too many proposals waiting for review")
			return
		}
	}
	var base *model.ProposalBase
	if p.Kind == model.ProposalCatalog {
		if p.Payload == nil || len(p.Payload.Items)+len(p.Payload.Relations) == 0 {
			respondError(w, http.StatusBadRequest, "bad_request", "the proposal changes nothing")
			return
		}
		rep, err := h.Store.ImportCatalog(r.Context(), p.Payload, true)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if len(rep.Errors) > 0 {
			respondJSON(w, http.StatusUnprocessableEntity, model.APIError{
				Error: "proposal_invalid", Message: "the proposal does not import", Details: rep,
			})
			return
		}
		if base, err = h.snapshot(r, p.Payload); err != nil {
			handleStoreError(w, r, err)
			return
		}
	}
	updated, err := h.Store.SubmitProposal(r.Context(), p.ID, base)
	if err != nil {
		proposalStoreError(w, r, err)
		return
	}
	respondJSON(w, status, updated)
}

// snapshot: current attributes of the existing entities the payload changes.
func (h *Handler) snapshot(r *http.Request, p *model.CatalogExport) (*model.ProposalBase, error) {
	items := make([]uuid.UUID, 0, len(p.Items))
	for _, it := range p.Items {
		items = append(items, it.ID)
	}
	rels := make([]uuid.UUID, 0, len(p.Relations))
	for _, rel := range p.Relations {
		rels = append(rels, rel.ID)
	}
	return h.Store.EntitySnapshot(r.Context(), items, rels)
}

func (h *Handler) WithdrawProposal(w http.ResponseWriter, r *http.Request) {
	p, ok := h.ownProposal(w, r)
	if !ok {
		return
	}
	updated, err := h.Store.WithdrawProposal(r.Context(), p.ID)
	if err != nil {
		proposalStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// --- Modération ---

func (h *Handler) CountProposalsToReview(w http.ResponseWriter, r *http.Request) {
	if !isModerator(r) {
		respondError(w, http.StatusForbidden, "forbidden", "moderators only")
		return
	}
	n, err := h.Store.CountProposalsToReview(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"count": n})
}

func sameJSON(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return bytes.Equal(a, b)
	}
	return reflect.DeepEqual(x, y)
}

// conflicts lists the entities of the base changed (or deleted) since.
func (h *Handler) conflicts(r *http.Request, p *model.Proposal, payload *model.CatalogExport) ([]model.ProposalConflict, error) {
	out := []model.ProposalConflict{}
	if p.Base == nil || payload == nil {
		return out, nil
	}
	cur, err := h.snapshot(r, payload)
	if err != nil {
		return nil, err
	}
	proposedItems := map[uuid.UUID]json.RawMessage{}
	for _, it := range payload.Items {
		proposedItems[it.ID] = it.Attributes
	}
	proposedRels := map[uuid.UUID]json.RawMessage{}
	for _, rel := range payload.Relations {
		proposedRels[rel.ID] = rel.Attributes
	}
	check := func(kind string, base, current, proposed map[uuid.UUID]json.RawMessage) {
		for id, b := range base {
			c, exists := current[id]
			if exists && sameJSON(b, c) {
				continue
			}
			pr, ok := proposed[id]
			if !ok {
				continue // no longer in the (corrected) payload
			}
			conf := model.ProposalConflict{Kind: kind, ID: id, Base: b, Current: json.RawMessage("null"), Proposed: pr}
			if exists {
				conf.Current = c
			}
			out = append(out, conf)
		}
	}
	check("item", p.Base.Items, cur.Items, proposedItems)
	check("relation", p.Base.Relations, cur.Relations, proposedRels)
	return out, nil
}

func (h *Handler) GetProposalConflicts(w http.ResponseWriter, r *http.Request) {
	if !isModerator(r) {
		respondError(w, http.StatusForbidden, "forbidden", "moderators only")
		return
	}
	p, ok := h.loadProposal(w, r)
	if !ok {
		return
	}
	c, err := h.conflicts(r, p, p.Payload)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, c)
}

func (h *Handler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	p, ok := h.moderated(w, r)
	if !ok {
		return
	}
	var in model.ProposalAccept
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &in); err != nil {
			respondDecodeError(w, err)
			return
		}
	}
	in.Comment = strings.TrimSpace(in.Comment)
	me := middleware.GetUserID(r.Context())
	if p.Kind != model.ProposalCatalog {
		updated, err := h.Store.ReviewProposal(r.Context(), p.ID, model.ProposalAccepted, me, in.Comment)
		if err != nil {
			proposalStoreError(w, r, err)
			return
		}
		h.notifyProposal(r, p, model.NotifyProposalAccepted, p.AuthorID)
		respondJSON(w, http.StatusOK, updated)
		return
	}
	payload := p.Payload
	if in.Payload != nil {
		if !checkPayload(w, in.Payload) {
			return
		}
		payload = in.Payload
	}
	if payload == nil {
		respondError(w, http.StatusBadRequest, "bad_request", "the proposal changes nothing")
		return
	}
	conflicts, err := h.conflicts(r, p, payload)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if len(conflicts) > 0 && !in.ConflictsResolved {
		respondJSON(w, http.StatusConflict, model.APIError{
			Error: "proposal_conflict", Message: "entities changed since the proposal was submitted",
			Details: map[string]any{"conflicts": conflicts},
		})
		return
	}
	// The writes are credited to the author of the proposal (history), not to the moderator.
	ctx := store.WithActor(r.Context(), store.Actor{UserID: p.AuthorID, Origin: store.OriginProposal, Ref: p.ID.String()})
	rep, err := h.Store.AcceptCatalogProposal(ctx, p.ID, me, payload, in.Comment)
	if err != nil {
		proposalStoreError(w, r, err)
		return
	}
	if len(rep.Errors) > 0 {
		respondJSON(w, http.StatusUnprocessableEntity, model.APIError{
			Error: "proposal_invalid", Message: "the proposal does not import", Details: rep,
		})
		return
	}
	h.notifyProposal(r, p, model.NotifyProposalAccepted, p.AuthorID)
	updated, err := h.Store.GetProposal(r.Context(), p.ID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request, status string) {
	p, ok := h.moderated(w, r)
	if !ok {
		return
	}
	var in model.ProposalDecision
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	in.Comment = strings.TrimSpace(in.Comment)
	if in.Comment == "" || utf8.RuneCountInString(in.Comment) > maxProposalText {
		respondError(w, http.StatusBadRequest, "bad_request", "comment: 1 to 10000 characters")
		return
	}
	updated, err := h.Store.ReviewProposal(r.Context(), p.ID, status, middleware.GetUserID(r.Context()), in.Comment)
	if err != nil {
		proposalStoreError(w, r, err)
		return
	}
	kind := model.NotifyProposalRejected
	if status == model.ProposalChangesRequested {
		kind = model.NotifyProposalChangesRequested
	}
	h.notifyProposal(r, p, kind, p.AuthorID)
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) RejectProposal(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, model.ProposalRejected)
}

func (h *Handler) RequestProposalChanges(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, model.ProposalChangesRequested)
}

// --- Discussion ---

func (h *Handler) ListProposalComments(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadProposal(w, r)
	if !ok {
		return
	}
	list, err := h.Store.ListProposalComments(r.Context(), p.ID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (h *Handler) CreateProposalComment(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadProposal(w, r)
	if !ok {
		return
	}
	if p.Final() {
		respondProposalState(w)
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" || utf8.RuneCountInString(in.Body) > maxProposalText {
		respondError(w, http.StatusBadRequest, "bad_request", "body: 1 to 10000 characters")
		return
	}
	c, err := h.Store.CreateProposalComment(r.Context(), p.ID, middleware.GetUserID(r.Context()), in.Body)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	h.notifyComment(r, p)
	respondJSON(w, http.StatusCreated, c)
}

// --- Doublons ---

// FindDuplicates lists the items of the same type that already carry a value
// of an `x-identifier` field of the attributes.
func (h *Handler) FindDuplicates(w http.ResponseWriter, r *http.Request) {
	var in model.DuplicateCheck
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	it, err := h.Store.GetItemType(r.Context(), in.ItemTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	out := []model.Duplicate{}
	seen := map[uuid.UUID]bool{}
	for _, f := range identifier.Fields(it.AttributesSchema) {
		v, ok := identifier.Value(in.Attributes, f)
		if !ok {
			continue
		}
		items, err := h.Store.FindItemsByAttribute(r.Context(), it.ID, f, []string{v}, false)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		for _, item := range items {
			if seen[item.ID] || (in.ExcludeID != nil && item.ID == *in.ExcludeID) {
				continue
			}
			seen[item.ID] = true
			item.ItemType = it
			out = append(out, model.Duplicate{Field: strings.Join(f, "."), Value: v, Item: item})
		}
	}
	respondJSON(w, http.StatusOK, out)
}
