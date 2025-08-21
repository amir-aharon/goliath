#!/usr/bin/env bash
# make_paste.sh — emit a paste-ready snapshot of your repo into chat_prompt.md
# Usage:
#   ./make_paste.sh                 # core files only (lean)
#   ./make_paste.sh --all           # include all matched files
#   OUTFILE=foo.md ./make_paste.sh  # change output name

set -euo pipefail

OUTFILE="${OUTFILE:-chat_prompt.md}"
MODE="core" # or "all"
if [[ "${1:-}" == "--all" ]]; then MODE="all"; fi

# ------------------------------
# 0) Hardcoded project summary (authoritative)
# ------------------------------
read -r -d '' PROJECT_DESCRIPTION <<'DESC'
This project is a tiny, Redis-inspired in-memory key-value server written in Go.
It exposes a line-based TCP interface (telnet/nc friendly) with a layered design:
server (accepts TCP) → session (parse line, dispatch) → router (arity-checked dispatch) → store (thread-safe map with TTL) → proto (CRLF replies).
Supported commands: PING, ECHO, QUIT, SET, GET, DEL, SETEX, TTL, PERSIST.
TTLs are enforced both lazily (on access) and proactively via a randomized sweeper (configurable interval/sample size).
Replies are centralized so switching to RESP later won’t require touching handlers.
DESC

# If README has a "## paragraph_project_description" block, prefer it; otherwise use hardcoded.
if [[ -f README.md ]] && awk '/^## *paragraph_project_description/{found=1} END{exit !found}' README.md >/dev/null 2>&1; then
  DESCRIPTION="$(awk '/^## *paragraph_project_description/{flag=1; next} /^## /{flag=0} flag {print}' README.md)"
else
  DESCRIPTION="$PROJECT_DESCRIPTION"
fi

# ------------------------------
# 1) Repo metadata
# ------------------------------
in_git=false
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then in_git=true; fi

repo_name="$(basename "$(pwd)")"
branch="(no-git)"
commit="(no-git)"
dirty=""
if $in_git; then
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
  commit="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
  git diff --quiet --ignore-submodules -- || dirty=" (dirty)"
fi

# Determine Go version (best-effort)
go_ver="$( (go version 2>/dev/null | awk '{print $3}') || true )"
mod_go="$( (grep -E '^go [0-9]+\.[0-9]+' go.mod | awk '{print $2}') || true )"
go_summary="${go_ver:-unknown}${mod_go:+ (module go $mod_go)}"

# Parse Makefile targets (optional, best-effort)
make_targets="$( (grep -E '^[a-zA-Z0-9_\-]+:' -o Makefile 2>/dev/null | sed 's/:$//' | sort -u | paste -sd ', ' -) || true )"

# ------------------------------
# 2) File selection
# ------------------------------
# Default include set tries to stay lean. --all broadens it.
include_globs=(
  "*.go" "go.mod" "go.sum" "*.md" "*.yaml" "*.yml" "*.json" "*.sh" "Makefile"
)
exclude_globs=(
  "vendor/**" "**/bin/**" "**/.git/**" "**/.idea/**" "**/.vscode/**" "**/node_modules/**"
  "**/*.exe" "**/*.dll" "**/*.so" "**/*.dylib" "**/*.a" "**/*.o"
  "**/*.png" "**/*.jpg" "**/*.jpeg" "**/*.gif" "**/*.svg"
  "**/*.log" "**/*.out"
  "**/coverage*"
)
# In "core" mode, further narrow files to essentials
core_dirs=( "cmd" "internal" )
core_allow=( "README.md" "Makefile" "go.mod" "go.sum" )

# Collect candidate files
if $in_git; then
  mapfile -t all_files < <(git ls-files)
else
  mapfile -t all_files < <(find . -type f | sed 's#^\./##' | sort -u)
fi

# Filter by includes
filtered=()
for f in "${all_files[@]}"; do
  keep=false
  # include filter
  for pat in "${include_globs[@]}"; do
    if [[ "$f" == $pat ]]; then keep=true; break; fi
  done
  $keep || continue
  # exclude filter
  for pat in "${exclude_globs[@]}"; do
    if [[ "$f" == $pat ]]; then keep=false; break; fi
  done
  $keep || continue
  # core mode additional narrowing
  if [[ "$MODE" == "core" ]]; then
    in_core=false
    for d in "${core_dirs[@]}"; do
      [[ "$f" == "$d/"* ]] && in_core=true
    done
    for a in "${core_allow[@]}"; do
      [[ "$f" == "$a" ]] && in_core=true
    done
    $in_core || continue
  fi
  filtered+=("$f")
done

# Deterministic sort
IFS=$'\n' filtered=($(sort <<<"${filtered[*]}")); unset IFS

# ------------------------------
# 3) Output header
# ------------------------------
cat > "$OUTFILE" <<EOF
# Chat Prompt — Project Summary

This file is meant to bootstrap ChatGPT with context when starting a new conversation.
Paste this whole file into a new chat to restore project awareness.

---

## Project Description
$DESCRIPTION

---

## Repo At A Glance
- **Repo:** $repo_name
- **Branch:** $branch
- **Commit:** $commit$dirty
- **Go:** $go_summary
- **Make targets:** ${make_targets:-n/a}
- **Mode:** $MODE
- **Files included:** ${#filtered[@]}

> Tip: Ask for “a test plan by layers” or “generate a failing test for X” right after pasting.

---
EOF

# ------------------------------
# 4) Append files with syntax fences (truncate very large files)
# ------------------------------
max_lines_per_file=800   # keep paste snappy; adjust if needed
tail_lines=100           # when truncating, show head and tail

for f in "${filtered[@]}"; do
  lang="text"
  case "$f" in
    *.go)   lang="go" ;;
    *.md)   lang="markdown" ;;
    *.yaml|*.yml) lang="yaml" ;;
    *.json) lang="json" ;;
    *.sh)   lang="bash" ;;
    Makefile) lang="make" ;;
    go.mod) lang="go" ;;
    go.sum) lang="text" ;; # often huge
  esac

  echo "## $f" >> "$OUTFILE"
  echo '```'"$lang" >> "$OUTFILE"

  line_count=$(wc -l < "$f" || echo 0)
  if [[ "$line_count" -gt "$max_lines_per_file" ]]; then
    head -n "$((max_lines_per_file - tail_lines - 5))" "$f" >> "$OUTFILE" || true
    echo -e "\n# ... [truncated: $((line_count - (max_lines_per_file - tail_lines - 5))) lines omitted] ...\n" >> "$OUTFILE"
    tail -n "$tail_lines" "$f" >> "$OUTFILE" || true
  else
    cat "$f" >> "$OUTFILE"
  fi

  echo '```' >> "$OUTFILE"
  echo >> "$OUTFILE"
done

# Footer
cat >> "$OUTFILE" <<'EOF'
---
_End of snapshot. Paste everything above into a new ChatGPT thread._
EOF

echo "Wrote $OUTFILE (${#filtered[@]} files, mode=$MODE)"
