// Package bookkeeper provides the QBP-tenant live CTH inventory handle.
//
// At Crawl, the inventory is backed by a JSON file at qbp-data/inventory.json
// (seeded from the canonical QBP v0.3 fixture on first-run). At Walk, the
// single-line constructor swap per CTH design §7 switches to Wyrd:
//
//	// Crawl
//	li, err := bookkeeper.Open("qbp-data/inventory.json", hooks)
//
//	// Walk (after CTH #87 lands OpenLiveInventoryWyrd)
//	li, err := store.OpenLiveInventoryWyrd(graph, "QBP", hooks)
//
// The returned *store.LiveInventory surface (AppendAnchor, UpdateAnchor,
// Snapshot, Close) is identical in both cases.
//
// References:
//   - CTH doc/integration/live-api.md §2 (QBP bookkeeper wiring pattern)
//   - CTH design §7 (Wyrd substrate-swap at Walk)
//   - CTH #87 (OpenLiveInventoryWyrd implementation, Walk-α milestone)
//   - QBP #403 Federation Tenancy v0.2 §5.3
package bookkeeper

import (
	"github.com/JamesPagetButler/confluent-trust/store"
)

// Open loads the QBP programme inventory from path and returns a live
// handle with the supplied hooks. path is typically "qbp-data/inventory.json"
// in the mounted federation data volume.
//
// hooks may be nil. When non-nil, OnAnchorChange fires after every
// successful AppendAnchor or status-field UpdateAnchor, enabling NATS
// publishing per CTH #19.
//
// At Walk this call is replaced by store.OpenLiveInventoryWyrd; the
// AppendAnchor / UpdateAnchor / Snapshot call sites require no changes.
func Open(path string, hooks *store.Hooks) (*store.LiveInventory, error) {
	return store.OpenLiveInventory(path, hooks)
}
