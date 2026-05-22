module github.com/JamesPagetButler/qbp-systema

go 1.23

// External federation substrate dependencies — added when W2 + W3 substrate
// PRs merge:
//
//   require github.com/JamesPagetButler/wyrd vX.Y.Z          // W2 dependency (Wyrd PR #54 scoutd impl)
//   require github.com/JamesPagetButler/Contextus vX.Y.Z      // W3 dependency (Contextus #9 scope-loader Go impl)
//
// Substrate SHA pinning will reflect the Compute Manifest current SHA per
// `bma compute-manifest current` (bma-systema PR #178); the daemon
// verifies match at startup per cmd/qbp-scout-daemon/main.go.
