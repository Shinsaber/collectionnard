package handler

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// Formats lus pour les vignettes (en plus de JPEG) : ceux qu'accepte l'envoi.
	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// thumbWidths whitelists the thumbnail widths (?w=): one per use, so a client
// cannot make the server generate and store arbitrary sizes.
var thumbWidths = map[string]int{"240": 240}

// maxImagePixels caps the decoded size of an uploaded image (decompression
// bombs): refused at upload time when the format can be read (UploadFile), and
// never decoded for a thumbnail. A cover needs far less; 25 Mpx is ~100 MiB in RGBA.
const maxImagePixels = 25_000_000

// maxConcurrentThumbnails bounds the decodings in flight: each may hold up to
// ~100 MiB and the originals are sent by any account (audit S-001).
const maxConcurrentThumbnails = 2

// thumbnailWait is how long a request waits for a decoding slot before the
// original is served instead (sendfile, no decoding).
const thumbnailWait = 5 * time.Second

// thumbGate is the semaphore of thumbnail generation; the zero value is ready.
type thumbGate struct {
	once sync.Once
	sem  chan struct{}
}

// acquire takes a decoding slot, or reports false when none frees up before
// wait expires or the request is cancelled.
func (g *thumbGate) acquire(ctx context.Context, wait time.Duration) bool {
	g.once.Do(func() { g.sem = make(chan struct{}, maxConcurrentThumbnails) })
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case g.sem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	case <-timer.C:
		return false
	}
}

// release gives back a slot taken by acquire.
func (g *thumbGate) release() { <-g.sem }

// thumbDir holds the generated thumbnails, inside UPLOAD_DIR but hidden: the
// purge only lists the regular files at the top of UPLOAD_DIR.
const thumbDir = ".thumbs"

// thumbPath is where the thumbnail of `name` at `width` is stored.
func (h *Handler) thumbPath(width, name string) string {
	return filepath.Join(h.Config.UploadDir, thumbDir, width, strings.TrimSuffix(name, filepath.Ext(name))+".jpg")
}

// serveThumbnail answers GET /uploads/<name>?w=240: a JPEG at most `w` pixels
// wide (offline covers of the mobile app, lot E5d), generated on first request
// and kept on disk. An original that cannot be decoded, or asked for while
// maxConcurrentThumbnails decodings are already running, is served as is.
func (h *Handler) serveThumbnail(w http.ResponseWriter, r *http.Request, name, width string) {
	px, ok := thumbWidths[width]
	if !ok {
		respondError(w, http.StatusBadRequest, "bad_request", "unsupported thumbnail width")
		return
	}
	src := filepath.Join(h.Config.UploadDir, name)
	dst := h.thumbPath(width, name)
	if _, err := os.Stat(dst); err == nil {
		http.ServeFile(w, r, dst)
		return
	}
	if _, err := os.Stat(src); err != nil { //nolint:gosec // name sans « / » ni « .. », extension en liste blanche (route appelante)
		http.NotFound(w, r)
		return
	}
	if !h.thumbs.acquire(r.Context(), thumbnailWait) {
		http.ServeFile(w, r, src) //nolint:gosec // idem ; saturé : l'original reste affichable sans décodage
		return
	}
	err := writeThumbnail(src, dst, px)
	h.thumbs.release()
	if err != nil {
		http.ServeFile(w, r, src) //nolint:gosec // idem ; format non décodable ici : l'original reste affichable
		return
	}
	http.ServeFile(w, r, dst)
}

func writeThumbnail(src, dst string, width int) error {
	f, err := os.Open(src) //nolint:gosec // chemin construit par l'appelant (nom sans « / », extension en liste blanche)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return err
	}
	if imageTooLarge(cfg) {
		return errors.New("image too large or empty")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	b := img.Bounds()
	if b.Dx() < width {
		width = b.Dx() // jamais d'agrandissement
	}
	height := max(1, b.Dy()*width/b.Dx())
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fond blanc : la transparence (PNG, GIF, WebP) n'existe pas en JPEG.
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Over, nil)

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".thumb-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // sans effet après le renommage
	if err := jpeg.Encode(tmp, out, &jpeg.Options{Quality: 80}); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil { //nolint:gosec // images publiques
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

// imageTooLarge reports whether an image header describes an empty image or
// one above maxImagePixels.
func imageTooLarge(cfg image.Config) bool {
	return cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImagePixels || cfg.Width*cfg.Height > maxImagePixels
}

// removeThumbnails deletes the thumbnails of an original (purge of unused images).
func (h *Handler) removeThumbnails(name string) {
	for width := range thumbWidths {
		_ = os.Remove(h.thumbPath(width, name))
	}
}
