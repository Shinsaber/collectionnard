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
