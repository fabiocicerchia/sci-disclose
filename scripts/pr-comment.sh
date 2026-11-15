#!/usr/bin/env bash
# The action's PR comment step: post $RUNNER_TEMP/sci.md as one comment on PR,
# editing the previous one on a re-run. A failed post is a warning, not a failure.
# Inputs by env: GH_TOKEN, PR.
set -euo pipefail

# The marker finds this action's comment again, so a re-run edits it.
marker='<!-- sci-disclose -->'
body="$marker"$'\n'"$(cat "$RUNNER_TEMP/sci.md")"
api="repos/$GITHUB_REPOSITORY/issues"
pick=".[] | select(.body | startswith(\"$marker\")) | .id"
id=$(gh api --paginate "$api/$PR/comments" --jq "$pick" | head -1) ||
  { echo "::warning::sci-disclose: could not read the PR comments"; exit 0; }
if [ -n "$id" ]; then
  gh api -X PATCH "$api/comments/$id" -f body="$body" >/dev/null
else
  gh api "$api/$PR/comments" -f body="$body" >/dev/null
fi || echo "::warning::sci-disclose: could not post the PR comment"
