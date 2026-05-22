// Package stance is the QBP Type-Node match library. It loads the 22-Type-Node
// QBP v0.2 Stance vocabulary from `configs/scope-nodes.yaml` conceptual scope
// nodes at daemon startup and provides case-insensitive substring matching
// against text (arXiv title + abstract per W5.2; arbitrary text per W4.3).
//
// Per QBP federation tenancy v0.2 §1 Stance Type-Nodes:
//
//   §1.1 Foundational algebraic (11):
//     qbp:hamilton-product, qbp:quaternion-conjugation, qbp:hopf-locale,
//     qbp:su2-double-cover, qbp:hurwitz-norm-multiplicativity,
//     qbp:octonion-non-associativity, qbp:sedenion-zero-divisor,
//     qbp:g2-holonomy, qbp:fano-aut-pgl27, qbp:fano-stab-24,
//     qbp:spectral-triple, qbp:cd-tower-zeta-numerator
//
//   §1.2 Theoretical bridges (7+DM-fork):
//     qbp:gw-em-coincidence, qbp:slow-slip-tidal-coupling,
//     qbp:mixed-species-ion-fidelity, qbp:nv-center-fidelity,
//     qbp:topological-materials-q1-q10, qbp:kitaev-z2-gauge,
//     qbp:spectral-action-entropy,
//     qbp:dm-branch-a-modified-gravity, qbp:dm-branch-b-particle-or-field-extension,
//     qbp:dm-axion, qbp:dm-wimp, qbp:dm-sidm, qbp:dm-pbh,
//     qbp:dm-sterile-neutrino, qbp:dm-fimp
//
//   §1.3 Predictive bridges (5+fu-from-hawking):
//     qbp:cascadia-tremor-onset-24h, qbp:tidal-coupling-norm-threshold,
//     qbp:loon-watershed-correlation, qbp:gw-grb-joint-detection-rate,
//     qbp:fidelity-asymmetry-velocity-correlated,
//     qbp:fu-from-hawking-time-reverse
//
// Each Type-Node has aliases (canonical-name expansions) that match against
// natural-language text. E.g. "qbp:spectral-triple" matches "spectral triple"
// (case-insensitive); "qbp:cd-tower-zeta-numerator" matches "Cayley-Dickson
// tower" OR "zeta moments"; etc.
//
// CTH anchor mapping is loaded from the scope-nodes.yaml conceptual-node
// definitions at startup. Each matched Type-Node resolves to ≥1 CTH anchor
// ID per `archive/cth-inventory/confluent-trust-inventory-v5_3.json`.
//
// # Status
//
// SKELETON STAGED IN /tmp/qbp-sprint2-staging/ — committed to qbp-systema
// at Sprint 2 kickoff per Phase 2 of /home/prime/.claude/plans/
// misty-chasing-clover.md.
package stance

import (
	"strings"
	"sync"
)

// typeNodeAliases is the canonical alias table — natural-language synonyms
// that should match the same qbp:* Type-Node when scanning arXiv text.
// Maintained synchronously with tenancy v0.2 §1.
var typeNodeAliases = map[string][]string{
	// §1.1 Foundational algebraic
	"qbp:hamilton-product":              {"hamilton product", "quaternion multiplication"},
	"qbp:quaternion-conjugation":        {"quaternion conjugation", "quaternion conjugate"},
	"qbp:hopf-locale":                   {"hopf locale", "hopf coordinate", "hopf fibration"},
	"qbp:su2-double-cover":              {"su(2) double cover", "su2 double cover", "spin double cover"},
	"qbp:hurwitz-norm-multiplicativity": {"hurwitz norm", "norm multiplicativity"},
	"qbp:octonion-non-associativity":    {"octonion non-associativity", "non-associative octonion"},
	"qbp:sedenion-zero-divisor":         {"sedenion zero divisor", "sedenion zero-divisor", "cawagas", "moreno 1998"},
	"qbp:g2-holonomy":                   {"g2 holonomy", "g_2 holonomy", "exceptional g2"},
	"qbp:fano-aut-pgl27":                {"fano automorphism", "pgl(2,7)", "fano plane automorphism"},
	"qbp:fano-stab-24":                  {"fano stabiliser", "fano stabilizer"},
	// Note (PR #1 Gemini F3): dropped "noncommutative geometry" from
	// qbp:spectral-triple aliases — too broad; would over-trigger on
	// arXiv abstracts mentioning NCG without QBP-specific context.
	// "connes triple" + "spectral triple" remain canonical.
	"qbp:spectral-triple":               {"spectral triple", "connes triple"},
	"qbp:cd-tower-zeta-numerator":       {"cayley-dickson tower", "zeta moments", "cd tower"},

	// §1.2 Theoretical bridges
	"qbp:gw-em-coincidence":             {"gravitational wave em counterpart", "gw-em coincidence", "multimessenger"},
	"qbp:slow-slip-tidal-coupling":      {"slow slip tidal", "tidal coupling slow earthquake"},
	"qbp:mixed-species-ion-fidelity":    {"mixed-species trapped-ion", "mixed species ion fidelity"},
	"qbp:nv-center-fidelity":            {"nv center", "nitrogen-vacancy center", "nv-center fidelity"},
	"qbp:topological-materials-q1-q10":  {"bi2se3", "matbg", "magic angle bilayer graphene", "alpha-rucl3", "α-rucl3", "topological insulator"},
	"qbp:kitaev-z2-gauge":               {"kitaev z2", "kitaev gauge", "non-abelian braid"},
	"qbp:spectral-action-entropy":       {"spectral action entropy", "ccvs entropy", "von neumann entropy spectral"},
	"qbp:dm-branch-a-modified-gravity":  {"modified gravity dark matter", "mond"},
	"qbp:dm-branch-b-particle-or-field-extension": {"dark matter particle", "dark matter field extension"},
	"qbp:dm-axion":                      {"axion dark matter", "axion-like particle"},
	"qbp:dm-wimp":                       {"wimp dark matter", "weakly interacting massive particle"},
	"qbp:dm-sidm":                       {"self-interacting dark matter", "sidm"},
	"qbp:dm-pbh":                        {"primordial black hole", "pbh dark matter"},
	"qbp:dm-sterile-neutrino":           {"sterile neutrino dark matter"},
	"qbp:dm-fimp":                       {"feebly interacting massive particle", "fimp"},

	// §1.3 Predictive bridges
	"qbp:cascadia-tremor-onset-24h":     {"cascadia tremor", "episodic tremor and slip", "ets cascadia"},
	"qbp:tidal-coupling-norm-threshold": {"tidal coupling norm", "holon norm"},
	"qbp:loon-watershed-correlation":    {"squam lake", "loon vocalization"},
	"qbp:gw-grb-joint-detection-rate":   {"gw-grb joint detection", "ligo-fermi coincidence"},
	"qbp:fidelity-asymmetry-velocity-correlated": {"velocity-correlated fidelity asymmetry", "trapped-ion fidelity asymmetry"},
	"qbp:fu-from-hawking-time-reverse":  {"hawking time-reverse", "time-reversed hawking decay"},
}

// matcher is package state populated by LoadFromScopeNodes at startup;
// guarded by mu for concurrent safety though Load is typically called once.
var (
	mu             sync.RWMutex
	loadedTypeNodes []string                // all loaded qbp:* IDs
	typeToAnchors  map[string][]string      // qbp:foo -> CTH anchor IDs (from scope-nodes.yaml)
)

// LoadFromScopeNodes initialises the matcher state from the scope-nodes.yaml
// loaded by Contextus scope-loader. Called once at daemon startup; safe to
// call again on config reload (replaces in-place).
//
// scopeNodes is the parsed conceptual scope nodes (id + type_nodes list +
// CTH anchor refs). Per Contextus #9 schema.
//
// STUB: signature placeholder pending Contextus Go impl. Final signature
// firms up post-W3.
func LoadFromScopeNodes(scopeNodes any) {
	// TODO(W3 dependency): walk scope nodes, extract type_nodes lists,
	// build loadedTypeNodes + typeToAnchors map. For now, seed with the
	// alias keys so Match works against the v0.2 vocabulary.
	mu.Lock()
	defer mu.Unlock()
	loadedTypeNodes = make([]string, 0, len(typeNodeAliases))
	typeToAnchors = make(map[string][]string)
	for tn := range typeNodeAliases {
		loadedTypeNodes = append(loadedTypeNodes, tn)
		// CTH anchor mapping is currently empty — populated from
		// scope-nodes.yaml conceptual-node CTH-anchor refs at runtime.
		typeToAnchors[tn] = nil
	}
}

// TypeNodes returns the loaded Type-Node ID list (read-only snapshot).
func TypeNodes() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, len(loadedTypeNodes))
	copy(out, loadedTypeNodes)
	return out
}

// Match scans `text` against the loaded Type-Node alias table and returns
// the matched Type-Node IDs (deduplicated; in load order). Case-insensitive
// substring matching.
//
// Per tenancy v0.2 §4.1 filter spec: "Stance Type-Node match in title/abstract".
//
// Note (PR #1 Gemini F4): the typeNodeAliases values are maintained as
// lowercase-canonical strings (see the var declaration above); no per-alias
// `strings.ToLower(alias)` call is needed inside the loop. Only the input
// text gets lowercased once.
//
// Note (PR #1 Red Team G2): the alias values are dual-source-of-truth with
// configs/scope-nodes.yaml's `type_nodes` lists. The matcher's
// LoadFromScopeNodes hook is the bridge that derives `loadedTypeNodes` +
// `typeToAnchors` from the YAML at startup, but the alias TEXT mapping
// (qbp:X → ["natural-language match 1", "match 2", ...]) currently lives
// only in this file. When Contextus scope-loader Go impl ships (W3), the
// alias values should migrate into scope-nodes.yaml conceptual-node entries
// (e.g., as a `match_aliases:` field) so the YAML is the single source of
// truth. Until then, schema drift between this file and scope-nodes.yaml
// is a known risk — caught by future TestAliasesMatchScopeNodesYAML test
// (not yet added; W5.2 follow-up).
func Match(text string) []string {
	if text == "" {
		return nil
	}
	lower := strings.ToLower(text)

	mu.RLock()
	defer mu.RUnlock()

	var matched []string
	seen := make(map[string]struct{})
	for _, tn := range loadedTypeNodes {
		aliases := typeNodeAliases[tn]
		for _, alias := range aliases {
			// alias is already lowercase per the var declaration's discipline
			if strings.Contains(lower, alias) {
				if _, ok := seen[tn]; !ok {
					matched = append(matched, tn)
					seen[tn] = struct{}{}
				}
				break // one match per Type-Node suffices
			}
		}
	}
	return matched
}

// AnchorsForTypeNodes resolves matched Type-Node IDs to their CTH anchor
// references per scope-nodes.yaml conceptual-node CTH-anchor map. Returns
// deduplicated anchor IDs.
//
// Currently returns empty until scope-nodes.yaml populates typeToAnchors
// via LoadFromScopeNodes — pending W3 Contextus scope-loader Go impl.
func AnchorsForTypeNodes(typeNodes []string) []string {
	mu.RLock()
	defer mu.RUnlock()

	var anchors []string
	seen := make(map[string]struct{})
	for _, tn := range typeNodes {
		for _, a := range typeToAnchors[tn] {
			if _, ok := seen[a]; !ok {
				anchors = append(anchors, a)
				seen[a] = struct{}{}
			}
		}
	}
	return anchors
}
