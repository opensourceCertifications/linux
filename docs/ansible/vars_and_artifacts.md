# Variables & Artifacts — `ansible_vars.yml` and `results.yml`

This document explains the two main data files used by the Ansible checks.

---

## `ansible_vars.yml` (input)

Path: `monitor/ansible/ansible_vars.yml`

- Maintained by your **Go monitor** (it appends values such as new corrupted paths).
- Copied by `checks.yml` to **`/tmp/ansible_vars.yml`** on the testenv before any checks run.
- Loaded by plays via `vars_files` and exposed to each included check.

**Current keys**

- `corruptedBootFiles` *(list[string])* — absolute paths that should exist and be **CLEAN**. The
  verification check (`verify_restored.yml`) iterates these.

Example:

```yaml
corruptedBootFiles:
  - /boot/initramfs-5.14.0-570.12.1.el9_6.x86_64.img
  - /boot/grub2/grubenv
  - /boot/vmlinuz-5.14.0-570.49.1.el9_6.x86_64
```

> If the Go monitor adds new categories (e.g., `sensitiveConfigFiles`), new checks can consume them
> without changing the orchestration.

---

## `results.yml` (output / artifact)

Remote: **`/tmp/results.yml`** on testenv (authoritative during a run)  
Fetched to repo: `monitor/ansible/results.yml` (snapshot after the run)

- A YAML **mapping** whose values are **lists** (deduplicated).
- Populated by checks through `library/append_to_results.yml`.
- Intended for **machine consumption** *and* human reading.

**Example (after one run)**

```yaml
restored_files:
  - /boot/grub2/grubenv
corrupted_files:
  - /boot/vmlinuz-5.14.0-570.49.1.el9_6.x86_64
```

**Notes**

- It’s safe to delete the fetched `monitor/ansible/results.yml` locally; a new run will regenerate it.
- The remote `/tmp/results.yml` is recreated by the orchestration on every run (initialized as `{}`),
  ensuring clean merges.
