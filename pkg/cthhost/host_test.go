package cthhost_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
	"github.com/JamesPagetButler/qbp-systema/pkg/cthhost"
)

// the canonical QBP v0.3 fixture lives with the bookkeeper package.
const fixture = "../bookkeeper/testdata/confluent-trust-inventory-v5_3.v0.3.json"

func readFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

func TestSeedIfAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "qbp-data", "inventory.json")
	data := readFixture(t)

	seeded, err := cthhost.SeedIfAbsent(path, data)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if !seeded {
		t.Fatal("first SeedIfAbsent must seed (file absent)")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("seeded file missing: %v", err)
	}

	// Second call must NOT re-seed — existing runtime state is preserved.
	seeded2, err := cthhost.SeedIfAbsent(path, []byte("DIFFERENT"))
	if err != nil {
		t.Fatalf("seed2: %v", err)
	}
	if seeded2 {
		t.Fatal("second SeedIfAbsent must not overwrite an existing inventory")
	}
	got, _ := os.ReadFile(path)
	if string(got) == "DIFFERENT" {
		t.Fatal("existing inventory was overwritten — runtime state lost")
	}
}

func TestOpenHealthQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	if _, err := cthhost.SeedIfAbsent(path, readFixture(t)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	h, err := cthhost.Open(path, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer h.Close()

	hr := h.Health()
	if hr.AnchorCount != 141 {
		t.Errorf("AnchorCount = %d, want 141", hr.AnchorCount)
	}
	if !hr.Healthy || hr.RhoNet <= 1.0 {
		t.Errorf("ρ_net = %.4f, want > 1.0 (theory must compress)", hr.RhoNet)
	}

	if a, ok := h.Query("PROOF-hurwitz"); !ok {
		t.Error("PROOF-hurwitz not found")
	} else if a.ProvenanceKind != model.ProvenanceKindTheoryExternal {
		t.Errorf("PROOF-hurwitz provenance_kind = %v, want theory-external", a.ProvenanceKind)
	}
	if _, ok := h.Query("NOPE-does-not-exist"); ok {
		t.Error("nonexistent anchor reported as found")
	}
}

// TestEmitterSeam verifies the OnAnchorChange hook fires on append — the seam
// that becomes the producer/emit-into-shared-CTH path at federation.
func TestEmitterSeam(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	if _, err := cthhost.SeedIfAbsent(path, readFixture(t)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	fired := make(chan string, 1)
	hooks := &store.Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			if before == nil {
				fired <- after.ID
			}
		},
	}
	h, err := cthhost.Open(path, hooks)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer h.Close()

	err = h.AppendAnchor(model.Anchor{
		ID:              "TEST-cthhost-emit",
		Name:            "cthhost emitter-seam test",
		Tier:            1,
		Status:          model.StatusUntested,
		ProvenanceKind:  model.ProvenanceKindHypothesis,
		PredictionChain: []string{},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	select {
	case id := <-fired:
		if id != "TEST-cthhost-emit" {
			t.Errorf("hook fired for %q, want TEST-cthhost-emit", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnAnchorChange (emitter seam) did not fire within 2s")
	}
}
