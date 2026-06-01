//go:build substrate

// Package arxiv — Transform half of the QBP arXiv daily-batch scout.
//
// Sub-issue closure: QBP #449 (W5.2).
//
// # Contract
//
// Transform implements the wyrd/scoutd.Scout.Transform interface (Wyrd
// PR #54 §2):
//
//	type Scout struct {
//	    ...
//	    Transform func(raw []byte) ([]model.Node, []model.Hyperedge, error)
//	    ...
//	}
//
// Consumes the Atom XML bytes Fetch returned; parses to []AtomEntry; runs
// Stance Type-Node match against title + abstract per tenancy v0.2 §4.1
// "filter: Stance Type-Node match in title/abstract"; for each matched
// entry mints a NT_SIGNAL hyperedge with provenance per the federation
// reflex pathway.
//
// All-or-nothing semantic per Wyrd PR #54 §2 contract: partial transforms
// not allowed; failure returns error.
//
// # Status
//
// SKELETON STAGED IN /tmp/qbp-sprint2-staging/ — committed to qbp-systema
// at Sprint 2 kickoff per Phase 3 of /home/prime/.claude/plans/
// misty-chasing-clover.md. Won't compile until:
//
//   - wyrd/scoutd impl PR merges (Wyrd PR #54 → impl PR; tracked QBP #443)
//   - wyrd/model package provides Node + Hyperedge types
//   - qbp-systema/pkg/stance package provides TypeNodes list (loaded from configs/scope-nodes.yaml at startup)
//
// # NT_SIGNAL hyperedge schema
//
// Per tenancy v0.2 §4.1 output spec + my Wyrd PR #58 review F4 (consumer
// attribution chain back to QBP scout origin):
//
//	NT_SIGNAL {
//	    id: "sig.qbp.arxiv.<arxiv_id>.<timestamp>"
//	    source: "scout"
//	    scout_id: "qbp.arxiv.daily"
//	    agent_class: "scout"
//	    referent_kind: "scalar"
//	    provenance: {
//	        arxiv_id: <id>
//	        matched_type_nodes: [...]
//	        cth_anchor: "archive/cth-inventory/confluent-trust-inventory-v5_3.json#<anchor_id>"
//	        published_at: <ISO 8601>
//	        title: <paper title>
//	        authors: [<author names>]
//	        categories: [<arxiv categories>]
//	    }
//	}
//
// # CTH anchor provenance
//
// Per `archive/cth-inventory/confluent-trust-inventory-v5_3.json` (141
// anchors; tracked via QBP PR #422). Each matched Type-Node maps to one or
// more CTH anchor IDs. The mapping is loaded at qbp-scout-daemon startup
// from `configs/scope-nodes.yaml` via Contextus scope-loader.
//
// v0.3 note: post-confluent-trust #71 ratification, NT_SIGNAL provenance
// gains `provenance_kind: theory|experiment|...` field per the decomposed
// schema. v0.2 scout cycles use the v0.2 CTH baseline; migration to v0.3
// is mechanical via `cth migrate v0.2 -> v0.3` (cth-implementor deliverable).
package arxiv

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	wyrdmodel "github.com/JamesPagetButler/wyrd/model"

	"github.com/JamesPagetButler/qbp-systema/pkg/stance"
)

// Transform implements wyrd/scoutd.Scout.Transform. Returns the minted
// NT_SIGNAL hyperedges (one per matched arXiv entry) + any new nodes
// (none in v0.1 — all NT_SIGNAL hyperedges reference existing scope nodes
// loaded at startup; no new nodes minted by the arXiv scout).
func Transform(raw []byte) ([]wyrdmodel.Node, []wyrdmodel.Hyperedge, error) {
	if len(raw) == 0 {
		return nil, nil, ErrEmptyInput
	}

	entries, err := parseAtomFeedsConcatenated(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse atom feeds: %w", err)
	}

	// Scan each entry against the Stance vocabulary. The matcher is
	// case-insensitive substring + canonical-name alias per stance.Match.
	var hyperedges []wyrdmodel.Hyperedge
	for _, entry := range entries {
		matched := stance.Match(entry.Title + " " + entry.Summary)
		if len(matched) == 0 {
			continue
		}

		// Determine CTH anchor IDs from matched Type-Nodes. Loaded at
		// startup from configs/scope-nodes.yaml conceptual scope nodes.
		anchorIDs := stance.AnchorsForTypeNodes(matched)

		// Mint one NT_SIGNAL per arXiv entry per tenancy §4.1
		// referent_kind=scalar; provenance carries the matched Type-Node
		// list + originating arXiv metadata.
		he, err := mintNTSignal(entry, matched, anchorIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("mint NT_SIGNAL for arxiv:%s: %w", entry.ID, err)
		}
		hyperedges = append(hyperedges, he)
	}

	// All-or-nothing per Wyrd #54 §2 contract: no entries doesn't error
	// (genuine no-match cycle), but parse errors above already returned.
	return nil, hyperedges, nil
}

// parseAtomFeedsConcatenated handles Fetch's concatenated Atom output
// (per-source `<feed>` blocks separated by XML comments). Each
// `<feed>...</feed>` block parses independently; entries from all blocks
// merge into one returned slice.
func parseAtomFeedsConcatenated(raw []byte) ([]AtomEntry, error) {
	var entries []AtomEntry
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "feed" {
			continue
		}
		var feed AtomFeed
		if err := dec.DecodeElement(&feed, &se); err != nil {
			return nil, fmt.Errorf("decode feed: %w", err)
		}
		entries = append(entries, feed.Entries...)
	}
	return entries, nil
}

// mintNTSignal builds the Hyperedge representation per the schema in this
// file's package doc. Returns ErrInvalidEntry if the entry lacks required
// fields (id, published).
func mintNTSignal(entry AtomEntry, matched []string, anchorIDs []string) (wyrdmodel.Hyperedge, error) {
	arxivID := arxivIDFromAtomID(entry.ID)
	if arxivID == "" {
		return wyrdmodel.Hyperedge{}, fmt.Errorf("%w: empty arxiv ID in atom entry", ErrInvalidEntry)
	}

	publishedAt, err := time.Parse(time.RFC3339, entry.Published)
	if err != nil {
		return wyrdmodel.Hyperedge{}, fmt.Errorf("%w: invalid published timestamp %q: %v", ErrInvalidEntry, entry.Published, err)
	}

	authors := make([]string, 0, len(entry.Authors))
	for _, a := range entry.Authors {
		authors = append(authors, a.Name)
	}
	categories := make([]string, 0, len(entry.Categories))
	for _, c := range entry.Categories {
		categories = append(categories, c.Term)
	}

	// Hyperedge ID: deterministic from arxiv_id + published timestamp so
	// same paper observed across daemon restarts dedups by ID.
	heID := fmt.Sprintf("sig.qbp.arxiv.%s.%d", strings.ReplaceAll(arxivID, "/", "."), publishedAt.Unix())

	// Hyperedge.Properties carries the provenance map per package doc.
	// Note: wyrd/model.Hyperedge fields are placeholder; exact field
	// names will firm up when Wyrd scoutd impl PR lands. The shape here
	// is intentional: provenance is opaque key-value map.
	he := wyrdmodel.Hyperedge{
		ID:   heID,
		Type: "NT_SIGNAL",
		Properties: map[string]any{
			"source":              "scout",
			"scout_id":            "qbp.arxiv.daily",
			"agent_class":         "scout",
			"referent_kind":       "scalar",
			"matched_type_nodes":  matched,
			"cth_anchor_refs":     anchorIDs,
			"arxiv_id":            arxivID,
			"arxiv_title":         strings.TrimSpace(entry.Title),
			"arxiv_authors":       authors,
			"arxiv_categories":    categories,
			"published_at":        publishedAt.Format(time.RFC3339),
			"cth_baseline":        "archive/cth-inventory/confluent-trust-inventory-v5_3.json",
		},
	}
	return he, nil
}

// arxivIDFromAtomID extracts the arXiv paper ID from an Atom entry id URL.
// arXiv entries use IDs like "http://arxiv.org/abs/2401.12345v1"; this
// returns "2401.12345v1" (the suffix after the last /abs/).
func arxivIDFromAtomID(atomID string) string {
	idx := strings.LastIndex(atomID, "/abs/")
	if idx < 0 {
		return ""
	}
	return atomID[idx+len("/abs/"):]
}

// Errors.
var (
	// ErrEmptyInput is returned when Transform receives no bytes.
	ErrEmptyInput = errors.New("arxiv transform: empty input")
	// ErrInvalidEntry is returned when an Atom entry lacks required fields.
	ErrInvalidEntry = errors.New("arxiv transform: invalid entry")
)