# qbp-systema

**QBP federation tenancy runtime.** The Go-side runtime that makes the QBP programme operational as a federation tenant: scout daemon + scope-node configs + Stance Type-Node matcher.

Tenant role per `~/Documents/inter/workspace-phase-architecture.md`: **tier T5 federation-integration** (per Sprint 2 BADASS dashboard). First tenant to consume Wyrd `scoutd/` + Contextus scope-loader as a federation member; the runtime that exercises the federation tenancy pattern (`~/Documents/QBP/docs/qbp-federation-tenancy.md` v0.2, merged via QBP PR #403).

## What lives here vs in the QBP repo

- **`JamesPagetButler/QBP`** — physics paper-corpus, Lean proofs, CTH inventory, federation tenancy doc, paper-corpus PRs (theory-side qbp-oppenheimer + integration-side qbp-implementor)
- **`JamesPagetButler/qbp-systema`** (this repo) — Go runtime that consumes the substrate; never authors theory or papers

## Layout (Sprint 2 W1.2 bootstrap)

```
qbp-systema/
├── cmd/
│   └── qbp-scout-daemon/        # entrypoint; instantiates wyrd/scoutd.Daemon
├── pkg/
│   ├── scouts/
│   │   └── arxiv/               # arXiv daily-batch scout Fetch + Transform
│   ├── stance/                  # Stance Type-Node alias matcher (22 v0.2 Stance vocab)
│   └── hopf/                    # Hopf-locale conversion helpers (tenancy v0.2 §1.5)
├── configs/
│   ├── scope-nodes.yaml         # per tenancy v0.2 §3
│   └── scouts.yaml              # per tenancy v0.2 §4
├── scripts/
│   ├── m1_proof_of_life.sh      # W6.1a verification
│   ├── m1_roundtrip_test.sh     # W6.1b verification
│   └── m2_end_to_end_smoke.sh   # W6.2 verification
├── CONTRIBUTING.md
└── README.md (this file)
```

## Sprint 2 status

| Workstream | Sub-issue | State |
|---|---|---|
| W1.1 — repo + branch protection | [QBP #440](https://github.com/JamesPagetButler/QBP/issues/440) | ✅ done (this commit) |
| W1.2 — Go module + directory layout | [QBP #441](https://github.com/JamesPagetButler/QBP/issues/441) | in progress |
| W1.3 — CI workflows | [QBP #442](https://github.com/JamesPagetButler/QBP/issues/442) | next |
| W2 — Wyrd `scoutd` impl tracking | [QBP #443](https://github.com/JamesPagetButler/QBP/issues/443) | tracking; substrate-gated |
| W3 — Contextus scope-loader Go impl tracking | [QBP #444](https://github.com/JamesPagetButler/QBP/issues/444) | tracking; substrate-gated |
| W4.1 — `configs/scope-nodes.yaml` | [QBP #445](https://github.com/JamesPagetButler/QBP/issues/445) | draft staged |
| W4.2 — `configs/scouts.yaml` | [QBP #446](https://github.com/JamesPagetButler/QBP/issues/446) | draft staged |
| W4.3 — configs validate against schemas | [QBP #447](https://github.com/JamesPagetButler/QBP/issues/447) | gated on W2 + W3 |
| W5.1 — arXiv `Fetch` impl | [QBP #448](https://github.com/JamesPagetButler/QBP/issues/448) | draft staged |
| W5.2 — arXiv `Transform` impl | [QBP #449](https://github.com/JamesPagetButler/QBP/issues/449) | draft staged |
| W6.1a — M1 proof-of-life | [QBP #450](https://github.com/JamesPagetButler/QBP/issues/450) | draft staged |
| W6.1b — M1 round-trip | [QBP #451](https://github.com/JamesPagetButler/QBP/issues/451) | draft staged |
| W6.2 — M2 end-to-end smoke | [QBP #452](https://github.com/JamesPagetButler/QBP/issues/452) | draft staged |
| W6.3 — Documentation cross-link | [QBP #453](https://github.com/JamesPagetButler/QBP/issues/453) | gated on M1 + M2 |

Parent tracking issue: [QBP #439](https://github.com/JamesPagetButler/QBP/issues/439).

## Phase 0 staged drafts

9 draft artifacts pre-staged at `/tmp/qbp-sprint2-staging/` (per Sprint 2 kickoff readiness plan Phase 0; YAML + bash lint clean). Committed to this repo as W1.2 / W4.1 / W4.2 / W5.1 / W5.2 / W6.x land.

## Build

```bash
go build ./...
```

(Will compile once W2 (Wyrd `scoutd`) + W3 (Contextus scope-loader Go) substrate dependencies merge.)

## Federation tenancy pattern

This repo is the canonical reference implementation for the federation tenancy pattern. Future tenants (Sharp Butler, Möbius Fusion, Materia, etc.) follow the same `<tenant>-systema` shape per `inter/workspace-phase-architecture.md` §0.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

— Tenant role: qbp-implementor (Integration); federation reviewer-list per `CONTRIBUTING.md` §3
