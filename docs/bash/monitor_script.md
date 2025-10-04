# `monitor_script.sh` — Monitor VM Bootstrap

**Purpose**: Prepare the *monitor* VM to control the *testenv* VM:
install tools, generate SSH keys, and export environment variables
used by Go orchestration and Ansible playbooks.

> This documentation reflects how the script is invoked and described
> in your `Vagrantfile`. If your local copy differs, adjust accordingly.

---

## Responsibilities

1. **Generate SSH keypair** (non‑root, user `vagrant`) and publish the public key into the shared folder:
   - Key path: `$HOME/.ssh/id_ed25519`
   - Published path: `/vagrant_transfer/monitor.pub`

2. **Install tooling** (typical):
   - Go toolchain (for building the chaos agent)
   - Ansible (for running playbooks from `/vagrant/monitor/ansible`)

3. **Export environment** for the session/system:
   - `TESTENV_ADDRESS` → IP of the test environment VM
   - `MONITOR_ADDRESS` → IP of the monitor VM
   - Typically written into `/etc/environment` for persistence.

---

## Idempotence Notes

- `ssh-keygen` should be guarded to avoid overwriting an existing keypair:
  ```bash
  [ -f "$HOME/.ssh/id_ed25519" ] || ssh-keygen -t ed25519 -f "$HOME/.ssh/id_ed25519" -N ""
  ```
- Install steps should tolerate being re-run (e.g., `dnf install -y ...` is safe).
- Exporting env vars can append duplicates; prefer updating if present.

---

## Security Considerations

- Keypair is created for the `vagrant` user and the public key is transferred via the shared folder.
  Avoid placing private keys in shared/synced folders.
- Ensure `/vagrant_transfer/monitor.pub` is world‑readable is fine, but keep private key permissions strict.

---

## Verifying Setup

```bash
# On monitor VM
$ test -f ~/.ssh/id_ed25519 && echo "key OK"
$ grep -E 'TESTENV_ADDRESS|MONITOR_ADDRESS' /etc/environment
$ ansible --version && go version
```

---

## Failures & Remedies

- **Ansible not found** — rerun `dnf install -y ansible-core` or your bootstrap step.
- **Go not found** — install per your distribution or download official tarball.
- **SSH key missing** — re-run the keygen snippet above.
