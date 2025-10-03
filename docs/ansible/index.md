# Ansible Checks & Playbooks — Overview

This folder documents the **Ansible side** of the project: how the monitor orchestrates checks
against the test environment, where results are stored, and how to extend the system with new
checks.

> **Context**: The monitor VM runs Ansible against a test VM (the *testenv*). Your Go monitor may
> corrupt boot‑related files remotely; Ansible **verifies** them and buckets results into
> `restored_files` and `corrupted_files`. All results are aggregated on the testenv at
> `/tmp/results.yml` and fetched back to your repo as `monitor/ansible/results.yml`.

---

## Files & layout

```
monitor/ansible/
├─ ansible_vars.yml                 # Input vars produced by the Go monitor (e.g., corruptedBootFiles)
├─ checks.yml                       # Orchestration entrypoint: discover checks, run them, fetch results
├─ checks/
│  └─ verify_restored.yml           # A check: probes boot paths and buckets into restored/corrupted
├─ library/
│  ├─ append_to_results.yml         # Helper: merge a list into /tmp/results.yml under a given key
│  └─ boot_checker.sh               # Probe script installed on testenv (read‑only checks)
└─ results.yml                      # Fetched artifact: copy of /tmp/results.yml after a run
```

- **Per‑check docs** now live under [`checks/`](checks/) to keep this overview lean.
  - See: [`verify_restored.md`](checks/verify_restored.md)
- Helper docs live under [`library/`](library/).
  - See: [`append_to_results.md`](library/append_to_results.md)
- Variables & artifacts are explained in [`vars_and_artifacts.md`](vars_and_artifacts.md).

---

## High‑level flow

```
monitor (localhost)
  ├─ Add host “testenv” from TESTENV_ADDRESS → dynamic inventory group: testenv
  ├─ Discover checks: monitor/ansible/checks/*.yml, *.yaml
  └─ [debug] print what will run

testenv (root)
  ├─ Copy ansible_vars.yml → /tmp/ansible_vars.yml
  ├─ Initialize /tmp/results.yml to "{}"
  └─ For each discovered check (include_tasks):
       run the check → compute buckets → include library/append_to_results.yml

testenv (root)
  └─ Fetch /tmp/results.yml → monitor/ansible/results.yml
```

- **Checks are additive**: each one merges its own keys into `/tmp/results.yml`. Keys are lists.
- The merge helper ensures **deduplication** and **idempotence**.

---

## Entrypoint: `checks.yml`

`checks.yml` is the orchestration playbook. It:

1. **Creates dynamic inventory** on the monitor (adds host from `TESTENV_ADDRESS` into group `testenv`).
2. **Discovers** all check task files under `checks/`.
3. On **testenv**, copies `ansible_vars.yml` → `/tmp/ansible_vars.yml`, initializes `/tmp/results.yml`, and
   **includes** each discovered check file.
4. **Fetches** `/tmp/results.yml` back to the repo as `results.yml`.

For the full, annotated explanation of each step, see the inline comments in
`monitor/ansible/checks.yml` and the runbook below.

---

## Runbook (quick start)

```
# From repo root (host)
vagrant ssh monitor -c 'cd /vagrant/monitor/ansible && ansible-playbook checks.yml'
```

Results will appear in `monitor/ansible/results.yml`.

- Make sure `TESTENV_ADDRESS` and `MONITOR_ADDRESS` are exported on the monitor.
- `remote_user: root` is used; adjust with `become: true` if using a non‑root SSH user.

See: [`runbook.md`](runbook.md) for more tips, debugging, and single‑check execution.

---

## Extend with new checks

- Add a task file under `monitor/ansible/checks/` (e.g., `verify_permissions.yml`).
- Follow the recommended structure in [`checks/_template.md`](checks/_template.md):
  1. Define targets (often from `corruptedBootFiles`).
  2. Initialize buckets (two lists).
  3. Run probes/stat/commands to classify each item.
  4. Append your buckets via `library/append_to_results.yml` with meaningful keys.

> Each check should be **read‑only** unless it explicitly fixes something. Keep remediation in a
> separate task file to preserve clarity.

---

## References

- Per‑check deep‑dives live alongside their task files: [`checks/`](checks/)
- Helper details: [`library/append_to_results.md`](library/append_to_results.md)
- Variable and artifact explanation: [`vars_and_artifacts.md`](vars_and_artifacts.md)
- Operational details: [`runbook.md`](runbook.md)
