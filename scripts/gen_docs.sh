#!/usr/bin/env bash
## gen_docs.sh: generate documentation files from source code comments
## - Emits per-package README.md files under docs/go/<pkg-relative-path>/
## - Requires: gomarkdoc, Go toolchain, git

set -euo pipefail

# Repo root and module root for Go sources
ROOT="$(git rev-parse --show-toplevel)"
MOD_ROOT="$ROOT/monitor/go"

cd "$MOD_ROOT"

# Enumerate package directories once so we know exactly what to generate
mapfile -t PKG_DIRS < <(go list -f '{{.Dir}}' ./...)

# Generate docs for each package into docs/go/<rel>/README.md
for abs in "${PKG_DIRS[@]}"; do
	# Path relative to monitor/go (e.g., ".", "shared/library", "breaks")
	rel="${abs#"$PWD"/}"
	# Normalize "." (module root) to empty for nicer output paths
	[[ "$rel" == "$PWD" || "$rel" == "." ]] && rel=""

	outdir="$ROOT/docs/go/${rel}"
	outfile="$outdir/README.md"

	mkdir -p "$outdir"

	# Run gomarkdoc for this specific package directory
	# (use a relative package path like ./shared/library, or . for the module root)
	pkg="./${rel:-.}"

	gomarkdoc \
		--output "$outfile" \
		--repository.url 'https://github.com/opensourceCertifications/linux' \
		--repository.default-branch main \
		"$pkg"

	# Stage just the file we generated
	git add "$outfile"
done
