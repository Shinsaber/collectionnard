package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shinsaber/colec/internal/model"
)

// allowedMimeTypes maps MIME types to file extensions.
var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadFile handles POST /api/v1/uploads (admin only).
// Accepts a multipart file upload and saves it to the configured upload dir.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	maxSize := h.Config.MaxUploadSize
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil { //nolint:gosec // corps déjà borné par MaxBytesReader ci-dessus
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "payload_too_large",
				fmt.Sprintf("file too large (max %d MiB)", maxSize/(1<<20)))
			return
		}
		respondError(w, http.StatusBadRequest, "bad_request", "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "missing 'file' field")
		return
	}
	defer func() { _ = file.Close() }()

	// Detect content type from file header (first 512 bytes)
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		respondInternal(w, r, err, "cannot read file")
		return
	}
	contentType := http.DetectContentType(buf[:n])

	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("file type not allowed: %s (accepted: %s)",
				contentType, strings.Join(allowedExtensions(), ", ")))
		return
	}

	// Reset reader to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(h.Config.UploadDir, 0o750); err != nil {
		respondInternal(w, r, err, "cannot create upload directory")
		return
	}

	filename, written, reused, err := h.storeUpload(file, ext)
	if err != nil {
		respondInternal(w, r, err, "cannot save file")
		return
	}
	respondJSON(w, http.StatusCreated, model.UploadResponse{
		URL:      "/api/v1/uploads/" + filename,
		Filename: filename,
		Size:     written, // octets réellement écrits (header.Size est déclaré par le client)
		Reused:   reused,
	})
}

func allowedExtensions() []string {
	exts := make([]string, 0, len(allowedMimeTypes))
	for mime := range allowedMimeTypes {
		exts = append(exts, mime)
	}
	return exts
}

// storeUpload writes the file under its content hash: <sha256 hex><ext>. An
// identical file already stored is reused (reused = true) and not rewritten.
// The content goes to a temporary file first (no whitelisted extension: never
// served nor purged), renamed once its hash is known.
func (h *Handler) storeUpload(src io.Reader, ext string) (filename string, written int64, reused bool, err error) {
	tmp, err := os.CreateTemp(h.Config.UploadDir, ".upload-*")
	if err != nil {
		return "", 0, false, err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name()) // sans effet après le renommage
	}()
	hash := sha256.New()
	if written, err = io.Copy(io.MultiWriter(tmp, hash), src); err != nil {
		return "", 0, false, err
	}
	if err = tmp.Close(); err != nil {
		return "", 0, false, err
	}
	filename = hex.EncodeToString(hash.Sum(nil)) + ext
	dest := filepath.Join(h.Config.UploadDir, filename)
	if _, statErr := os.Stat(dest); statErr == nil {
		// Date rafraîchie : le délai de grâce de la purge repart (l'image va être référencée).
		now := time.Now()
		return filename, written, true, os.Chtimes(dest, now, now)
	}
	// Lecture publique (comme les fichiers créés par os.Create auparavant) : images servies à tous.
	if err = os.Chmod(tmp.Name(), 0o644); err != nil { //nolint:gosec // images publiques
		return "", 0, false, err
	}
	return filename, written, false, os.Rename(tmp.Name(), dest)
}
