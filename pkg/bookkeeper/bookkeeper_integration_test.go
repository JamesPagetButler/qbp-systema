package bookkeeper_test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/JamesPagetButler/confluent-trust/compute"
	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/report"
	"github.com/JamesPagetButler/confluent-trust/store"
	"github.com/JamesPagetButler/qbp-systema/pkg/bookkeeper"
	"github.com/stretchr/testify/require"
)

// seedTestInventory copies the canonical QBP v0.3 fixture to a temp file
// and returns its path. The temp file is cleaned up by t.Cleanup.
func seedTestInventory(t *testing.T) string {
	t.Helper()
	src, err := os.Open("testdata/confluent-trust-inventory-v5_3.v0.3.json")
	require.NoError(t, err)
	defer src.Close()

	dst, err := os.CreateTemp(t.TempDir(), "qbp-inventory-*.json")
	require.NoError(t, err)
	_, err = io.Copy(dst, src)
	require.NoError(t, err)
	require.NoError(t, dst.Close())
	return dst.Name()
}

// findByID returns the anchor with the given ID from snap, or nil.
func findByID(snap model.Inventory, id string) *model.Anchor {
	for i := range snap.Anchors {
		if snap.Anchors[i].ID == id {
			return &snap.Anchors[i]
		}
	}
	return nil
}

// TestQBPInventoryLoads verifies the canonical QBP v0.3 fixture loads
// cleanly and key structural metrics are in expected ranges.
//
// Metric baselines from cth-implementor cth-qbp-live-testing seq=3
// (2026-05-29, cth analyse against confluent-trust-inventory-v5_3.v0.3.json).
// ρ_net > 1.0 is the load-bearing regression guard: if it drops below 1.0
// the theory has lost compression — something structural broke in the
// anchor graph.
func TestQBPInventoryLoads(t *testing.T) {
	path := seedTestInventory(t)

	li, err := bookkeeper.Open(path, nil)
	require.NoError(t, err)
	defer li.Close()

	snap := li.Snapshot()

	// Structural counts — fixture is stable; exact matches expected.
	require.Equal(t, 141, len(snap.Anchors), "anchor count")

	tier1, tier3 := 0, 0
	for _, a := range snap.Anchors {
		switch a.Tier {
		case 1:
			tier1++
		case 3:
			tier3++
		}
	}
	require.Equal(t, 46, tier1, "tier-1 anchor count")
	require.Equal(t, 26, tier3, "tier-3 anchor count")

	// ρ_net > 1.0 is the load-bearing invariant: theory must compress.
	// Pass nil axiomEntropy to use default (zero axiom-entropy baseline).
	fa := report.RunFullAnalysis(snap, nil)
	require.Greater(t, fa.NetRho, 1.0,
		"ρ_net must be > 1.0 (theory compresses over inputs)")

	// Coherence ratio sanity — Crawl-era QBP is expected in 0.2–0.5 range.
	require.Greater(t, fa.CoherenceRatio, 0.0, "coherence ratio must be positive")
}

// TestQBPInventoryKnownAnchor validates that a specific well-known anchor
// loads with the expected v0.3 provenance_kind after the Phase 0 C migration.
//
// PROOF-hurwitz was migrated from provenance T to theory-external (Hurwitz 1898)
// by qbp-implementor in Phase 0 C (PR #458 commit 8795d8d).
func TestQBPInventoryKnownAnchor(t *testing.T) {
	path := seedTestInventory(t)

	li, err := bookkeeper.Open(path, nil)
	require.NoError(t, err)
	defer li.Close()

	snap := li.Snapshot()
	anchor := findByID(snap, "PROOF-hurwitz")
	require.NotNil(t, anchor, "PROOF-hurwitz must exist in fixture")

	// Use the typed constant — NOT string(anchor.ProvenanceKind).
	// ProvenanceKind is uint8; string(uint8) produces a byte-value char,
	// not the canonical JSON string. (cth-implementor cth-qbp-live-testing seq=3)
	require.Equal(t, model.ProvenanceKindTheoryExternal, anchor.ProvenanceKind,
		"PROOF-hurwitz must be theory-external (Hurwitz 1898 citation)")
	require.Equal(t, "Hurwitz 1898", anchor.TheoryCitation,
		"PROOF-hurwitz must cite Hurwitz 1898")
}

// TestQBPInventoryLiveRoundTrip is the end-to-end Crawl-phase smoke test.
// It opens the QBP inventory as a LiveInventory (JSON backend), appends a
// synthetic test anchor, verifies OnAnchorChange fires, and confirms the
// new anchor appears in Snapshot.
//
// Walk-phase transition: only bookkeeper.Open changes to
// store.OpenLiveInventoryWyrd(graph, "QBP", hooks) per CTH design §7 /
// CTH #87. All assertions below are backend-agnostic.
func TestQBPInventoryLiveRoundTrip(t *testing.T) {
	path := seedTestInventory(t)

	hookFired := make(chan *model.Anchor, 1)
	hooks := &store.Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			if before == nil {
				// AppendAnchor fires with before==nil per live-inventory-api.md §2.1.
				hookFired <- after
			}
		},
	}

	li, err := bookkeeper.Open(path, hooks)
	require.NoError(t, err)
	defer li.Close()

	// Baseline count before write.
	snapBefore := li.Snapshot()
	countBefore := len(snapBefore.Anchors)

	// Append synthetic test anchor.
	// StatusUntested is correct: untested anchors are excluded from the
	// R_c numerator so they don't skew coherence metrics.
	// ProvenanceKindHypothesis satisfies Invariant 4 (no proof_* fields).
	// (cth-implementor cth-qbp-live-testing seq=3, 2026-05-29)
	testAnchor := model.Anchor{
		ID:              "TEST-qbp-systema-smoke",
		Name:            "qbp-systema bookkeeper smoke test anchor",
		Tier:            1,
		Status:          model.StatusUntested,
		ProvenanceKind:  model.ProvenanceKindHypothesis,
		PredictionChain: []string{},
	}
	require.NoError(t, li.AppendAnchor(testAnchor))

	// Hook must fire within 2s.
	select {
	case appended := <-hookFired:
		require.Equal(t, "TEST-qbp-systema-smoke", appended.ID)
	case <-time.After(2 * time.Second):
		t.Fatal("OnAnchorChange hook did not fire within 2s after AppendAnchor")
	}

	// Snapshot must reflect new anchor.
	snapAfter := li.Snapshot()
	require.Equal(t, countBefore+1, len(snapAfter.Anchors),
		"anchor count must increment by 1 after AppendAnchor")
	require.NotNil(t, findByID(snapAfter, "TEST-qbp-systema-smoke"),
		"newly appended anchor must appear in Snapshot")
}

// TestQBPInventoryNetCompressionStable is a regression guard for the
// NetCompression primitive against the canonical QBP fixture.
//
// This test is separate from TestQBPInventoryLoads so a compute regression
// can be distinguished from a load regression in CI output.
func TestQBPInventoryNetCompressionStable(t *testing.T) {
	path := seedTestInventory(t)
	li, err := bookkeeper.Open(path, nil)
	require.NoError(t, err)
	defer li.Close()

	snap := li.Snapshot()
	netRho, detail := compute.NetCompression(snap, nil)

	// ρ_net > 1.0 is the primary invariant.
	require.Greater(t, netRho, 1.0, "ρ_net must exceed 1.0")

	// GrossRho >= NetRho: gross always >= net.
	require.GreaterOrEqual(t, detail.GrossRho, netRho,
		"gross compression ratio must be >= net")
}
