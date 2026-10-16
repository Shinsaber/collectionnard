package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestPurgeUploads(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.UploadDir = t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	write := func(name string, mtime time.Time) {
		path := filepath.Join(cfg.UploadDir, name)
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	write("used.jpg", old)          // referenced
	write("orphan.png", old)        // unused, old: purged
	write("fresh.webp", time.Now()) // unused but within the grace period
	write(".upload-123", old)       // temporary file: ignored
	write("notes.txt", old)         // not an image: ignored
	mock := &testutil.MockStore{
		ReferencedUploadsFn: func(context.Context) ([]string, error) { return []string{"used.jpg", "gone.jpg"}, nil },
	}
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(mock, cfg))
	defer srv.Close()

	purge := func(token, query string) (int, model.UploadPurgeReport) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/uploads/purge"+query, nil)
		req.Header = testutil.AuthHeader(token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		var rep model.UploadPurgeReport
		_ = json.NewDecoder(resp.Body).Decode(&rep)
		return resp.StatusCode, rep
	}
	if status, _ := purge(testutil.UserToken(), "?dry_run=true"); status != http.StatusForbidden {
		t.Errorf("user: %d, want 403", status)
	}

	status, rep := purge(testutil.AdminToken(), "?dry_run=true")
	if status != http.StatusOK || rep.Applied || len(rep.Files) != 1 || rep.Files[0].Filename != "orphan.png" ||
		rep.Referenced != 1 || rep.Recent != 1 || rep.TotalSize != 1 || rep.GraceHours != 24 {
		t.Fatalf("dry run: %d %+v", status, rep)
	}
	if _, err := os.Stat(filepath.Join(cfg.UploadDir, "orphan.png")); err != nil {
		t.Error("dry run must not delete")
	}

	if status, rep = purge(testutil.AdminToken(), ""); status != http.StatusOK || !rep.Applied || len(rep.Files) != 1 {
		t.Fatalf("purge: %d %+v", status, rep)
	}
	entries, _ := os.ReadDir(cfg.UploadDir)
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	if len(left) != 4 {
		t.Errorf("left after purge: %v (only orphan.png must go)", left)
	}
}
