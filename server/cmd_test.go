package main

import "testing"

// S-007 : `colec-server seed` ne s'exécute qu'avec SEED_ON_EMPTY=true (jamais par
// mégarde sur une base de production, où la variable vaut false ou manque).
func TestSeedConfirmed(t *testing.T) {
	for value, want := range map[string]bool{"": false, "false": false, "0": false, "yes": false, "true": true, "1": true, "TRUE": true} {
		t.Setenv("SEED_ON_EMPTY", value)
		if got := seedConfirmed(); got != want {
			t.Errorf("SEED_ON_EMPTY=%q: got %v, want %v", value, got, want)
		}
	}
	t.Setenv("SEED_ON_EMPTY", "")
	t.Setenv("DATABASE_URL", "postgres://unused")
	if code := runSeed(); code != 2 {
		t.Errorf("runSeed without confirmation: exit %d, want 2", code)
	}
}
