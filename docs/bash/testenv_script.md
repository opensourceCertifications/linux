# `testenv_script.sh` — Test Environment Bootstrap (optional)

> Your `Vagrantfile` uses an *inline* shell provisioner for `testenv` that performs
> similar steps (system update and key authorization). If you also maintain a
> standalone `testenv_script.sh`, it should align with the behavior below.

---

## Responsibilities

1. **Update system packages** (safe to rerun):
   ```bash
   dnf update -y && dnf upgrade -y
   ```

2. **Wait for monitor’s public key** to appear in the shared folder (`/vagrant_transfer/monitor.pub`)
   and **append** it to `~vagrant/.ssh/authorized_keys`.

3. **Permissions**:
   - Ensure `~vagrant/.ssh` exists with mode `0700`.
   - `authorized_keys` should be mode `0600`.

---

## Robust Wait Loop Example

```bash
set -euo pipefail

PUB=/vagrant_transfer/monitor.pub
AK="$HOME/.ssh/authorized_keys"

mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
touch "$AK"
chmod 600 "$AK"

until [ -s "$PUB" ]; do
  echo "Waiting for $PUB ..."
  sleep 2
done

# Append if not already present
grep -qxF "$(cat "$PUB")" "$AK" || cat "$PUB" >> "$AK"
```

This prevents duplicate entries and ensures correct permissions.

---

## Troubleshooting

- **Shared folder empty** — verify the monitor side wrote `monitor.pub` and that the folder is mounted.
- **Permission denied on SSH** — check ownership (`vagrant:vagrant`) and modes (`0700` dir, `0600` file).
