chaostok/
├── Vagrantfile                         # 🚀 Defines the AlmaLinux VM; used to build and upload the final chaos-infested image
├── install.sh                            # 🛠️  Main installer run by the Vagrant provisioner to kick off infection/mutation
├── chaos.service                       # ⚙️  Systemd service to launch chaos agent on boot
│
├── agent/                              # 🧠 Core logic of the chaos daemon
│   ├── __init__.py
│   ├── engine.py                       # Selects sabotage actions based on system state and rules
│   ├── scheduler.py                    # Triggers sabotage every X mins (3–7)
│   ├── sandbox.py                      # Detects if we're in a VM and guards against running on host
│   └── logger.py                       # Handles hidden/in-memory logging of sabotage actions
│
├── sabotage_modules/                  # 💣 Individual sabotage functions (pluggable)
│   ├── __init__.py
│   ├── inject_bom.py                  # Adds invisible BOMs to critical files
│   ├── disable_aliases.py            # Breaks common shell aliases
│   ├── corrupt_apt.py                # Modifies repo sources to use bad mirrors
│   ├── break_cron.py                 # Corrupts crontabs or disables cron service
│
├── checker/                            # ✅ Validation tools (can be run externally or locally)
│   ├── verify.py                      # Reads hidden/in-memory logs and checks if system was healed
│   └── report.py                      # Generates human-readable output or score
│
├── config/                             # ⚙️  Configuration files
│   ├── rules.yaml                     # Sabotage rules & weights
│   └── whitelist.yaml                # Services/files never to touch (e.g., sshd)
│
├── payloads/                           # 📦 (To be added) Contains backup infected binaries or wrapper templates
│   ├── cron                           # Infected binary used for re-infection if healed
│   ├── dbus-daemon
│   └── ...
│
├── mutator/                            # 🧬 Mutation logic for selecting new process hosts and generating wrappers
│   ├── rotate.py                      # Picks a new host binary and infects it
│   ├── checksum.py                    # Verifies and tracks infected binary integrity
│   └── recover.py                     # Re-injects if cleaned (downloads replacement or copies from payloads/)
│
├── watchdog/                           # 🛡️ Early-boot binary/service to detect system healing
│   ├── init_watchdog.py              # Embedded in early boot; triggers reboots or alerts on tampering
│   └── watchdog.service              # Systemd unit file for early launch
│
├── requirements.txt                   # 🧪 Python dependencies
└── README.md                          # 📘 Docs on what the project is, how to run it, goals, etc.

