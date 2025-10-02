#!/usr/bin/env bash
set -euo pipefail
cd monitor/go

# Enumerate package dirs so we know exactly which READMEs to expect.
mapfile -t PKG_DIRS < <(go list -f '{{.Dir}}' ./...)

# Generate docs to per-package README.md
gomarkdoc \
	--output '{{.Dir}}/README.md' \
	--repository.url 'https://github.com/opensourceCertifications/linux' \
	--repository.default-branch main \
	./...

# Stage only the READMEs that correspond to actual packages
for d in "${PKG_DIRS[@]}"; do
	[ -f "$d/README.md" ] && git add "$d/README.md"
done
