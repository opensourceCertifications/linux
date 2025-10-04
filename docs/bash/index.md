# Bash Scripts — Detailed Documentation

This directory documents the Bash scripts that support your **Linux boot corruption lab**.
It explains what each script does, inputs/outputs, exit codes, dependencies, and common
conventions enforced by pre-commit hooks.

**Contents**

- [`boot_checker.md`](sandbox:/mnt/data/docs/bash/boot_checker.md) — authoritative checks for boot-related files on the target VM.
- [`monitor_script.md`](sandbox:/mnt/data/docs/bash/monitor_script.md) — bootstrap actions run on the *monitor* VM.
- [`testenv_script.md`](sandbox:/mnt/data/docs/bash/testenv_script.md) — bootstrap actions for the *testenv* VM (if used as a standalone script).
- [`gen_docs.md`](sandbox:/mnt/data/docs/bash/gen_docs.md) — docs generation via `gomarkdoc` (Go package READMEs).
- [`ensure_bash_header.md`](sandbox:/mnt/data/docs/bash/ensure_bash_header.md) — pre-commit enforcement of a top‑of‑file “shdoc‑style” header.
- [`conventions.md`](sandbox:/mnt/data/docs/bash/conventions.md) — shell standards, safety practices, and patterns used throughout.
