package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shinsaber/colec/internal/model"
)

// uploadPurgeGrace protects recent files: an image is sent as soon as it is
// chosen, before the form that references it is saved.
const uploadPurgeGrace = 24 * time.Hour

// PurgeUploads answers POST /uploads/purge (admin): the files of UPLOAD_DIR that
// no attribute references and older than the grace period. dry_run=true only
// lists them; otherwise they are deleted. Only image files (served extensions)
// are considered: temporary files and anything else are left alone.
func (h *Handler) PurgeUploads(w http.ResponseWriter, r *http.Request) {
	dryRun := r.URL.Query().Get("dry_run") == "true"
	refs, err := h.Store.ReferencedUploads(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	referenced := make(map[string]bool, len(refs))
	for _, n := range refs {
		referenced[n] = true
	}

	rep := model.UploadPurgeReport{DryRun: dryRun, Files: []model.UploadFileInfo{}, GraceHours: int(uploadPurgeGrace.Hours())}
	entries, err := os.ReadDir(h.Config.UploadDir)
	if err != nil && !os.IsNotExist(err) {
		respondInternal(w, r, err, "cannot read upload directory")
		return
	}
	limit := time.Now().Add(-uploadPurgeGrace)
	for _, e := range entries {
		if !e.Type().IsRegular() || !servedImageExt(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // supprimé entre-temps
		}
		switch {
		case referenced[e.Name()]:
			rep.Referenced++
		case info.ModTime().After(limit):
			rep.Recent++
		default:
			rep.Files = append(rep.Files, model.UploadFileInfo{Filename: e.Name(), Size: info.Size(), ModifiedAt: info.ModTime().UTC()})
			rep.TotalSize += info.Size()
		}
	}
	sort.Slice(rep.Files, func(i, j int) bool { return rep.Files[i].ModifiedAt.Before(rep.Files[j].ModifiedAt) })

	if !dryRun {
		for _, f := range rep.Files {
			// Nom lu dans UPLOAD_DIR (pas de « / »), extension en liste blanche.
			if err := os.Remove(filepath.Join(h.Config.UploadDir, f.Filename)); err != nil && !os.IsNotExist(err) {
				respondInternal(w, r, err, "cannot delete "+f.Filename)
				return
			}
		}
		rep.Applied = true
	}
	respondJSON(w, http.StatusOK, rep)
}

// servedImageExt reports whether the file name has an extension served under /api/v1/uploads/.
func servedImageExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}
