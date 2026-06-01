//go:build substrate

// Command qbp-scout-daemon is the QBP-tenant scout daemon entrypoint.
//
// Per QBP federation tenancy v0.2 §7 Bootstrap Sequence + Wyrd PR #54 §11.3
// migration step 3:
//
//	Walk-α QBP-side: `qbp-systema/cmd/qbp-scout-daemon/main.go` instantiates
//	`wyrd/scoutd.NewDaemon`, registers QBP's arXiv scout, runs.
//
// Sub-issue closure: QBP #441 (W1.2 — Go module bootstrap; entrypoint stub here).
// Followed by W4.3 (configs validate; QBP #447), W5.1/W5.2 (arXiv Fetch+Transform
// registration; QBP #448/#449), W6.1a/b + W6.2 (M1/M2 verification; QBP #450-#452).
//
// # Status
//
// SKELETON STAGED IN /tmp/qbp-sprint2-staging/ — committed to qbp-systema
// at Sprint 2 kickoff per Phase 2 of /home/prime/.claude/plans/
// misty-chasing-clover.md. Won't compile until:
//
//   - wyrd/scoutd impl PR merges (Wyrd PR #54 → impl PR; tracked QBP #443)
//   - Contextus scope-loader Go impl ships (Contextus #9; tracked QBP #444)
//   - bma-systema PR #178 merges (bma compute-manifest current; consumed at startup)
//
// # Startup sequence (intended)
//
//  1. Read $BMA_FEDERATION_ROOT/manifest/CURRENT via `bma compute-manifest current`
//     (operator-facing wrapper around wyrd/model.LoadComputeManifest per bma-systema PR #178).
//     Verify substrate.commit_sha matches qbp-systema's vendored wyrd dependency
//     SHA pin. If mismatch: emit NT_SIGNAL referent_kind=substrate-drift; continue
//     (warning-only at v0.1; hard-fail at v0.2 per Walk-α discipline).
//
//  2. Load `configs/scope-nodes.yaml` via Contextus scope-loader Go impl
//     (Contextus #9). Populates Wyrd hypergraph with NT_SCOPE_PHYSICAL +
//     NT_SCOPE_CONCEPTUAL hyperedges per tenancy v0.2 §3.
//
//  3. Load `configs/scouts.yaml` via Wyrd scoutd config-loader (Wyrd PR #54 §4).
//     Validates scout names + cadences + agent classes; matches against the
//     programmatically registered Fetch/Transform pairs below.
//
//  4. Construct wyrd/scoutd.Daemon with a *model.Graph reference to the
//     Wyrd substrate (consumes Compute Manifest substrate SHA from step 1).
//
//  5. Register each scout with daemon.Register(scout). For v0.1 this is just
//     the arXiv daily-batch scout; additional scouts (qbp.ligo.event-driven,
//     qbp.usgs-cascadia.hourly, qbp.pnsn-ets.daily, qbp.fermi-gbm.event-driven,
//     qbp.cross-domain.reins) land in subsequent W5.x sub-issues post-v0.1.
//
//  6. daemon.Run(ctx) blocks until ctx cancellation. Clean shutdown signals
//     each scout via its own ctx; 30s grace period per Wyrd PR #54 §3.
//
// # Compute Manifest substrate-drift detection
//
// Per my Wyrd PR #58 review F4 + bma-systema PR #178 consumer-side review:
// the daemon invokes `bma compute-manifest current` at startup, parses
// stdout to extract substrate.commit_sha, and compares against the SHA
// pinned in this binary's go.mod (set by Wyrd dependency at build time).
// Drift indicates the operator's federation substrate has moved since this
// daemon was compiled — the appropriate response is to log + emit a
// substrate-drift NT_SIGNAL and continue (qbp-scout-daemon's signals
// still flow into Wyrd graph regardless; observer-side handling decides
// whether to act on the drift).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	contextusconfig "github.com/JamesPagetButler/Contextus/pkg/scopeconfig"
	wyrdmodel "github.com/JamesPagetButler/wyrd/model"
	"github.com/JamesPagetButler/wyrd/scoutd"

	"github.com/JamesPagetButler/qbp-systema/pkg/scouts/arxiv"
	"github.com/JamesPagetButler/qbp-systema/pkg/stance"
)

// build-time variables; populated via -ldflags by qbp-systema CI.
var (
	pinnedSubstrateSHA = "unknown" // qbp-systema's vendored wyrd commit SHA at compile time
	buildVersion       = "dev"
)

// flags
var (
	flagLoadConfigsOnly = flag.Bool("load-configs-only", false, "M1 mode: load configs into Wyrd graph; do not start scout goroutines")
	flagTriggerOnce     = flag.String("trigger-once", "", "M2 mode: dispatch one cycle of the named scout immediately; exit after completion")
	flagFederationRoot  = flag.String("federation-root", "", "Override $BMA_FEDERATION_ROOT (default: read from env or fallback to ~/Documents/Wyrd per bma-systema PR #178)")
	flagScopeNodesYAML  = flag.String("scope-nodes", "configs/scope-nodes.yaml", "Path to scope-nodes.yaml")
	flagScoutsYAML      = flag.String("scouts", "configs/scouts.yaml", "Path to scouts.yaml")
)

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "qbp-scout-daemon: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	fmt.Printf("qbp-scout-daemon %s (pinned wyrd SHA=%s)\n", buildVersion, pinnedSubstrateSHA)

	// Step 1: verify Compute Manifest substrate SHA matches pinned dependency.
	if err := verifyComputeManifestSubstrate(ctx); err != nil {
		// v0.1: warning only; emit NT_SIGNAL substrate-drift and continue.
		fmt.Fprintf(os.Stderr, "WARNING: compute-manifest substrate drift detected: %v\n", err)
		// TODO: emit NT_SIGNAL with referent_kind=substrate-drift via wyrd graph
	}

	// Step 2: load scope-nodes.yaml into Wyrd graph.
	g, err := contextusconfig.LoadScopeNodes(*flagScopeNodesYAML)
	if err != nil {
		return fmt.Errorf("load scope-nodes: %w", err)
	}
	fmt.Printf("loaded %d scope nodes from %s\n", len(g.ScopeNodes()), *flagScopeNodesYAML)

	// M1 mode: stop here without starting scout goroutines.
	if *flagLoadConfigsOnly {
		fmt.Println("M1 mode (--load-configs-only): scope-nodes loaded; daemon NOT starting scout goroutines")
		return nil
	}

	// Step 3+4: initialise stance vocabulary + scout daemon.
	stance.LoadFromScopeNodes(g.ScopeNodes())
	daemon := scoutd.NewDaemon(wyrdmodel.GraphFromContextus(g))

	// Step 5: register scouts (v0.1: arXiv daily-batch only).
	arxivScout := scoutd.Scout{
		Name:       "qbp.arxiv.daily",
		Cadence:    24 * time.Hour,
		Fetch:      arxiv.Fetch,
		Transform:  arxiv.Transform,
		AgentClass: "scout",
	}
	if err := daemon.Register(arxivScout); err != nil {
		return fmt.Errorf("register arxiv scout: %w", err)
	}

	// M2 mode: trigger one cycle of the named scout and exit.
	if *flagTriggerOnce != "" {
		fmt.Printf("M2 mode (--trigger-once=%s): dispatching one cycle...\n", *flagTriggerOnce)
		return daemon.TriggerOnce(ctx, *flagTriggerOnce)
	}

	// Step 6: Run blocks until ctx cancellation.
	fmt.Println("qbp-scout-daemon running; SIGINT/SIGTERM to shut down")
	return daemon.Run(ctx)
}

// verifyComputeManifestSubstrate invokes `bma compute-manifest current`
// per bma-systema PR #178; parses stdout substrate.commit_sha; compares
// against pinnedSubstrateSHA. Returns nil on match; error on drift.
//
// v0.1 STUB: implementation pending bma-systema PR #178 merge + qbp-systema
// integration test. The `bma` binary path comes from PATH or
// $BMA_BINARY env var override.
func verifyComputeManifestSubstrate(ctx context.Context) error {
	// TODO(W5.x): exec.Command("bma", "compute-manifest", "current");
	// parse the formatManifestSummary output; extract substrate.commit_sha;
	// compare to pinnedSubstrateSHA. Treat any non-zero exit + parse error
	// as warning-only at v0.1 per Wyrd PR #58 F4 framing.
	_ = ctx
	if pinnedSubstrateSHA == "unknown" {
		return fmt.Errorf("build-time substrate SHA not set; rebuild with -ldflags '-X main.pinnedSubstrateSHA=<sha>'")
	}
	return nil
}