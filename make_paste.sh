#!/usr/bin/env bash
# make_paste.sh — emit a paste-ready snapshot of your repo into chat_prompt.md
# Usage: ./make_paste.sh

set -euo pipefail

OUTFILE="chat_prompt.md"

# Extract the paragraph_project_description block from README.md
# (Assumes it starts with "paragraph_project_description" heading and ends at the next heading or EOF)
description=$(awk '/^## *paragraph_project_description/{flag=1; next} /^## /{flag=0} flag {print}' README.md)

# Write the extracted description at the top of the file
cat > "$OUTFILE" <<EOF
# Chat Prompt — Project Summary

This file is meant to bootstrap ChatGPT with context when starting a new conversation.
Paste this whole file into a new chat to restore project awareness.

---

## Project Description
$description

---
EOF

# Collect source files
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  files=$(git ls-files | grep -E '\.(go|mod|sum|md|yaml|yml|json|sh)$')
else
  files=$(find . -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' -o -name '*.md' -o -name '*.yaml' -o -name '*.yml' -o -name '*.json' -o -name '*.sh' \) | sort)
fi

# Append files with syntax highlighting
while IFS= read -r f; do
  [ -z "$f" ] && continue
  case "$f" in
    *.go)   lang=go ;;
    *.md)   lang=markdown ;;
    *.yaml|*.yml) lang=yaml ;;
    *.json) lang=json ;;
    *.sh)   lang=bash ;;
    go.mod) lang=go ;;
    go.sum) lang=text ;;
    *)      lang=text ;;
  esac

  {
    echo "## $f"
    echo '```'"$lang"
    cat "$f"
    echo '```'
    echo
  } >> "$OUTFILE"
done <<< "$files"
