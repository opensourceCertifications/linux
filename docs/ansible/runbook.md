# Runbook — Operating the Ansible checks

This runbook shows how to execute the checks, limit scope, and debug issues.

---

## Quick start

From the repo root on the **host**:

```bash
vagrant ssh monitor -c 'cd /vagrant/monitor/ansible && ansible-playbook checks.yml'
```

Prereqs:
- VMs are up (`vagrant up`).
- The monitor VM has `TESTENV_ADDRESS` and `MONITOR_ADDRESS` exported (your provisioning already does this).
- SSH from monitor → testenv works as root (or use `become: true` in plays).

---

## Fetch artifacts

After a run, results land in your repo as:

```
monitor/ansible/results.yml
```

The authoritative remote file during the run is `/tmp/results.yml` on testenv.

---

## Run a **single** check for debugging

Two common options:

1. **Temporarily narrow discovery** in `checks.yml` (Play 1) to a single file pattern, e.g.:

```yaml
query('ansible.builtin.fileglob', playbook_dir ~ '/checks/verify_restored.yml')
```

2. **Use a `when:` guard** inside the include loop in Play 2 to match a specific basename:

```yaml
when: (item | basename) == 'verify_restored.yml'
```

Revert once done.

---

## Common problems

- **“bootcheck not found”** — ensure Play 0 copied it to `/usr/local/bin/bootcheck` with mode `0755`.
- **“command not found: lsinitrd”** — ensure `dracut` is present (Play 0 installs it).
- **Permission denied** — if not connecting as root, add `become: true` and ensure sudoers allow the required tasks.
- **Check produced no output** — make sure the vars file you expect is present on testenv
  (`/tmp/ansible_vars.yml`) and contains the key used by the check.

---

## Extending

- See the authoring guide and skeleton in [`checks/_template.md`](checks/_template.md).
- Keep validation and remediation **separate** to preserve safety and clarity.
- Prefer read‑only probes; avoid shells (`shell:`) unless necessary (use `command:`/`stat:` etc.).
