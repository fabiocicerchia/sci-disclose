#!/usr/bin/env bash
# The action's estimate step: build the CLI from this checkout, run it, and
# turn the JSON disclosure into step outputs and the job summary.
# Inputs by env: MANIFEST, BUDGET, ARGS.
set -euo pipefail

bin="$RUNNER_TEMP/sci-disclose"
(cd "$GITHUB_ACTION_PATH" && go build -buildvcs=false -o "$bin" ./cmd/sci-disclose) || exit 2

read -ra extra <<<"$ARGS"
gate=()
[ -n "$BUDGET" ] && gate=(--budget "$BUDGET")

# `|| code=$?` keeps an over-budget exit 1 from aborting under -e.
code=0
# extra goes first: the flag parser keeps the last value, so args cannot redirect the JSON.
"$bin" estimate "${extra[@]}" -f "$MANIFEST" --format json -o "$RUNNER_TEMP/sci.json" \
  --markdown "$RUNNER_TEMP/sci.md" "${gate[@]}" || code=$?
[ "$code" -le 1 ] || exit "$code"

jq -r '"sci=\(.sci)", "sci-unit=\(.sci_unit)",
       "total-gco2e=\(.total_gco2e)",
       "within-budget=\(if .budget then .budget.pass else true end)"' \
  "$RUNNER_TEMP/sci.json" >>"$GITHUB_OUTPUT"
cat "$RUNNER_TEMP/sci.md" >>"$GITHUB_STEP_SUMMARY"
echo "code=$code" >>"$GITHUB_OUTPUT"
