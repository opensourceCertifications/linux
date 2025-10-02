#!/usr/bin/env bash
## gen_shdoc.sh: generate documentation files from Bash scripts using shdoc
## - Generates docs/bash/<script>.md from each tracked .sh file
## - Requires shdoc:
mkdir -p docs/bash
# Find all tracked .sh files; generate docs/bash/<name>.md
while IFS= read -r f; do
	base="$(basename "$f" .sh)"
	if command -v shdoc > /dev/null; then
		shdoc < "$f" > "docs/bash/${base}.md"
	else
		echo "shdoc not found (https://github.com/reconquest/shdoc) — skipping $f" >&2
	fi
done < <(git ls-files "*.sh")
