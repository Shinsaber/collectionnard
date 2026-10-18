package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shinsaber/colec/internal/csvexport"
	"github.com/shinsaber/colec/internal/label"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

const (
	// maxExportRows caps the CSV exports (the JSON catalogue export is not capped).
	maxExportRows = 10000
	// exportPageSize is the page size used to stream the store listings.
	exportPageSize = 500
)

// ExportCatalog answers GET /export (admin): the whole catalogue as an
// indented JSON attachment, ready to be committed or re-imported.
func (h *Handler) ExportCatalog(w http.ResponseWriter, r *http.Request) {
	cat, err := h.Store.ExportCatalog(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	body, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	attachment(w, "application/json", "collectionnard-catalog-"+today()+".json")
	_, _ = w.Write(append(body, '\n'))
}

// ImportCatalog answers POST /import (admin): 200 with the report when the
// import was applied (or simulated without error), 422 with the report when
// nothing was applied because of validation errors.
func (h *Handler) ImportCatalog(w http.ResponseWriter, r *http.Request) {
	var cat model.CatalogExport
	dryRun, ok := h.readImport(w, r, &cat)
	if !ok {
		return
	}
	rep, err := h.Store.ImportCatalog(store.WithOrigin(r.Context(), store.OriginImport, ""), &cat, dryRun)
	respondImport(w, r, rep, err)
}

// ImportCollection answers POST /collections/import: restores a collection
// export into the account of the current user (same responses as /import).
func (h *Handler) ImportCollection(w http.ResponseWriter, r *http.Request) {
	var exp model.CollectionExport
	dryRun, ok := h.readImport(w, r, &exp)
	if !ok {
		return
	}
	rep, err := h.Store.ImportCollection(r.Context(), middleware.GetUserID(r.Context()), &exp, dryRun)
	respondImport(w, r, rep, err)
}

// readImport reads `dry_run` and decodes an import file (up to MaxImportSize)
// into v. It answers 400 / 413 and returns ok=false on failure.
func (h *Handler) readImport(w http.ResponseWriter, r *http.Request, v any) (dryRun, ok bool) {
	if dryRun, ok = parseDryRun(w, r); !ok {
		return false, false
	}
	// A large file and its transaction may outlast the server-wide timeouts (10 s read, 30 s write).
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(2 * time.Minute))
	_ = rc.SetWriteDeadline(time.Now().Add(3 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, h.Config.MaxImportSize)
	defer func() { _ = r.Body.Close() }()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "payload_too_large",
				fmt.Sprintf("import file too large (max %d bytes)", h.Config.MaxImportSize))
			return false, false
		}
		respondError(w, http.StatusBadRequest, "bad_request", "invalid JSON file: "+err.Error())
		return false, false
	}
	return dryRun, true
}

// parseDryRun reads the optional `dry_run` boolean (400 when malformed).
func parseDryRun(w http.ResponseWriter, r *http.Request) (dryRun, ok bool) {
	raw := r.URL.Query().Get("dry_run")
	if raw == "" {
		return false, true
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "dry_run must be true or false")
		return false, false
	}
	return b, true
}

// respondImport answers 200 with the report, 422 when it lists errors.
func respondImport(w http.ResponseWriter, r *http.Request, rep *model.ImportReport, err error) {
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	status := http.StatusOK
	if len(rep.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	respondJSON(w, status, rep)
}

// ExportItemsCSV answers GET /items/export: the items of one type, with the
// filters and sort of GET /items, as a CSV attachment.
func (h *Handler) ExportItemsCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("item_type_id") == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "item_type_id is required")
		return
	}
	delim, lang, ok := parseCSVOptions(w, r)
	if !ok {
		return
	}
	filters, ok := h.parseItemFilters(w, r)
	if !ok {
		return
	}
	it, err := h.Store.GetItemType(r.Context(), *filters.ItemTypeID)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusBadRequest, "bad_request", "unknown item_type_id")
		return
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}

	// Read everything before writing: an error can still become a JSON response.
	var items []model.Item
	for page := 1; len(items) < maxExportRows; page++ {
		batch, total, err := h.Store.ListItems(r.Context(), model.Pagination{Page: page, PerPage: exportPageSize}, filters)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		items = append(items, batch...)
		if len(batch) < exportPageSize || len(items) >= total {
			break
		}
	}
	if len(items) > maxExportRows {
		items = items[:maxExportRows]
	}

	cols := csvexport.Columns(it.AttributesSchema)
	header := []string{"id", "label", "tags"}
	for _, c := range cols {
		header = append(header, c.Header())
	}
	attachment(w, "text/csv; charset=utf-8", "items-"+safeName(it.Name)+"-"+today()+".csv")
	cw, err := csvexport.Writer(w, delim)
	if err != nil {
		return // client gone
	}
	_ = cw.Write(header)
	for _, item := range items {
		tags := make([]string, len(item.Tags))
		for i, t := range item.Tags {
			tags[i] = t.Name
		}
		row := []string{
			item.ID.String(),
			csvexport.Text(label.Render(it.AttributesSchema, item.Attributes, lang)),
			csvexport.Text(strings.Join(tags, "; ")),
		}
		_ = cw.Write(append(row, csvexport.Values(cols, item.Attributes)...))
	}
	cw.Flush()
}

// ExportCollection answers GET /collections/{id}/export: one collection of
// the current user and all its items, as JSON or CSV.
func (h *Handler) ExportCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		respondError(w, http.StatusBadRequest, "bad_request", "format must be json or csv")
		return
	}
	delim, lang, ok := parseCSVOptions(w, r)
	if !ok {
		return
	}
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	var entries []model.CollectionItem
	for page := 1; len(entries) < maxExportRows; page++ {
		batch, total, err := h.Store.ListCollectionItems(r.Context(), id,
			model.Pagination{Page: page, PerPage: exportPageSize}, nil, "created_at")
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		entries = append(entries, batch...)
		if len(batch) < exportPageSize || len(entries) >= total {
			break
		}
	}

	var ctSchema json.RawMessage
	header := model.CollectionExportHeader{ID: col.ID, Name: col.Name, Description: col.Description}
	if ct := col.CollectionType; ct != nil {
		ctSchema = ct.AttributesSchema
		header.CollectionType = model.ExportMetaType{
			ID: ct.ID, Name: ct.Name, Description: ct.Description, AttributesSchema: ct.AttributesSchema,
		}
	}
	rows := make([]model.CollectionExportItem, len(entries))
	for i, e := range entries {
		row := model.CollectionExportItem{ItemID: e.ItemID, Attributes: e.Attributes, AddedAt: e.CreatedAt}
		if e.Item != nil {
			row.ItemAttributes = e.Item.Attributes
			if e.Item.ItemType != nil {
				row.ItemType = e.Item.ItemType.Name
				row.Label = label.Render(e.Item.ItemType.AttributesSchema, e.Item.Attributes, lang)
			}
		}
		rows[i] = row
	}
	base := "collection-" + safeName(col.Name) + "-" + today()

	if format == "json" {
		body, err := json.MarshalIndent(model.CollectionExport{
			Format: model.CollectionFormat, Version: model.ExportVersion,
			ExportedAt: time.Now().UTC().Truncate(time.Second), Collection: header, Items: rows,
		}, "", "  ")
		if err != nil {
			respondInternal(w, r, err, "internal error")
			return
		}
		attachment(w, "application/json", base+".json")
		_, _ = w.Write(append(body, '\n'))
		return
	}

	cols := csvexport.Columns(ctSchema)
	head := []string{"item_id", "item_type", "label"}
	for _, c := range cols {
		head = append(head, c.Header())
	}
	attachment(w, "text/csv; charset=utf-8", base+".csv")
	cw, err := csvexport.Writer(w, delim)
	if err != nil {
		return
	}
	_ = cw.Write(append(head, "added_at"))
	for _, row := range rows {
		line := []string{row.ItemID.String(), row.ItemType, csvexport.Text(row.Label)}
		line = append(line, csvexport.Values(cols, row.Attributes)...)
		_ = cw.Write(append(line, row.AddedAt.UTC().Format(time.RFC3339)))
	}
	cw.Flush()
}

// langPattern accepts BCP 47-like codes ("fr", "pt-BR").
var langPattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// parseCSVOptions reads `delimiter` and `lang` (default label.Langs[0]).
func parseCSVOptions(w http.ResponseWriter, r *http.Request) (rune, string, bool) {
	q := r.URL.Query()
	delim, err := csvexport.Delimiter(q.Get("delimiter"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return 0, "", false
	}
	lang := q.Get("lang")
	if lang == "" {
		lang = label.Langs[0]
	}
	if !langPattern.MatchString(lang) {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid lang")
		return 0, "", false
	}
	return delim, lang, true
}

// attachment sets the headers of a downloaded file.
func attachment(w http.ResponseWriter, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")
}

var unsafeFilename = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// safeName keeps a file-name friendly version of a user-provided name.
func safeName(s string) string {
	s = strings.Trim(unsafeFilename.ReplaceAllString(s, "-"), "-")
	if len(s) > 60 {
		s = s[:60]
	}
	if s == "" {
		return "export"
	}
	return s
}

func today() string { return time.Now().UTC().Format("2006-01-02") }
