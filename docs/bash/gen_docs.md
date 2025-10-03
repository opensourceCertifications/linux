# `gen_docs.sh` — Generate Go Package READMEs with `gomarkdoc`

**Purpose**: Generate per‑package `README.md` files from your Go source comments (godoc style)
so each package directory is self‑documenting. This is wired into pre‑commit.

---

## How It Works (as implemented in pre‑commit)

1. Change into the Go module root: `monitor/go`.
2. Enumerate package directories with `go list -f "{{.Dir}}" ./...`.
3. Run `gomarkdoc` to write documentation into `{{.Dir}}/README.md` for *each* package:
   ```bash
   gomarkdoc      --output "{{.Dir}}/README.md"      --repository.url "https://github.com/opensourceCertifications/linux"      --repository.default-branch main      ./...
   ```
4. Stage only the generated `README.md` files that correspond to actual packages.

---

## Prerequisites

- Go installed (for `go list`).
- `gomarkdoc` installed. The pre‑commit hook either expects it on `PATH` or installs it via:
  ```bash
  go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
  ```

---

## Tips

- Godoc comments drive the quality of these READMEs. Start package files with a
  `// Package <name> ...` synopsis and write function/type comments (`// Name ...`).
- Generated READMEs are committed so documentation travels with the code.
