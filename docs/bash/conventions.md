# Shell Conventions & Safety Guide

These conventions are followed and (partially) enforced by pre‑commit hooks.

---

## Safety Flags

- Always start scripts with:
  ```bash
  set -euo pipefail
  IFS=$'\n\t'   # optional: tighten word splitting when reading untrusted input
  ```
- Use `trap` for cleanup when creating temp files or background jobs.

## Quoting & Expansion

- Quote all variable expansions: `"$var"` unless you *explicitly* want word splitting.
- Use `--` to end options before paths: `grep -q -- "$needle" -- "$file"`.
- Prefer `printf` over `echo` for predictable formatting.

## Command Discovery & Dependencies

- Test for commands: `command -v tool >/dev/null 2>&1 || err "missing tool: tool"`
- Record runtime deps in the script header or dedicated docs section.

## Flow Control

- Avoid `A && B || C` as if/else — it breaks when `B` fails; write a real `if` block:
  ```bash
  if A; then B; else C; fi
  ```

## Files & Permissions

- Create parent dirs with proper modes, then files, then set permissions:
  ```bash
  install -d -m 0755 /some/dir
  install -m 0644 file /some/dir/file
  ```

## Logging

- Send user messages to stdout; diagnostics to stderr:
  ```bash
  log() { printf '%s\n' "$*" >&2; }
  ```

## Testing & Linting

- Lint with `shellcheck` (pre‑commit hook already configured).
- Format with `shfmt` (pre‑commit hook configured to use **tabs** per your project).
- Keep the shebang on line 1 to avoid `SC1128` errors.
