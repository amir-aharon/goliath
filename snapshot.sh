#!/usr/bin/env bash

set -euo pipefail

OUTFILE="snapshot.md"

# Start fresh
echo "# Chat Prompt — Full Project Snapshot" > "$OUTFILE"
echo >> "$OUTFILE"
echo "This file contains the entire repo, concatenated for context." >> "$OUTFILE"
echo >> "$OUTFILE"

# Iterate over tracked files (skip .git and large binaries)
# Adjust the grep -v filters if needed
for f in $(git ls-files | grep -vE '(^\.git|\.md$|\.png$|\.jpg$|\.gif$|\.svg$|\.lock$|go\.sum$)'); do
  echo "---" >> "$OUTFILE"
  echo "## $f" >> "$OUTFILE"
  echo '```'$(basename "$f" | awk -F. '{print $NF}') >> "$OUTFILE"
  cat "$f" >> "$OUTFILE"
  echo '```' >> "$OUTFILE"
  echo >> "$OUTFILE"
done

echo "Snapshot written to $OUTFILE"

