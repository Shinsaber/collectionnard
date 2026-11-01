package handler

import (
	"context"
	"fmt"
	"log/slog"
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
	rep, err := h.purgeUploads(r.Context(), r.URL.Query().Get("dry_run") == "true")
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, rep)
}

// RunUploadPurge deletes the unreferenced uploads now, then every interval,
// until ctx ends (UPLOAD_PURGE_INTERVAL ; the admin button does the same on
// demand). Without it, a member could fill the volume faster than an admin
// notices (audit S-004).
func (h *Handler) RunUploadPurge(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if rep, err := h.purgeUploads(ctx, false); err != nil {
			if ctx.Err() == nil {
				slog.Error("upload purge", "err", err)
			}
		} else if len(rep.Files) > 0 {
			slog.Info("upload purge", "removed", len(rep.Files), "bytes", rep.TotalSize)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// purgeUploads builds the purge report and, unless dryRun, deletes the listed
// files with their thumbnails.
func (h *Handler) purgeUploads(ctx context.Context, dryRun bool) (model.UploadPurgeReport, error) {
	rep := model.UploadPurgeReport{DryRun: dryRun, Files: []model.UploadFileInfo{}, GraceHours: int(uploadPurgeGrace.Hours())}
	refs, err := h.Store.ReferencedUploads(ctx)
	if err != nil {
		return rep, err
	}
	referenced := make(map[string]bool, len(refs))
	for _, n := range refs {
		referenced[n] = true
	}

	entries, err := os.ReadDir(h.Config.UploadDir)
	if err != nil && !os.IsNotExist(err) {
		return rep, fmt.Errorf("cannot read upload directory: %w", err)
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
				return rep, fmt.Errorf("cannot delete %s: %w", f.Filename, err)
			}
			h.removeThumbnails(f.Filename)
		}
		rep.Applied = true
	}
	return rep, nil
}

// servedImageExt reports whether the file name has an extension served under /api/v1/uploads/.
func servedImageExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}
