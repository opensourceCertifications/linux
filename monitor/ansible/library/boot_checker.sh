#!/usr/bin/env bash
# bootcheck: verify a single boot-related path on RHEL-family hosts
# stdout: "CLEAN" or "CORRUPTED"
# exit:  0 = CLEAN, 1 = CORRUPTED, 2 = usage error

set -euo pipefail

err() { printf 'bootcheck: %s\n' "$*" >&2; }
clean() {
  printf 'CLEAN\n'
  exit 0
}
corrupt() {
  printf 'CORRUPTED\n'
  exit 1
}

if [[ $# -ne 1 ]]; then
  err "usage: $0 /absolute/path"
  exit 2
fi

path="$1"

# Treat missing path as corrupted (your list says it should exist)
if [[ ! -e "$path" ]]; then
  err "missing: $path"
  corrupt
fi

# Helpers
have() { command -v "$1" > /dev/null 2>&1; }

# ----- Type classification by path pattern (RHEL/Alma) -----
is_initramfs=0
is_kernel=0
is_grubenv=0
is_grubcfg=0
is_bls=0
is_efi=0

[[ "$path" =~ ^/boot/initramfs-.*\.img$ ]] && is_initramfs=1
[[ "$path" =~ ^/boot/vmlinuz- ]] && is_kernel=1
[[ "$path" == "/boot/grub2/grubenv" || "$path" == "/boot/grub/grubenv" ]] && is_grubenv=1
[[ "$path" == "/boot/grub2/grub.cfg" || "$path" == "/boot/grub/grub.cfg" ]] && is_grubcfg=1
[[ "$path" =~ ^/boot/loader/entries/.*\.conf$ ]] && is_bls=1
[[ "$path" =~ ^/boot/efi/EFI/.*\.efi$ ]] && is_efi=1

# ----- Probes (authoritative, read-only) -----

# initramfs: lsinitrd should succeed regardless of compression
# initramfs
if ((is_initramfs)); then
  if ! have lsinitrd; then
    err "lsinitrd not found"
    corrupt
  fi
  if lsinitrd "$path" > /dev/null 2>&1; then
    clean
  else
    corrupt
  fi
fi

# kernel image: `file` must say "linux kernel"
# kernel image
if ((is_kernel)); then
  if ! have file; then
    err "file(1) not found"
    corrupt
  fi
  desc="$(file -b -- "$path" 2> /dev/null || true)"
  shopt -s nocasematch
  if [[ "$desc" =~ linux\ kernel ]]; then
    clean
  else
    corrupt
  fi
fi

# grubenv: grub2-editenv list OK, or header line matches
if ((is_grubenv)); then
  if have grub2-editenv && grub2-editenv "$path" list > /dev/null 2>&1; then
    clean
  else
    # grubenv fallback header
    header="$(head -n1 -- "$path" 2> /dev/null || true)"
    if [[ "$header" == "GRUB Environment Block" ]]; then
      clean
    else
      corrupt
    fi
  fi
fi

# grub.cfg: must contain blscfg OR at least one menuentry
if ((is_grubcfg)); then
  if grep -Eq 'blscfg|menuentry' -- "$path"; then clean; else corrupt; fi
fi

# BLS entry: must include title, linux, initrd keys
if ((is_bls)); then
  grep -Eq '^title' -- "$path" &&
    grep -Eq '^linux' -- "$path" &&
    grep -Eq '^initrd' -- "$path" && clean
else
  corrupt
fi

# EFI binaries: `file` must report EFI application
# EFI binaries
if ((is_efi)); then
  if ! have file; then
    err "file(1) not found"
    corrupt
  fi
  desc="$(file -b -- "$path" 2> /dev/null || true)"
  shopt -s nocasematch
  if [[ "$desc" =~ efi\ application ]]; then
    clean
  else
    corrupt
  fi
fi

# Fallback: if file is RPM-owned, verify just that file; else treat as corrupted
# RPM-owned fallback
if rpm -qf --quiet -- "$path"; then
  out="$(rpm -Vf -- "$path" 2> /dev/null || true)"
  if [[ -z "$out" ]]; then
    clean
  else
    corrupt
  fi
else
  err "unknown type and not RPM-owned: $path"
  corrupt
fi
