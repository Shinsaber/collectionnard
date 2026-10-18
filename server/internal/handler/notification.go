package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
)

// Notifications in the application (P4-3c). Sending one never fails the
// request that triggers it: errors are only logged.

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	list, total, unread, err := h.Store.ListNotifications(r.Context(), middleware.GetUserID(r.Context()), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NotificationPage{
		PaginatedResponse: model.NewPaginatedResponse(list, total, pg), Unread: unread,
	})
}

func (h *Handler) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &in); err != nil {
			respondDecodeError(w, err)
			return
		}
	}
	if err := h.Store.MarkNotificationsRead(r.Context(), middleware.GetUserID(r.Context()), in.IDs); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// notifyProposal records kind for users (the current user excluded).
func (h *Handler) notifyProposal(r *http.Request, p *model.Proposal, kind string, users ...uuid.UUID) {
	me := middleware.GetUserID(r.Context())
	var to []uuid.UUID
	for _, u := range users {
		if u != uuid.Nil && u != me {
			to = append(to, u)
		}
	}
	if len(to) == 0 {
		return
	}
	payload := map[string]any{"proposal_id": p.ID, "title": p.Title}
	if u, err := h.Store.GetUserByID(r.Context(), me); err == nil {
		payload["by"] = u.Username
	}
	if err := h.Store.CreateNotifications(r.Context(), to, kind, payload); err != nil {
		slog.WarnContext(r.Context(), "notification not recorded", "kind", kind, "proposal", p.ID, "err", err)
	}
}

// notifyComment tells the other participants of a discussion: the author, the
// reviewer and everyone who wrote in it.
func (h *Handler) notifyComment(r *http.Request, p *model.Proposal) {
	users := []uuid.UUID{p.AuthorID}
	if p.ReviewerID != nil {
		users = append(users, *p.ReviewerID)
	}
	comments, err := h.Store.ListProposalComments(r.Context(), p.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "notification recipients unknown", "proposal", p.ID, "err", err)
	}
	for _, c := range comments {
		users = append(users, c.AuthorID)
	}
	h.notifyProposal(r, p, model.NotifyProposalComment, users...)
}
