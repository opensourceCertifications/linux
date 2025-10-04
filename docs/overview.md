# Linux Bootloader Chaos Lab

    This repository provisions a **monitor VM** and a **testenv VM** (AlmaLinux 9)
    via **Vagrant/VirtualBox**. The monitor compiles and ships a small Go “break”
    binary to the testenv which deliberately corrupts selected boot files (e.g. `vmlinuz`,
    `initramfs`, GRUB files). A lightweight **TCP+AES‑GCM** channel reports progress
    and results back to the monitor. Ansible playbooks then **verify** the testenv state
    and bucket results.

    ## Repository structure (abridged)
    ```text
    linux-39
├── monitor
│   ├── ansible
│   │   ├── checks
│   │   │   └── verify_restored.yml
│   │   ├── library
│   │   │   ├── append_to_results.yml
│   │   │   └── boot_checker.sh
│   │   └── checks.yml
│   └── go
│       ├── breaks
│       │   ├── broken_boot_loader.go
│       │   └── README.md
│       ├── shared
│       │   ├── library
│       │   │   ├── corrupt_file.go
│       │   │   ├── messages.go
│       │   │   └── README.md
│       │   └── types
│       │       ├── README.md
│       │       └── shared_types.go
│       ├── go.mod
│       ├── go.sum
│       ├── monitor_logic.go
│       └── README.md
├── scripts
│   ├── ensure_bash_header.sh
│   ├── gen_docs.sh
│   └── monitor_script.sh
├── .ansible-lint
├── .gitignore
├── .golangci.yml
├── .pre-commit-config.yaml
├── .prettierignore
├── .prettierrc
├── .rubocop.yml
├── .yamllint
├── package-lock.json
├── package.json
├── README.md
└── Vagrantfile
    ```

    See also: `docs/setup.md`, `docs/ansible/PLAYBOOKS.md`, `docs/go/*`.
