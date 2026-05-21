# Contributing to qbp-systema

This repo participates in the Helpful Engineering federation. Federation-wide conventions are canonical; this doc adds tenant-specific overlay only.

---

## 1. Federation-wide canonical references

Before contributing, read or skim these (in `~/Documents/inter/`):

- `inter/issue-authoring-best-practices.md` — issue shapes (design-surface / ratification / sub-issue / bug-or-incident); §I4 reader-list; closes-when discipline; repo-prefix referencing
- `inter/test-quality-best-practices.md` — test categories; naked-vs-workshop levels; test-plan-in-PR template
- `inter/pr-review-completion-best-practices.md` — qbp-architecture review filter; completion verification; **Federation Rule #7 named-reviewer responsiveness (4h SLA)**
- `inter/code-review-best-practices.md` — six-category code review checklist; gograph + testo tooling guidance
- `inter/github-best-practices.md` — branch protection; CI gates; linear history
- `inter/project-management-best-practices.md` — federation sprint structure; BMA:BADASS dashboard

All conventions in these docs apply to qbp-systema unless this doc explicitly documents a tenant-scoped exception with rationale.

---

## 2. Tenant overview

- **Tenant name:** qbp-systema
- **Tenant role in federation:** Tier T5 federation-integration runtime — QBP-tenant scout daemon that consumes Wyrd `scoutd/` + Contextus scope-loader to expose QBP physics work to the federation's BMA observer layer
- **Primary language(s):** Go (runtime); YAML (tenant configs)
- **Federation phase status:** Crawl-Toddle bridge (cross-ref `~/Documents/inter/workspace-phase-architecture.md`)
- **Companion repo:** `JamesPagetButler/QBP` (physics paper-corpus + Lean proofs + CTH inventory + federation tenancy doc; cross-references to qbp-systema for the runtime side)

---

## 3. Default reviewer-list for qbp-systema issues + PRs

| Persona | Domain | When always included |
|---|---|---|
| @qbp-implementor | Tenant code owner (Integration role) | All qbp-systema PRs |
| @qbp-cu-implementor | Substrate publisher (QBP-CU emulator referenced by Compute Manifest) | All PRs touching `pkg/scouts/` substrate-dependent code |
| @bma-implementor | Subconscious consumer of scout output | All PRs minting `NT_SIGNAL` hyperedges or touching scout output schemas |
| @qbp-architecture | Federation-impact architect | Per `pr-review-completion-best-practices.md` §2 filter |
| @beekeeper | Final HVR | All ratification issues; all first-of-class events (e.g., first NT_SIGNAL emission) |

Cross-tenant reviewers added per-issue when federation-impact triggers fire (per `pr-review-completion-best-practices.md` §2).

---

## 4. Tenant-specific issue labels

| Label | Use |
|---|---|
| `qbp-systema:scout-daemon` | qbp-scout-daemon binary changes |
| `qbp-systema:scouts` | per-scout impl (arxiv / ligo / cascadia / fermi-gbm / etc.) |
| `qbp-systema:stance` | Stance Type-Node alias matcher |
| `qbp-systema:configs` | scope-nodes.yaml + scouts.yaml |
| `qbp-systema:verification` | M1/M2 milestone verification scripts |

Federation-uniform labels (apply to all repos):

| Label | Use |
|---|---|
| `shape:design-surface` | Per `issue-authoring-best-practices.md` §2.1 |
| `shape:ratification` | Per §2.2 |
| `shape:sub-issue` | Per §2.3 |
| `shape:bug-incident` | Per §2.4 |
| `priority:safety-critical` | Per Spec 9.5 §2.1 SAFETY_CRITICAL boundaries |
| `phase:crawl` / `phase:toddle` / `phase:walk` / `phase:run` | Federation phase scoping |

---

## 5. Language-specific test + code overlay

### 5.B Go overlay

- **Test runner:** standard library `testing` (federation default; no test framework wrapper)
- **Table-driven test convention:** recommended for unit tests; mandatory for parametric scout-config tests
- **Fuzz tests:** required for arXiv Atom XML parsing in `pkg/scouts/arxiv/transform.go`; use `testing.F` native fuzz
- **Race detector:** `go test -race` default-required for any package with goroutines (scoutd daemon coordinates concurrent scouts)
- **Property-test framework:** prefer hand-rolled with `testing.Quick` initially; consider `gopter` if scope grows
- **Build tags:**
  - `//go:build integration` — integration tests requiring live arXiv API or Wyrd substrate
  - `//go:build e2e` — M2 end-to-end smoke covering daemon → fetch → transform → write chain
  - `//go:build slow` — slow tests excluded from default `go test ./...`
- **Mock conventions:** interfaces in caller package; no generated mocks unless justified. `httptest.Server` for arXiv API mocking is canonical.
- **Lint stack:** `golangci-lint` (per `code-review-best-practices.md`); `staticcheck`; `govulncheck`
- **gograph integration:** Per `code-review-best-practices.md` §5; reviewers run `gograph impact --since <base>` pre-review
- **gopls modernization:** pre-Sonnet-dispatch gopls modernization clause applies (per cth-implementor §2.3 + qbp-implementor §5.c §2.3)

---

## 6. Tenant-specific closes-when extensions

In addition to the universal closes-when criteria in `issue-authoring-best-practices.md` §5, every qbp-systema ratification issue's closes-when **also** includes:

- **For W2 / W3 substrate-tracking issues:** the substrate dependency (Wyrd `scoutd` impl PR; Contextus scope-loader Go impl) must be merged on its home repo; `qbp-systema/go.mod` updated to consume; `go build ./...` clean
- **For W6.x verification issues:** the named verification script (`scripts/m1_*.sh` or `scripts/m2_*.sh`) runs to completion with exit 0 against a live Wyrd substrate
- **For PRs minting NT_SIGNAL hyperedges:** provenance carries valid CTH anchor reference per QBP's tracked `archive/cth-inventory/confluent-trust-inventory-v5_3.json` (or v0.3 post-`cth migrate`)
- **For first-time arXiv scout cycle:** smoke-test artifacts posted to PR (sample Atom XML response → matched Type-Nodes → NT_SIGNAL hyperedge)

Tenant-specific extensions are **additive** to federation closes-when; they do not relax federation requirements.

---

## 7. Tenant-specific shape preferences

- **Sub-issue pattern:** qbp-systema uses sub-issues aggressively under parent tracking issues (e.g., `repo-QBP-issue-#439` Sprint 2 parent + 14 sub-issues #440-#453). Each sub-issue is independently mergeable with its own PR; parent closes when all sub-issues close.
- **No PIVOT-class incidents pattern:** unlike Wyrd, qbp-systema is a runtime; failures are debugged via standard `shape:bug-incident` issues.

---

## 8. Setup + getting started

qbp-systema setup steps:

```bash
# Prereqs
go version  # Go 1.21+
git --version

# Clone + build
git clone https://github.com/JamesPagetButler/qbp-systema.git
cd qbp-systema
go build ./...   # NOTE: will not compile until W2 (Wyrd scoutd) + W3 (Contextus scope-loader) substrate ships

# Test
go test ./...
go test -race -count=1 ./...

# Lint
golangci-lint run ./...
```

For Sprint 2 development: see `~/Documents/QBP/docs/qbp-federation-tenancy.md` v0.2 + Sprint 2 parent tracking issue [QBP #439](https://github.com/JamesPagetButler/QBP/issues/439).

---

## 9. Cross-references

- `JamesPagetButler/QBP` (companion repo) — federation tenancy doc + CTH inventory + Lean proofs
- `JamesPagetButler/wyrd` PR #54 (scoutd design surface; impl PR pending)
- `JamesPagetButler/contextus` (scope-loader Go impl; tracking via QBP #444)
- `JamesPagetButler/bma-systema` PR #178 (`bma compute-manifest current/validate` reins wrapper; consumed by `cmd/qbp-scout-daemon/main.go` at startup for substrate-drift detection)

---

*qbp-systema CONTRIBUTING.md v0.1 | 2026-05-21*
*Federation template source: `inter/contributing-md-template.md`*
