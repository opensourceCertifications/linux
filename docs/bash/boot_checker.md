# `boot_checker.sh` — Boot File Health Probe

**Purpose**: Validate integrity/health of *one* boot-related path on RHEL/AlmaLinux‑family hosts.
Outputs a single word and an exit code so Ansible can bucket results deterministically.

---

## Synopsis

```bash
bootcheck /absolute/path
```

**Output (stdout)**: exactly one of

- `CLEAN` — path appears healthy/valid
- `CORRUPTED` — path is missing or fails the relevant probe

**Exit codes**

| Code | Meaning        | Notes                                  |
|-----:|----------------|-----------------------------------------|
| 0    | CLEAN          | Health checks passed                    |
| 1    | CORRUPTED      | Missing or failed probe                 |
| 2    | Usage error    | Wrong CLI usage                         |

The script also prints `bootcheck: ...` diagnostic messages to **stderr** when useful.

---

## Classification & Probes

The script first classifies the path by regex and then runs an appropriate, **read‑only** probe.
If a tool is missing, it errs on the safe side (treat as corrupted) and explains why on stderr.

### Initramfs (`/boot/initramfs-*.img`)

- **Check**: `lsinitrd "$path"` must succeed regardless of compression.
- **Why**: If the archive is truncated/garbled, `lsinitrd` fails.
- **Deps**: `dracut` (provides `lsinitrd`).

### Kernel image (`/boot/vmlinuz-*`)

- **Check**: `file -b -- "$path"` must contain “linux kernel” (case‑insensitive).
- **Why**: Ensures it’s a recognized kernel binary.
- **Deps**: `file`.

### GRUB environment (`/boot/grub2/grubenv` or `/boot/grub/grubenv`)

- **Check**: Prefer `grub2-editenv "$path" list` success.
- **Fallback**: First line must be exactly `GRUB Environment Block`.
- **Why**: Detects damaged GRUB env block.

### GRUB config (`/boot/grub2/grub.cfg` or `/boot/grub/grub.cfg`)

- **Check**: Must contain `blscfg` *or* at least one `menuentry`.
- **Why**: Empty or malformed grub.cfg won’t provide boot entries.

### BLS entry (`/boot/loader/entries/*.conf`)

- **Check**: Must contain `title`, `linux`, and `initrd` keys.
- **Why**: Minimal fields needed for BLS boot entries.

### EFI application (`/boot/efi/EFI/*.efi`)

- **Check**: `file -b -- "$path"` must report “EFI application” (case‑insensitive).
- **Why**: Verifies it’s a valid PE/EFI app.

### Fallback: RPM verification

If none of the above patterns matched **and** the file is owned by an RPM package (`rpm -qf -- "$path"`),
then `rpm -Vf -- "$path"` must print **nothing** (pristine) to qualify as `CLEAN`. Any output → `CORRUPTED`.

If the path isn’t recognized **and** isn’t RPM‑owned, we report `CORRUPTED` as a defensive default.

---

## Missing Path Behavior

If the path does **not** exist, we immediately report `CORRUPTED` with a diagnostic
since the file was expected to exist (based on your corruption targets).

---

## Dependencies Summary

Install via Ansible (already done in `checks/verify_restored.yml`):

- `dracut` — provides `lsinitrd`
- `grub2-tools-minimal` — provides `grub2-editenv`
- `file` — file type detection
- `rpm` — for fallback integrity checks (installed by default on RHEL/Alma)

---

## Examples

```bash
$ bootcheck /boot/initramfs-$(uname -r).img
CLEAN
$ echo $?    # exit code
0

$ bootcheck /boot/vmlinuz-5.14.0-bad
bootcheck: missing: /boot/vmlinuz-5.14.0-bad
CORRUPTED
$ echo $?
1
```

---

## Implementation Notes (High Confidence)

- The script uses `set -euo pipefail` for strict error handling.
- A helper `have()` detects whether a given command exists on the system.
- Classification is by anchored regex on the absolute path.
- `shopt -s nocasematch` is used temporarily for case‑insensitive `file(1)` checks.
- For readability and correctness, it avoids `A && B || C` constructs where `C` may run if `B` fails.
  Instead, it uses `if ...; then ...; else ...; fi` blocks.

---

## Extending `bootcheck`

To add a new boot artifact type:

1. Add a classification flag (e.g., `is_whatever=1`) using a path regex.
2. Add a probe block that returns `CLEAN` or `CORRUPTED` deterministically.
3. Keep probes read‑only. Any mutation belongs in separate remediation tooling.
4. Update Ansible docs if you want results to be bucketed in a new key.
