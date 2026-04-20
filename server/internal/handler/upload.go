package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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

	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("Fichier trop volumineux (max %d Mo)", maxSize/(1<<20)))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "Champ 'file' manquant")
		return
	}
	defer file.Close()

	// Detect content type from file header (first 512 bytes)
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		respondError(w, http.StatusInternalServerError, "internal", "Impossible de lire le fichier")
		return
	}
	contentType := http.DetectContentType(buf[:n])

	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("Type de fichier non autorisé: %s. Types acceptés: %s",
				contentType, strings.Join(allowedExtensions(), ", ")))
		return
	}

	// Reset reader to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		respondError(w, http.StatusInternalServerError, "internal", "Erreur interne")
		return
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(h.Config.UploadDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "internal", "Impossible de créer le dossier uploads")
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + ext
	destPath := filepath.Join(h.Config.UploadDir, filename)

	dst, err := os.Create(destPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal", "Impossible de sauvegarder le fichier")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(destPath) // cleanup on failure
		respondError(w, http.StatusInternalServerError, "internal", "Erreur d'écriture du fichier")
		return
	}

	url := "/api/v1/uploads/" + filename
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"url":      url,
		"filename": filename,
		"size":     header.Size,
	})
}

func allowedExtensions() []string {
	exts := make([]string, 0, len(allowedMimeTypes))
	for mime := range allowedMimeTypes {
		exts = append(exts, mime)
	}
	return exts
}
