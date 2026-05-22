#!/usr/bin/env bash
# scripts/m1_roundtrip_test.sh
#
# W6.1b verification — M1 round-trip. Per QBP #451 ACs:
#
#   For each scope-node in scope-nodes.yaml:
#     wyrd graph get <id>
#     Compare returned record against YAML source — type-nodes list,
#     description, geometry match byte-for-byte (after whitespace normalisation)
#
# Per architect seq=52 refinement of my §discovery response: M1 split into
# (a) proof-of-life + (b) round-trip for separable failure surfaces. If
# (a) passes but (b) fails, the failure is in graph storage / query
# serialisation, not config-load.

set -euo pipefail

# Preflight: verify external dependencies are on PATH (per PR #2 Red Team S2).
for cmd in wyrd yq; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "FAIL: $cmd not found; install before running"; exit 2; }
done

SCOPE_NODES_YAML="${SCOPE_NODES_YAML:-configs/scope-nodes.yaml}"

echo "M1 round-trip: extracting scope-node IDs from $SCOPE_NODES_YAML..."
IDS=$(yq -r '.scope_nodes[].id' "$SCOPE_NODES_YAML")

FAIL_COUNT=0
PASS_COUNT=0
TOTAL=0

for id in $IDS; do
  TOTAL=$((TOTAL + 1))
  echo -n "  $id ... "

  # Pull from Wyrd graph; check exit code explicitly (per PR #2 Red Team K2:
  # prior `2>&1 || echo SENTINEL` pattern merged stderr into the variable and
  # missed the succeeds-with-garbage-output edge case).
  if ! WYRD_RECORD=$(wyrd graph get "$id" --format=yaml 2>/dev/null); then
    echo "FAIL (not in graph)"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    continue
  fi

  # Extract source record from YAML for this id.
  YAML_RECORD=$(yq -y ".scope_nodes[] | select(.id == \"$id\")" "$SCOPE_NODES_YAML")

  # Normalise both via yq round-trip to canonicalise whitespace + key order.
  WYRD_NORM=$(echo "$WYRD_RECORD" | yq -y . 2>/dev/null)
  YAML_NORM=$(echo "$YAML_RECORD" | yq -y . 2>/dev/null)

  if [ "$WYRD_NORM" = "$YAML_NORM" ]; then
    echo "PASS"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    echo "FAIL (record drift)"
    echo "    --- yaml ---"
    echo "$YAML_NORM" | head -5
    echo "    --- wyrd ---"
    echo "$WYRD_NORM" | head -5
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
done

echo
echo "M1 round-trip: $PASS_COUNT / $TOTAL scope-nodes round-tripped byte-clean"

if [ "$FAIL_COUNT" -gt 0 ]; then
  echo "FAIL: $FAIL_COUNT scope-nodes failed round-trip"
  exit 1
fi

echo "PASS: M1 (b) round-trip — all $TOTAL scope-nodes byte-clean"
