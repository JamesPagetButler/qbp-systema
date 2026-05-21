#!/usr/bin/env bash
# scripts/m1_proof_of_life.sh
#
# W6.1a verification — M1 proof-of-life. Per QBP #450 ACs:
#
#   wyrd graph count --kind NT_SCOPE_PHYSICAL,NT_SCOPE_CONCEPTUAL >= 11
#
# Validates the config-load path: scope-nodes.yaml -> Contextus scope-loader
# -> Wyrd graph -> queryable count. No scout fires; no external network.
#
# Per QBP federation tenancy v0.2 §9 M1 row + my discovery response §6.1
# split: M1 = (a) + (b). This script is the (a) proof-of-life half;
# m1_roundtrip_test.sh is the (b) round-trip half.
#
# Per Sprint 2 kickoff readiness plan Phase 4 (commitment to qbp-systema repo).

set -euo pipefail

EXPECTED_MIN=11   # 6 NT_SCOPE_PHYSICAL + 6 NT_SCOPE_CONCEPTUAL minus tolerance for partial loads

echo "M1 proof-of-life: starting qbp-scout-daemon in --load-configs-only mode..."
./qbp-scout-daemon --load-configs-only &
DAEMON_PID=$!
trap "kill $DAEMON_PID 2>/dev/null || true" EXIT

# Wait briefly for configs to load + register in Wyrd graph.
sleep 2

echo "M1 proof-of-life: querying Wyrd graph..."
COUNT=$(wyrd graph count --kind NT_SCOPE_PHYSICAL,NT_SCOPE_CONCEPTUAL --format=count-only)

echo "M1 proof-of-life: NT_SCOPE_* count = $COUNT (expected >= $EXPECTED_MIN)"

if [ "$COUNT" -lt "$EXPECTED_MIN" ]; then
  echo "FAIL: count $COUNT below minimum $EXPECTED_MIN"
  exit 1
fi

echo "PASS: M1 (a) proof-of-life — count $COUNT meets threshold"
