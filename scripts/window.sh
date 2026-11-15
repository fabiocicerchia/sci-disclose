#!/usr/bin/env bash
# The action's window step: sum what the ledger branch recorded in the last
# HOURS, add this run, gate on LIMIT, and record the run when it passes.
# Inputs by env: GH_TOKEN, LIMIT, HOURS, BRANCH. Reads $RUNNER_TEMP/sci.json.
set -euo pipefail

spent=$(jq .total_gco2e "$RUNNER_TEMP/sci.json")
run="$GITHUB_RUN_ID-$GITHUB_RUN_ATTEMPT"
remote="$GITHUB_SERVER_URL/$GITHUB_REPOSITORY"
auth="AUTHORIZATION: basic $(printf 'x-access-token:%s' "$GH_TOKEN" | base64 | tr -d '\n')"
g() {
  git -c http.extraheader="$auth" -c user.name='github-actions[bot]' \
    -c user.email='41898282+github-actions[bot]@users.noreply.github.com' "$@"
}

# The push is a compare-and-swap on the branch: a run that loses the race
# re-reads the ledger and is gated again, so two runs cannot both spend the rest.
recorded=false
for try in 1 2 3 4 5; do
  cd "$RUNNER_TEMP" && rm -rf sci-ledger && git init -q sci-ledger && cd sci-ledger
  if g fetch -q --depth 1 "$remote" "$BRANCH" 2>/dev/null; then git checkout -q FETCH_HEAD; fi
  touch ledger.jsonl
  window=$(jq -s --argjson h "$HOURS" --argjson spent "$spent" \
    '(now - $h * 3600 | todate) as $since
     | map(select(.at >= $since)) | unique_by(.run) | map(.gco2e) | add // 0 | . + $spent' ledger.jsonl)
  within=$(jq -n --argjson w "$window" --argjson l "$LIMIT" '$w <= $l')
  [ "$within" = true ] || break
  jq -nc --arg run "$run" --arg sha "$GITHUB_SHA" --argjson g "$spent" \
    '{at: (now | todate), gco2e: $g, run: $run, sha: $sha}' >>ledger.jsonl
  git add ledger.jsonl && g commit -qm "ledger: run $run, $spent gCO2e"
  if g push -q "$remote" "HEAD:refs/heads/$BRANCH"; then recorded=true; break; fi
  sleep "$try"
done

echo "window-gco2e=$window" >>"$GITHUB_OUTPUT"
echo "within-window-budget=$within" >>"$GITHUB_OUTPUT"
line="Rolling ${HOURS}h window: **$window gCO2e** of a $LIMIT gCO2e budget, this run included."
echo "$line" >>"$RUNNER_TEMP/sci.md"
echo "$line" >>"$GITHUB_STEP_SUMMARY"
if [ "$within" != true ]; then
  echo "code=1" >>"$GITHUB_OUTPUT"
elif [ "$recorded" != true ]; then
  echo "::error::sci-disclose: could not record this run on $BRANCH; failing rather than leave it uncounted"
  echo "code=2" >>"$GITHUB_OUTPUT"
fi
