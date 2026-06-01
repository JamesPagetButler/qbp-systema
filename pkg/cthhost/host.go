// Package cthhost is the QBP-tenant active CTH host — the runtime that holds
// the QBP Confluent-Trust inventory live and serves query/health over it.
//
// Role per the multi-tenant federation design (qbp-architecture, cth-qbp-live-testing
// seq=10):
//
//   - Crawl (now): qbp-systema is its own CTH *authority* — this host opens the
//     inventory via store.OpenLiveInventory (the bookkeeper path) and serves it.
//   - Walk/federation: contextus hosts the shared CTH (Wyrd-backed on qbp-cu);
//     qbp-systema becomes a *producer/emitter*, giving up authority over CTH
//     state while keeping participation. The transition seam is the one-line
//     store.OpenLiveInventory → store.OpenLiveInventoryWyrd swap (CTH #87): at
//     Walk the Host's backing LiveInventory points read-through to the shared
//     CTH, and local AppendAnchor becomes an emit-into-shared-CTH.
//
// The OnAnchorChange hook is the emitter seam: in Crawl it can drive local
// reactions (NATS publish per CTH #19); at federation it is where QBP-domain
// assertions flow outward into the contextus-hosted CTH.
package cthhost

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JamesPagetButler/confluent-trust/compute"
	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
	"github.com/JamesPagetButler/qbp-systema/pkg/bookkeeper"
)

// Host wraps a live QBP CTH inventory.
type Host struct {
	li   *store.LiveInventory
	path string
}

// Open loads the QBP inventory at path as a live host. hooks may be nil.
// At Walk this call swaps to store.OpenLiveInventoryWyrd(graph, "QBP", hooks)
// with no change to the Health/Query/Append surface below.
func Open(path string, hooks *store.Hooks) (*Host, error) {
	li, err := bookkeeper.Open(path, hooks)
	if err != nil {
		return nil, fmt.Errorf("cthhost: open %s: %w", path, err)
	}
	return &Host{li: li, path: path}, nil
}

// Close releases the underlying live inventory.
func (h *Host) Close() error { return h.li.Close() }

// SeedIfAbsent writes bundledData to path when path does not yet exist
// (hybrid bundle+volume seeding per cth-implementor's recommendation:
// the canonical inventory ships in the image and seeds the data volume on
// first run; an existing inventory at path is left untouched so runtime
// appends accumulate). Returns true if it seeded.
func SeedIfAbsent(path string, bundledData []byte) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil // already present — leave runtime state intact
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("cthhost: stat %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("cthhost: mkdir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, bundledData, 0o644); err != nil {
		return false, fmt.Errorf("cthhost: seed %s: %w", path, err)
	}
	return true, nil
}

// Query returns the anchor with the given ID, or (nil,false).
func (h *Host) Query(id string) (*model.Anchor, bool) {
	snap := h.li.Snapshot()
	for i := range snap.Anchors {
		if snap.Anchors[i].ID == id {
			a := snap.Anchors[i]
			return &a, true
		}
	}
	return nil, false
}

// HealthReport is a compact liveness + integrity summary of the hosted CTH.
type HealthReport struct {
	AnchorCount int
	RhoNet      float64 // > 1.0 invariant: theory compresses (cth-implementor seq=3)
	TierCounts  map[int]int
	Healthy     bool // RhoNet > 1.0
}

// Health computes the liveness/integrity summary over the current snapshot.
func (h *Host) Health() HealthReport {
	snap := h.li.Snapshot()
	netRho, _ := compute.NetCompression(snap, nil)
	tiers := map[int]int{}
	for _, a := range snap.Anchors {
		tiers[int(a.Tier)]++
	}
	return HealthReport{
		AnchorCount: len(snap.Anchors),
		RhoNet:      netRho,
		TierCounts:  tiers,
		Healthy:     netRho > 1.0,
	}
}

// AppendAnchor adds an anchor to the live inventory. In Crawl this writes to
// the local authoritative copy; at federation (post-swap) this becomes the
// emit-into-shared-CTH path (the producer role). Fires OnAnchorChange.
func (h *Host) AppendAnchor(a model.Anchor) error { return h.li.AppendAnchor(a) }
