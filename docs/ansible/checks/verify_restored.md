# Check: `verify_restored.yml` — Validate boot files are CLEAN

This check verifies that previously corrupted **boot‑critical paths** are **CLEAN** again by running
a read‑only probe script (`/usr/local/bin/bootcheck`) on the testenv. It classifies each target as
either **restored** or **still corrupted**, and merges results into `/tmp/results.yml`.

- **Location**: `monitor/ansible/checks/verify_restored.yml`
- **Helper used**: [`library/append_to_results.yml`](../library/append_to_results.md)
- **Probe script**: `monitor/ansible/library/boot_checker.sh` (installed as `/usr/local/bin/bootcheck`)

---

## Inputs

- `corruptedBootFiles` — a list of absolute paths to check (strings). Your Go monitor writes/updates
  this in `monitor/ansible/ansible_vars.yml`, which `checks.yml` copies to `/tmp/ansible_vars.yml` on
  the testenv before running checks.

Example (from `ansible_vars.yml`):

```yaml
corruptedBootFiles:
  - /boot/initramfs-5.14.0-570.12.1.el9_6.x86_64.img
  - /boot/grub2/grubenv
  - /boot/vmlinuz-5.14.0-570.49.1.el9_6.x86_64
```

---

## What it installs

Before probing, the check ensures the minimal toolchain exists and deploys the probe script:

- `dracut` — provides `lsinitrd` for initramfs inspection
- `grub2-tools-minimal` — provides `grub2-editenv` for `grubenv`
- `file` — used to confirm kernel images and EFI binaries
- `/usr/local/bin/bootcheck` — a copied version of `library/boot_checker.sh`

> These run with `remote_user: root`. If you prefer a non‑root account, add `become: true` with sudo
> permissions for package installation and file placement under `/usr/local/bin`.

---

## How classification works

For each path in `corruptedBootFiles`, the check runs:

```yaml
ansible.builtin.command:
  argv: ["/usr/local/bin/bootcheck", "{{ item }}"]
```

The `bootcheck` script classifies the path by PATTERN and executes a **read‑only, authoritative** test:

- **Initramfs** (`/boot/initramfs-*.img`) → `lsinitrd` must succeed
- **Kernel** (`/boot/vmlinuz-*`) → `file -b` must say *linux kernel*
- **GRUB env** (`/boot/grub2/grubenv` or `/boot/grub/grubenv`) → `grub2-editenv ... list` must succeed,
  else the file header must be `GRUB Environment Block`
- **grub.cfg** (`/boot/grub2/grub.cfg` or `/boot/grub/grub.cfg`) → must contain `blscfg` or at least one `menuentry`
- **BLS entries** (`/boot/loader/entries/*.conf`) → must contain `title`, `linux`, and `initrd` keys
- **EFI binaries** (`/boot/efi/EFI/*.efi`) → `file -b` must report *EFI application*

If none of the above patterns match **but the file is RPM‑owned**, `rpm -Vf <path>` must be clean
(no output) for the path to be considered **CLEAN**. Missing paths are considered **CORRUPTED**.

The script prints either `CLEAN` or `CORRUPTED` to **stdout** and returns 0/1 accordingly.

---

## Bucketing logic (Ansible)

The check treats **each target independently** and accumulates results into two lists:

- `restored_files` — items for which `rc==0` and `stdout=='CLEAN'`
- `corrupted_files` — items for which `rc!=0` or `stdout=='CORRUPTED'`

Then it **merges** those lists into `/tmp/results.yml` using the helper task file:

```yaml
- include_tasks: "{{ playbook_dir }}/library/append_to_results.yml"
  vars:
    result_key: "restored_files"
    new_items: "{{ restored_files }}"
```

> The merge helper deduplicates entries and preserves idempotence across runs.

---

## Outputs

After the check completes and the helper merges results, `/tmp/results.yml` (on testenv) will contain
keys like:

```yaml
restored_files:
  - /boot/grub2/grubenv
  - /boot/initramfs-5.14.0-570.12.1.el9_6.x86_64.img
corrupted_files:
  - /boot/vmlinuz-5.14.0-570.49.1.el9_6.x86_64
```

This file is then fetched back to your repo as `monitor/ansible/results.yml` by `checks.yml`.

---

## Failure modes & troubleshooting

- **Missing tools** → make sure `dracut`, `grub2-tools-minimal`, and `file` are installed (the check
  ensures this by default).
- **`bootcheck` not found** → confirm the copy task installed it at `/usr/local/bin/bootcheck` with mode `0755`.
- **Unexpected path types** → if the path doesn’t match any known type and isn’t RPM‑owned, it will be
  treated as **CORRUPTED**. Adjust the `boot_checker.sh` patterns if needed.
- **Permissions** → if running without `root`, add `become: true` and ensure the account can install
  packages and write `/usr/local/bin`.

---

## Extending this check

- If you discover additional boot‑critical file types, extend `boot_checker.sh` with a new pattern and
  probe. Keep probes **read‑only**.
- If you need remediation (e.g., rebuild initramfs), do that in a **separate** `fix_*` task file to
  keep this check “validation‑only.”
