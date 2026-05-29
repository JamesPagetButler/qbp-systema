module github.com/JamesPagetButler/qbp-systema

go 1.24

toolchain go1.24.2

// External federation substrate dependencies — added when W2 + W3 substrate
// PRs merge:
//
//   require github.com/JamesPagetButler/wyrd vX.Y.Z          // W2 dependency (Wyrd PR #54 scoutd impl)
//   require github.com/JamesPagetButler/Contextus vX.Y.Z      // W3 dependency (Contextus #9 scope-loader Go impl)
//
// Substrate SHA pinning will reflect the Compute Manifest current SHA per
// `bma compute-manifest current` (bma-systema PR #178); the daemon
// verifies match at startup per cmd/qbp-scout-daemon/main.go.

require (
	github.com/JamesPagetButler/confluent-trust v0.1.1-0.20260522031340-97aee0854b67
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
