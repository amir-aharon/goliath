#!/usr/bin/env bash
# make_test_prompt.sh — emit a paste-ready snapshot of test files into test_prompt.md
# Usage: ./make_test_prompt.sh

set -euo pipefail

OUTFILE="${OUTFILE:-test_prompt.md}"

# collect test files (*.go ending with _test.go)
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  mapfile -t files < <(git ls-files '*_test.go')
else
  mapfile -t files < <(find . -type f -name '*_test.go' | sed 's#^\./##' | sort)
fi

# deterministic sort
IFS=$'\n' files=($(sort <<<"${files[*]}")); unset IFS

# write header
cat > "$OUTFILE" <<EOF
# Chat Prompt — Test Suite Summary

This file is meant to bootstrap ChatGPT with context on the tests written so far.
Paste this into a new chat to restore test awareness.

---

## Project Tests Included
- Files: ${#files[@]}
EOF

echo >> "$OUTFILE"

# append files with syntax fences
for f in "${files[@]}"; do
  echo "## $f" >> "$OUTFILE"
  echo '```go' >> "$OUTFILE"
  cat "$f" >> "$OUTFILE"
  echo '```' >> "$OUTFILE"
  echo >> "$OUTFILE"
done

# footer
cat >> "$OUTFILE" <<'EOF'
---
_End of test snapshot. Paste above into a new ChatGPT thread to continue test planning or debugging._
EOF

echo "Wrote $OUTFILE (${#files[@]} test files)"
