# Linux Chaos Certification Lab
> Two-VM, Go + Ansible–driven lab for hands‑on Linux troubleshooting and recovery.

This repository provisions a **monitor** node and a **testenv** node using Vagrant. The monitor compiles and ships small Go "break" binaries to the test environment, which intentionally **corrupt or alter system state** (e.g., boot artifacts). After you remediate the issues on `testenv`, **Ansible checks** are run *from the monitor* to verify that the system has been restored.

> ⚠️ **Status**: work-in-progress. Interfaces and scenarios may change.

## Topology & Flow

```
Host (your laptop)
    └─ Vagrant
        ├─ monitor  (AlmaLinux 9)  192.168.56.10  ← Go toolchain, orchestrator, Ansible
        └─ testenv  (AlmaLinux 9)  192.168.56.11  ← You fix breaks here

Data flow
  1) monitor compiles a break from `monitor/breaks/` → scp to testenv
  2) break runs on testenv → connects back to monitor over TCP → sends encrypted JSON events
  3) you repair testenv
  4) monitor runs Ansible checks against testenv → collects `/tmp/results.yml` → evaluates
```

## What’s in here

```
project/
.
├── monitor
│   ├── ansible
│   │   ├── ansible_vars.yml
│   │   ├── checks
│   │   │   └── verify_restored.yml
│   │   ├── checks.yml
│   │   ├── library
│   │   │   ├── append_to_results.yml
│   │   │   └── boot_checker.sh
│   │   └── results.yml
│   └── go
│       ├── breaks
│       │   ├── broken_boot_loader.go
│       │   └── prompt.md
│       ├── checks
│       ├── go.mod
│       ├── go.sum
│       ├── monitor_logic.go
│       ├── nohup.out
│       ├── shared
│       │   ├── go.mod.disable
│       │   ├── library
│       │   │   ├── corrupt_file.go
│       │   │   └── messages.go
│       │   └── types
│       │       └── shared_types.go
│       └── transfer
├── README.md
```

### Key pieces
- **Vagrantfile** – Spins up both VMs on a private network (default `192.168.56.0/24`).
- **monitor/** – Go code:
  - `monitor_logic.go` (main): builds a random break, ships it to `testenv`, listens for AES‑GCM–encrypted status messages, and coordinates execution over SSH.
  - `breaks/` (examples): self‑contained Go programs that *cause* specific failures. Example: `broken_boot_loader.go` corrupts boot artifacts.
  - `shared/library/` and `shared/types/`: small helper library used by breaks to send encrypted, length‑prefixed JSON messages back to the monitor.
- **ansible/** – Playbooks run **from the monitor** to validate remediation:
  - `checks.yml` dynamically targets the `TESTENV_ADDRESS` and includes per‑check playbooks from `ansible/checks/`.
  - `checks/verify_restored.yml` verifies boot artifacts (initramfs, kernels, `grubenv`, etc.) and appends structured results to `/tmp/results.yml` on `testenv`.
  - `library/append_to_results.yml` & `library/check_boot_item.yml` are reusable building blocks for checks.

## Requirements

- **Vagrant** 2.3+
- **VirtualBox** 7.x (recommended provider for the default `192.168.56.x` host‑only network)
- **Host OS**: macOS, Linux, or Windows (WSL2)

## Quick start

### 1) Bring up the VMs

```bash
vagrant up
```

- Both VMs use the `almalinux/9` base box.
- A host ↔ VM sync folder `transfer/` is mounted at `/vagrant_transfer` on both VMs.
- The `testenv` provisioner waits until **`/vagrant_transfer/monitor.pub`** exists, and then adds it to **root**’s `authorized_keys`.

If the monitor’s provisioning did not generate a key automatically, do this once on the monitor:

```bash
vagrant ssh monitor
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N ''
cp ~/.ssh/id_ed25519.pub /vagrant_transfer/monitor.pub
# wait ~a few seconds; the testenv provisioner loop will pick it up and install it for root
```

### 2) Ensure toolchain on the monitor

The repo ships a helper script that installs Go 1.24.x and Ansible on AlmaLinux:

```bash
vagrant ssh monitor
sudo /vagrant/vagrant_script.sh    # idempotent
go version                         # verify
ansible --version                  # verify
```

### 3) Run a break from the monitor

1. Tell the monitor where the test VM lives (matches the Vagrantfile):

```bash
export TESTENV_ADDRESS=192.168.56.11
```

2. Start the orchestrator:

```bash
cd /vagrant/monitor
go mod tidy
go run monitor_logic.go
```

What happens:
- The monitor picks a random Go source from `./breaks/`, injects runtime values (`MonitorIP`, `MonitorPort`, `Token`, `EncryptionKey`) via **`-ldflags`**, and compiles to `/tmp/break_tool`.
- It securely copies the binary to `testenv` as **root** and executes it.
- The break connects back to the monitor over TCP, and sends AES‑GCM‑encrypted, length‑prefixed JSON using `shared/types.ChaosMessage`.

### 4) Fix the system on `testenv`

SSH directly if you prefer a terminal on the target VM:

```bash
vagrant ssh testenv
# troubleshoot and repair
```

### 5) Validate with Ansible (from the monitor)

```bash
cd /vagrant/ansible
export TESTENV_ADDRESS=192.168.56.11
ansible-playbook -i localhost, checks.yml
```

The playbook will:
- Dynamically add the host using `$TESTENV_ADDRESS`.
- Run every check playbook in `ansible/checks/` (e.g. `verify_restored.yml`).
- Fetch **`/tmp/results.yml`** from `testenv` into the local `ansible/` folder.

Example `results.yml` (from this repo):

```yaml
corrupted_files:
- /boot/vmlinuz-5.14.0-570.44.1.el9_6.x86_64
```

## Writing a new *break*

Put a new Go program under `monitor/breaks/`. Each break is a standalone `package main` that expects the following **link‑time variables** to be provided by the monitor:

```go
var (
    MonitorIP     string
    MonitorPort   int
    Token         string
    EncryptionKey string
)
```

Use the shared helpers to report progress back to the monitor:

```go
import (
    lib "github.com/opensourceCertifications/linux/shared/library"
)

func main() {
    // do something destructive (carefully!)
    // ...
    _ = lib.SendMessage(MonitorIP, MonitorPort, "chaos_report", "did something", Token, EncryptionKey)
}
```

> Tip: see `breaks/broken_boot_loader.go` for a concrete example.

## Adding a new *check*

Place a new playbook under `ansible/checks/`. It will be picked up automatically by `checks.yml`. Leverage the existing building blocks:

- `library/check_boot_item.yml` – probe/validate a specific boot artifact (kernel, initramfs, `grubenv`, etc.).
- `library/append_to_results.yml` – append a list of items to `/tmp/results.yml` on `testenv` in an idempotent, merge‑safe way.

If your break emits variables back to the monitor (e.g., `corruptedBootFiles`), capture them in `ansible/ansible_vars.yml` and consume them from your checks.

## Configuration

- **IP addresses** – Change `monitor_ip` / `testenv_ip` in the `Vagrantfile` if `192.168.56.0/24` is occupied.
- **SSH keys** – The test VM trusts the monitor by reading `/vagrant_transfer/monitor.pub` during provisioning. If you rotate keys, copy the new pubkey to that path and reprovision `testenv`.
- **Go toolchain** – Managed by `vagrant_script.sh` (Go 1.24.x). Update as needed.
- **Ansible vars** – See `ansible/ansible_vars.yml` for inputs used by checks (e.g., `corruptedBootFiles`).

## Troubleshooting

- **`testenv` stuck waiting for `monitor.pub`** – Generate the key on the monitor and copy it to `/vagrant_transfer/monitor.pub`.
- **Cannot SSH as root** – Confirm the pubkey was appended to `/root/.ssh/authorized_keys` on `testenv` (the provisioner loop does this).
- **Go not found** – Run `sudo /vagrant/vagrant_script.sh` on the monitor.
- **Ansible not found** – The script installs it via `pip --user`. Re‑login or ensure `~/.local/bin` is on your PATH.
- **Listener blocked** – If you added a firewall, ensure the monitor can accept inbound TCP from `testenv`.
- **Port or IP conflicts** – Adjust the private network or provider. VirtualBox host‑only networks work out of the box.

## Security notes

- The monitor uses `ssh.InsecureIgnoreHostKey()` for convenience. Do not use this in production.
- Breaks intentionally corrupt system files. Only use in disposable VMs.
- Messages from breaks are encrypted with AES‑GCM and length‑prefixed before being sent to the monitor.

## License

MIT (or your preferred license). Replace this section as appropriate.
