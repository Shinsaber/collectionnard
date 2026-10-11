package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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

	// Generate unique filename
	filename := uuid.New().String() + ext
	destPath := filepath.Join(h.Config.UploadDir, filename)

	// destPath = UPLOAD_DIR + UUID + extension issue de allowedMimeTypes : aucune donnée client.
	dst, err := os.Create(destPath) //nolint:gosec // nom généré (UUID), extension issue d'une liste blanche
	if err != nil {
		respondInternal(w, r, err, "cannot save file")
		return
	}
	defer func() { _ = dst.Close() }()

	written, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(destPath) // cleanup on failure
		respondInternal(w, r, err, "cannot write file")
		return
	}

	url := "/api/v1/uploads/" + filename
	respondJSON(w, http.StatusCreated, model.UploadResponse{
		URL:      url,
		Filename: filename,
		Size:     written, // octets réellement écrits (header.Size est déclaré par le client)
	})
}

func allowedExtensions() []string {
	exts := make([]string, 0, len(allowedMimeTypes))
	for mime := range allowedMimeTypes {
		exts = append(exts, mime)
	}
	return exts
}
