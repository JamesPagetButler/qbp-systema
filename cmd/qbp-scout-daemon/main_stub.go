//go:build !substrate

// Non-substrate build stub for qbp-scout-daemon.
//
// The real daemon (main.go) is gated behind `//go:build substrate` because it
// imports wyrd/scoutd + wyrd/model + Contextus/scopeconfig, which are not yet
// published (substrate-gated; tracked QBP #443/#444). Until those land, default
// builds compile this stub so `go build ./...` / vet / test / lint stay green;
// CI builds the real daemon with `-tags substrate` once the substrate is available.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "qbp-scout-daemon: substrate-gated build — rebuild with -tags substrate once wyrd/scoutd + Contextus are published (QBP #443/#444).")
	os.Exit(1)
}
