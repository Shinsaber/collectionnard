package handler

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/google/uuid"
)

// S-001 : au plus maxConcurrentThumbnails décodages en vol ; au-delà, acquire
// rend la main sans bloquer (l'original est servi).
func TestThumbGate(t *testing.T) {
	var g thumbGate
	ctx := context.Background()
	for i := 0; i < maxConcurrentThumbnails; i++ {
		if !g.acquire(ctx, time.Second) {
			t.Fatalf("slot %d refused", i+1)
		}
	}
	start := time.Now()
	if g.acquire(ctx, 30*time.Millisecond) {
		t.Fatal("extra slot granted while the gate is full")
	}
	if time.Since(start) > time.Second {
		t.Error("acquire did not give up after the wait")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if g.acquire(cancelled, time.Minute) {
		t.Fatal("slot granted to a cancelled request")
	}
	g.release()
	if !g.acquire(ctx, time.Second) {
		t.Fatal("slot not reusable after release")
	}
}

func TestImageTooLarge(t *testing.T) {
	cases := []struct {
		w, h int
		want bool
	}{
		{600, 900, false},
		{5000, 5000, false}, // 25 Mpx : limite incluse
		{5001, 5000, true},  // juste au-dessus
		{7000, 7000, true},  // bombe PNG unie
		{0, 10, true},       // vide
		{maxImagePixels + 1, 1, true},
		{1 << 40, 1 << 40, true}, // produit qui déborderait un int32
	}
	for _, c := range cases {
		if got := imageTooLarge(image.Config{Width: c.w, Height: c.h}); got != c.want {
			t.Errorf("%dx%d: got %v, want %v", c.w, c.h, got, c.want)
		}
	}
}

// S-004 : fenêtre d'une heure par compte, en fichiers et en octets ; la fenêtre
// suivante repart de zéro.
func TestUploadQuota(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	q := uploadQuota{now: func() time.Time { return now }}
	alice, bob := uuid.New(), uuid.New()

	for i := 0; i < 3; i++ {
		if ok, _ := q.allow(alice, 3, 1000); !ok {
			t.Fatalf("file %d refused", i+1)
		}
		q.add(alice, 100)
	}
	ok, retry := q.allow(alice, 3, 1000)
	if ok || retry != uploadQuotaWindow {
		t.Errorf("4th file: allowed %v, retry %s", ok, retry)
	}
	if ok, _ := q.allow(bob, 3, 1000); !ok {
		t.Error("another account is independent")
	}

	// Octets : refusé dès que la fenêtre est pleine, même sous le nombre de fichiers.
	if ok, _ := q.allow(bob, 100, 150); !ok {
		t.Fatal("bob 2nd file")
	}
	q.add(bob, 200)
	if ok, retry := q.allow(bob, 100, 150); ok || retry <= 0 {
		t.Errorf("bytes exhausted: allowed %v, retry %s", ok, retry)
	}

	now = now.Add(uploadQuotaWindow)
	if ok, _ := q.allow(alice, 3, 1000); !ok {
		t.Error("next window should reset the counters")
	}
	if len(q.users) != 1 {
		t.Errorf("stale windows not swept: %d", len(q.users))
	}
}
