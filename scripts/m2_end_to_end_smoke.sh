#!/usr/bin/env bash
# scripts/m2_end_to_end_smoke.sh
#
# W6.2 verification — M2 end-to-end smoke. Per QBP #452 ACs:
#
#   1. Start qbp-scout-daemon (all configs loaded; M1 passed)
#   2. Manually trigger arXiv scout
#   3. Wait for cycle: Fetch -> Transform -> Write
#   4. Query Wyrd: at least one NT_SIGNAL with provenance source=scout,
#      scout_id=qbp.arxiv.daily, valid CTH anchor reference, non-empty
#      matched_type_nodes list
#
# Per Sprint 2 kickoff readiness plan Phase 4. Gates: W6.1a + W6.1b
# passes + W5.1 + W5.2 implementations integrated against Wyrd scoutd
# real impl PR.

set -euo pipefail

CTH_BASELINE_PATH="archive/cth-inventory/confluent-trust-inventory-v5_3.json"

echo "M2 end-to-end smoke: starting qbp-scout-daemon in --trigger-once mode..."
./qbp-scout-daemon --trigger-once=qbp.arxiv.daily

echo "M2 end-to-end smoke: waiting for cycle to complete..."
# Trigger-once should block until cycle completes; if not, brief sleep
sleep 5

echo "M2 end-to-end smoke: querying Wyrd graph for NT_SIGNAL..."

SIGNAL_COUNT=$(wyrd graph list \
  --kind NT_SIGNAL \
  --filter "scout_id=qbp.arxiv.daily" \
  --since=1m \
  --format=count-only)

echo "M2 end-to-end smoke: NT_SIGNAL count (last 1m, scout_id=qbp.arxiv.daily) = $SIGNAL_COUNT"

if [ "$SIGNAL_COUNT" -lt 1 ]; then
  echo "FAIL: no NT_SIGNAL minted by arxiv scout cycle"
  exit 1
fi

# Spot-check one signal: must have non-empty matched_type_nodes + valid CTH anchor ref
FIRST_SIGNAL=$(wyrd graph list \
  --kind NT_SIGNAL \
  --filter "scout_id=qbp.arxiv.daily" \
  --since=1m \
  --limit=1 \
  --format=json)

MATCHED_NODES=$(echo "$FIRST_SIGNAL" | jq -r '.properties.matched_type_nodes | length')
CTH_BASELINE=$(echo "$FIRST_SIGNAL" | jq -r '.properties.cth_baseline')

echo "M2 end-to-end smoke: first signal has $MATCHED_NODES matched Type-Nodes; baseline=$CTH_BASELINE"

if [ "$MATCHED_NODES" -lt 1 ]; then
  echo "FAIL: signal has empty matched_type_nodes (Transform filter mismatch)"
  exit 1
fi

if [ "$CTH_BASELINE" != "$CTH_BASELINE_PATH" ]; then
  echo "FAIL: signal CTH baseline=$CTH_BASELINE; expected $CTH_BASELINE_PATH"
  exit 1
fi

echo "PASS: M2 end-to-end smoke — arxiv scout cycle produced NT_SIGNAL with valid provenance"
