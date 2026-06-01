// Command qbp-cth-host is the QBP-tenant active CTH host.
//
// It opens the QBP Confluent-Trust inventory as a live, queryable instance —
// the Crawl-phase "active version of qbp-systema" that holds the CTH live
// rather than as a static .json. When the foundations fleet lands the verified
// inventory (corrected octonionMul, anchors #474–#479, FORCED/SELECTED tags),
// it is a file-swap into the same path + restart (or hot-reload via the
// OnAnchorChange hook) — the ingest contract (architect cth-qbp-live-testing
// seq=10 step 4).
//
// Usage:
//
//	qbp-cth-host -inventory qbp-data/inventory.json [-seed bundled.json] health
//	qbp-cth-host -inventory qbp-data/inventory.json query PROOF-hurwitz
//
// At Walk the host's backing store swaps to OpenLiveInventoryWyrd (CTH #87)
// and this same binary becomes a producer/emitter into the contextus-hosted
// shared CTH — no change to the command surface.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JamesPagetButler/qbp-systema/pkg/cthhost"
)

func main() {
	inv := flag.String("inventory", "qbp-data/inventory.json", "path to the live QBP CTH inventory JSON")
	seed := flag.String("seed", "", "optional: bundled inventory to seed from if -inventory is absent (hybrid bundle+volume)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: qbp-cth-host -inventory <path> [-seed <path>] <health|query <anchor-id>>")
		os.Exit(2)
	}

	if *seed != "" {
		data, err := os.ReadFile(*seed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "qbp-cth-host: read seed %s: %v\n", *seed, err)
			os.Exit(1)
		}
		if seeded, err := cthhost.SeedIfAbsent(*inv, data); err != nil {
			fmt.Fprintf(os.Stderr, "qbp-cth-host: seed: %v\n", err)
			os.Exit(1)
		} else if seeded {
			fmt.Fprintf(os.Stderr, "qbp-cth-host: seeded %s from %s\n", *inv, *seed)
		}
	}

	h, err := cthhost.Open(*inv, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "qbp-cth-host: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	switch args[0] {
	case "health":
		hr := h.Health()
		status := "DEGRADED"
		if hr.Healthy {
			status = "HEALTHY"
		}
		fmt.Printf("CTH host: %s\n", status)
		fmt.Printf("  anchors: %d\n", hr.AnchorCount)
		fmt.Printf("  ρ_net:   %.4f  (invariant: > 1.0 — theory compresses)\n", hr.RhoNet)
		fmt.Printf("  tiers:   %v\n", hr.TierCounts)
		if !hr.Healthy {
			os.Exit(1) // ρ_net ≤ 1.0 — structural regression in the anchor graph
		}
	case "query":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: qbp-cth-host ... query <anchor-id>")
			os.Exit(2)
		}
		a, ok := h.Query(args[1])
		if !ok {
			fmt.Fprintf(os.Stderr, "anchor %q not found\n", args[1])
			os.Exit(1)
		}
		fmt.Printf("%s\n", a.ID)
		fmt.Printf("  name:           %s\n", a.Name)
		fmt.Printf("  tier:           %d\n", a.Tier)
		fmt.Printf("  status:         %s\n", a.Status)
		fmt.Printf("  provenance_kind: %s\n", a.ProvenanceKind)
		if a.TheoryCitation != "" {
			fmt.Printf("  theory_citation: %s\n", a.TheoryCitation)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (want: health | query)\n", args[0])
		os.Exit(2)
	}
}
