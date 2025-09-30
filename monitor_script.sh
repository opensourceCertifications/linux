#!/usr/bin/env bash
# monitor_script.sh
# Purpose: Prepare the "monitor" VM with Go and Ansible (user-scoped), plus basic deps.
# Runs as root from Vagrant provisioning.

set -euo pipefail

# -----------------------------
# 1) System refresh + basic tools
# -----------------------------
dnf update -y       # bring package metadata and packages up to date
dnf install -y wget # wget used below to fetch Go tarball

# -----------------------------
# 2) Install Go toolchain (system-wide)
#    - Downloads the specified version
#    - Unpacks to /usr/local/go (standard GOROOT location)
#    - Exposes go in PATH for all users via /etc/profile.d
# -----------------------------
GO_VER="1.24.5"
GO_TGZ="go${GO_VER}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TGZ}"

# Download to /tmp (idempotent-friendly: only if not already present)
if [[ ! -f "/tmp/${GO_TGZ}" ]]; then
  wget -P /tmp/ "${GO_URL}"
fi

# Remove any previous /usr/local/go to avoid mixing versions (optional, uncomment if desired)
# rm -rf /usr/local/go

# Unpack into /usr/local
tar -xvf "/tmp/${GO_TGZ}" -C /usr/local

# Clean up tarball
rm -f "/tmp/${GO_TGZ}"

# Set GOROOT for all users (picked up at next login)
# /etc/environment is a simple KEY=VALUE file (no shell expansion), so write literal values.
echo 'GOROOT=/usr/local/go' >> /etc/environment

# Put Go binaries on PATH for all shells via a profile.d snippet
# Use \$ to avoid immediate expansion when writing the file.
bash -c 'echo "export PATH=\$PATH:/usr/local/go/bin" > /etc/profile.d/go.sh'
chmod 644 /etc/profile.d/go.sh

# -----------------------------
# 3) Install Ansible for the vagrant user (user-site)
#    - Keep system Python clean; install ansible in vagrant’s ~/.local
# -----------------------------
dnf install -y python3-pip # ensure pip is available

# Install ansible into the vagrant user’s home (~/.local/bin/ansible, etc.)
# Using su -c executes the command as the vagrant user; --user avoids system-wide changes.
su vagrant -c "python3 -m pip install --user ansible"

# Notes:
# - Go is available to all users after re-login (or `source /etc/profile.d/go.sh`).
# - The 'vagrant' user will have Ansible in ~/.local/bin; ensure ~/.local/bin is on PATH
#   (it usually is by default on modern distros; if not, add it to vagrant’s ~/.bashrc).
