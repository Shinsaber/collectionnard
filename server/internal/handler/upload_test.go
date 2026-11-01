package handler_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestUploadFile_Success(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	// Create a valid PNG image in memory
	body, contentType := createMultipartImage(t, "file", "test.png")

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+testutil.AdminToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}
}

// A member may upload (profile avatar, P4-0; images of a proposal, P4-3).
func TestUploadFile_AllowedForMember(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, contentType := createMultipartImage(t, "file", "test.png")

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+testutil.UserToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestUploadFile_InvalidMimeType(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	// Create a multipart form with a text file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("this is not an image"))
	writer.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testutil.AdminToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUploadFile_MissingField(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testutil.AdminToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServeUploadedFile(t *testing.T) {
	mock := &testutil.MockStore{}
	h := testutil.NewTestHandler(mock)

	// Create a temp file in the upload dir
	uploadDir := os.TempDir()
	h.Config.UploadDir = uploadDir

	// Create a valid PNG
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var imgBuf bytes.Buffer
	png.Encode(&imgBuf, img)

	testFile := filepath.Join(uploadDir, "test-serve.png")
	os.WriteFile(testFile, imgBuf.Bytes(), 0o644)
	defer os.Remove(testFile)

	srv := testutil.NewTestServerFromHandler(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/uploads/test-serve.png", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServeUploadedFile_DirectoryTraversal(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/uploads/..%2F..%2Fetc%2Fpasswd", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// --- helpers ---

func createMultipartImage(t *testing.T, fieldName, fileName string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatal(err)
	}
	// Generate a tiny valid PNG
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := png.Encode(part, img); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	return &buf, writer.FormDataContentType()
}

// E4a: a file is stored under its SHA-256; sending it again reuses it.
func TestUploadFile_Deduplicated(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	send := func() model.UploadResponse {
		body, contentType := createMultipartImage(t, "file", "test.png")
		raw := body.Bytes()
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", bytes.NewReader(raw))
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+testutil.AdminToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		var out model.UploadResponse
		if resp.StatusCode != http.StatusCreated || json.NewDecoder(resp.Body).Decode(&out) != nil {
			t.Fatalf("upload: %d", resp.StatusCode)
		}
		return out
	}
	first, second := send(), send()
	if first.Reused || !second.Reused || first.URL != second.URL {
		t.Errorf("first %+v, second %+v", first, second)
	}
	if len(first.Filename) != 64+len(".png") || filepath.Ext(first.Filename) != ".png" {
		t.Errorf("filename %q: expected <sha256 hex>.png", first.Filename)
	}
	entries, _ := os.ReadDir(cfg.UploadDir)
	if len(entries) != 1 || entries[0].Name() != first.Filename {
		t.Errorf("upload dir: %v (temporary file left or duplicate written)", entries)
	}
}

// S-004 : quota horaire par compte (fichiers) ; 429 upload_quota avec Retry-After ;
// les admins n'y sont pas soumis.
func TestUploadFile_Quota(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	cfg.UploadRateLimit = 2
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	send := func(token string, i int) *http.Response {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, _ := writer.CreateFormFile("file", "test.png")
		img := image.NewRGBA(image.Rect(0, 0, 4, 1))
		img.Set(i, 0, color.White) // contenu distinct à chaque envoi
		_ = png.Encode(part, img)
		_ = writer.Close()
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}
	for i := 0; i < 2; i++ {
		if resp := send(testutil.UserToken(), i); resp.StatusCode != http.StatusCreated {
			t.Fatalf("upload %d: %d", i+1, resp.StatusCode)
		}
	}
	resp := send(testutil.UserToken(), 2)
	var apiErr model.APIError
	_ = json.NewDecoder(resp.Body).Decode(&apiErr)
	if resp.StatusCode != http.StatusTooManyRequests || apiErr.Error != "upload_quota" || resp.Header.Get("Retry-After") == "" {
		t.Errorf("3rd upload: %d %+v (Retry-After %q)", resp.StatusCode, apiErr, resp.Header.Get("Retry-After"))
	}
	if resp := send(testutil.ContributorToken(), 0); resp.StatusCode != http.StatusCreated {
		t.Errorf("another account is independent: %d", resp.StatusCode)
	}
	for i := 0; i < 3; i++ {
		if resp := send(testutil.AdminToken(), i); resp.StatusCode != http.StatusCreated {
			t.Errorf("admin upload %d: %d (admins have no quota)", i+1, resp.StatusCode)
		}
	}
}

// S-004 : le quota compte aussi les octets.
func TestUploadFile_QuotaBytes(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	cfg.UploadRateBytes = 100 // un PNG 1×1 pèse ~70 octets : le 2e envoi passe, le 3e non
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	codes := []int{}
	for i := 0; i < 3; i++ {
		body, contentType := createMultipartImage(t, "file", "test.png")
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+testutil.UserToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		codes = append(codes, resp.StatusCode)
	}
	if codes[0] != 201 || codes[1] != 201 || codes[2] != 429 {
		t.Errorf("codes %v, want [201 201 429]", codes)
	}
}

// S-001 : une image dont l'en-tête annonce plus de 25 Mpx est refusée à l'envoi.
func TestUploadFile_ImageTooLarge(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "bomb.png")
	_, _ = part.Write(fakePNGHeader(7000, 7000))
	_ = writer.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var apiErr model.APIError
	_ = json.NewDecoder(resp.Body).Decode(&apiErr)
	if resp.StatusCode != http.StatusBadRequest || apiErr.Error != "image_too_large" {
		t.Errorf("got %d %+v, want 400 image_too_large", resp.StatusCode, apiErr)
	}
	if entries, _ := os.ReadDir(cfg.UploadDir); len(entries) != 0 {
		t.Errorf("refused image left on disk: %v", entries)
	}
}
