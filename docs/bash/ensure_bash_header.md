# `ensure_bash_header.sh` — Enforce Top‑of‑File Header for Shell Scripts

**Purpose**: Fail the commit if any shell script lacks a short “shdoc‑style”
header near the top of the file. This encourages a concise synopsis per script.

> This complements `shellcheck` (quality), `shfmt` (format), and your other
> pre‑commit hooks. It does **not** generate docs by itself; it enforces that a
> human‑written header exists.

---

## Policy Enforced

For each shell script (file type detected as `shell` by pre‑commit):

1. **Shebang must be on line 1** — to satisfy shellcheck rule `SC1128`.
   Example shebang:
   ```bash
   #!/usr/bin/env bash
   ```

2. **Header comment using `##`** must appear within the first few lines
   (e.g., within lines 2–5) *after* the shebang. Minimal example:
   ```bash
   #!/usr/bin/env bash
   ## monitor_script.sh — bootstrap the monitor VM with Go, Ansible and env vars
   ## Idempotent and safe to rerun during Vagrant provisioning.
   ```

If any file violates the policy, the hook prints a clear error and exits non‑zero
to block the commit.

---

## Why `##`?

- A consistent marker that’s easy to grep.
- Compatible with shdoc-style tooling if you later decide to generate docs from comments.
- Leaves room for single‑`#` inline comments within the code body.

---

## Common Failures

- Header comment placed *before* the shebang → shellcheck will complain (`SC1128`).
- Only single‑`#` comments at the top → add at least one `##` synopsis line.
