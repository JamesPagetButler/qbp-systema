// Package arxiv implements the QBP arXiv daily-batch scout's Fetch +
// Transform halves per QBP federation tenancy v0.2 §4.1.
//
// # Future test surface (PR #1 Red Team S1)
//
// This file documents the named tests that land with W5.1 + W5.2
// follow-up PRs (closes QBP #448 / #449). Per qbp-systema CONTRIBUTING.md
// §5.B Go overlay: race-detector default-required; fuzz tests required
// for arXiv Atom XML parsing.
//
// ## Fetch (fetch.go)
//
//   - TestFetch_SourceRateLimit — assert >= 1/3s gap between
//     consecutive requests per tenancy §4.1 (3 req/s)
//   - TestFetch_UserAgentHeader — assert UA equals
//     "QBP-Federation-Tenant/0.2 (https://github.com/JamesPagetButler/QBP)"
//   - TestFetch_BackoffOnTransient5xx — httptest server returning 503
//     three times then 200; assert 3 retries + success on 4th attempt
//   - TestFetch_BackoffOnRateLimit429 — same pattern with 429
//   - TestFetch_FailFast4xxNonRetryable — 404 returns ErrXyz without retry
//   - TestFetch_RespectContextCancellation — cancel ctx mid-backoff;
//     assert ctx.Err() bubbles up
//   - TestFetch_AllSourcesFailReturnsError — all 13 sources 503; assert
//     ErrFetchAllSourcesFailed wraps per-source error list
//   - TestFetch_PartialSuccessFlowsToTransform — 10 of 13 sources succeed;
//     assert returned bytes contain 10 concatenated feeds + no error
//     (S3 TODO: future operator-visibility test once stress.Bus emission
//     lands per W5.x follow-up)
//
// ## Transform (transform.go)
//
//   - TestTransform_EmptyInputReturnsError — zero-byte input → ErrEmptyInput
//   - TestTransform_AtomFeedParseHappyPath — fixture with N entries
//     + matched Stance Type-Nodes; assert N hyperedges minted
//   - TestTransform_AtomFeedParseInvalidXML — malformed XML → wrapped error
//   - TestTransform_StanceTypeNodeMatch — fixture with arXiv abstract
//     containing "spectral triple"; assert qbp:spectral-triple in
//     matched_type_nodes; assert proper CTH anchor refs
//   - TestTransform_NoMatchEmptyResult — fixture with abstract mentioning
//     no Stance vocabulary; assert empty hyperedge list + no error
//   - TestTransform_MultipleMatchesPerEntry — fixture with abstract
//     hitting 3 different Type-Nodes; assert all 3 in matched_type_nodes
//   - TestTransform_ProvenanceFields — assert minted hyperedge has
//     source="scout", scout_id="qbp.arxiv.daily", agent_class="scout",
//     referent_kind="scalar", cth_baseline pointing at the v5_3 file
//
// ## Fuzz tests (per CONTRIBUTING.md §5.B)
//
//   - FuzzTransform_AtomXML — fuzz the Atom XML parser surface;
//     no input should crash the transform; well-formed-but-adversarial
//     XML should return wrapped error not panic
//
// ## Property tests
//
//   - PropertyTransform_MintIsIdempotent — minting NT_SIGNAL for the
//     same Atom entry twice produces identical hyperedge.ID (deterministic
//     from arxiv_id + published timestamp)
//   - PropertyTransform_DeduplicatedMatchedTypeNodes — duplicate aliases
//     in the same abstract don't double-count
//
// ## Integration tests (build tag `//go:build integration`)
//
//   - TestFetchIntegration_LiveArxivAPI — fetches from real arxiv.org;
//     skipped in default `go test`; runs in nightly CI with rate-limit
//     budget
//
// ## End-to-end tests (build tag `//go:build e2e`)
//
//   - TestEndToEnd_FetchToHyperedgeMint — Fetch + Transform composed;
//     assert the M2 verification AC: at least one NT_SIGNAL minted with
//     non-empty matched_type_nodes (mocked Wyrd graph write)
//
// All tests above land in W5.1 + W5.2 follow-up PRs once Wyrd scoutd
// impl PR (Wyrd PR #54 → impl PR) merges and the substrate-import
// errors clear. Tracked at QBP #448 + #449.
package arxiv
