package handler_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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

// S-001 : des demandes simultanées de vignettes répondent toutes (vignette ou
// original), le nombre de décodages en vol étant borné par le handler.
func TestServeThumbnail_Concurrent(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	const n = 8
	for i := 0; i < n; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 400, 300))
		img.Set(i, 0, color.RGBA{R: 255, A: 255}) // contenu distinct : fichiers distincts
		f, err := os.Create(filepath.Join(cfg.UploadDir, fmt.Sprintf("c%d.png", i)))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/uploads/c%d.png?w=240", srv.URL, i))
			if err != nil {
				return
			}
			_, _, decErr := image.Decode(resp.Body)
			_ = resp.Body.Close()
			if decErr == nil {
				codes[i] = resp.StatusCode
			}
		}(i)
	}
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusOK {
			t.Errorf("request %d: no decodable image (status %d)", i, c)
		}
	}
}

// S-001 : un en-tête PNG qui annonce plus de maxImagePixels n'est jamais décodé :
// l'original est servi tel quel à la place de la vignette.
func TestServeThumbnail_TooLargeHeader(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	raw := fakePNGHeader(7000, 7000)
	if err := os.WriteFile(filepath.Join(cfg.UploadDir, "bomb.png"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/uploads/bomb.png?w=240")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Equal(body, raw) {
		t.Errorf("expected the original served as is, got %d (%d bytes)", resp.StatusCode, len(body))
	}
	if _, err := os.Stat(filepath.Join(cfg.UploadDir, ".thumbs", "240", "bomb.jpg")); !os.IsNotExist(err) {
		t.Error("a thumbnail was written for an image above the cap")
	}
}

// fakePNGHeader builds the signature and IHDR chunk of a PNG of the given size,
// without pixel data: enough for image.DecodeConfig, nothing to decode.
func fakePNGHeader(width, height int) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(height))
	ihdr[8], ihdr[9] = 8, 2 // 8 bits, truecolor
	chunk := append([]byte("IHDR"), ihdr...)
	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	_ = binary.Write(&out, binary.BigEndian, uint32(13))
	out.Write(chunk)
	_ = binary.Write(&out, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return out.Bytes()
}
