#!/usr/bin/env bash
## Fails if a shell script doesn't have a top-of-file doc header that starts with '##'.
## We allow a shebang on the first line and any blank lines before the header.

set -euo pipefail

# If run manually with no args, show a friendly usage and exit.
if [ "$#" -eq 0 ]; then
  echo "usage: $0 <shell-files...>" >&2
  exit 2
fi

status=0

for f in "$@"; do
  # Only regular files
  [ -f "$f" ] || continue

  # Grab first non-empty, non-shebang line (trimmed)
  first_line="$(
    awk '
      NR==1 && /^#!/ { next }                 # skip shebang on first line
      { gsub(/^[[:space:]]+|[[:space:]]+$/, "") }   # trim
      length { print; exit }                  # first non-empty -> print & exit
    ' "$f" 2>/dev/null || true
  )"

  # Use a regex match instead of a brittle glob pattern
  if [[ -z $first_line || ! $first_line =~ ^## ]]; then
    echo "❌ $f: missing top-of-file doc header. Add a line starting with '## ' after the shebang."
    status=1
  fi
done

exit "$status"
