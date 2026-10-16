package handler_test

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shinsaber/colec/internal/testutil"
)

// E5d: /uploads/<name>?w=240 serves a 240 px wide JPEG, generated once and kept
// under .thumbs/ (removed with its original by the purge).
func TestServeThumbnail(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	src := image.NewRGBA(image.Rect(0, 0, 600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 600; x++ {
			src.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	f, err := os.Create(filepath.Join(cfg.UploadDir, "cover.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, src); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	old := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(filepath.Join(cfg.UploadDir, "cover.png"), old, old)

	mock := &testutil.MockStore{ReferencedUploadsFn: func(context.Context) ([]string, error) { return nil, nil }}
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(mock, cfg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/uploads/cover.png?w=240")
	if err != nil {
		t.Fatal(err)
	}
	thumb, format, err := image.Decode(resp.Body)
	_ = resp.Body.Close()
	if err != nil || format != "jpeg" || thumb.Bounds().Dx() != 240 || thumb.Bounds().Dy() != 360 {
		t.Fatalf("thumbnail: %v %s %v", err, format, thumb.Bounds())
	}
	cached := filepath.Join(cfg.UploadDir, ".thumbs", "240", "cover.jpg")
	if _, err := os.Stat(cached); err != nil {
		t.Errorf("thumbnail not kept on disk: %v", err)
	}
	if _, err := jpeg.DecodeConfig(mustOpen(t, cached)); err != nil {
		t.Errorf("cached thumbnail: %v", err)
	}

	if resp, _ := http.Get(srv.URL + "/api/v1/uploads/cover.png?w=999"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unsupported width: %d", resp.StatusCode)
	}
	if resp, _ := http.Get(srv.URL + "/api/v1/uploads/missing.png?w=240"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing original: %d", resp.StatusCode)
	}

	// The purge removes the thumbnail with its unused original.
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads/purge", nil)
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("purge: %v", err)
	}
	if _, err := os.Stat(cached); !os.IsNotExist(err) {
		t.Errorf("thumbnail left after purge: %v", err)
	}
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
