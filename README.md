```markdown
# Linux Certification Project

This repository is part of a **work-in-progress Linux certification project**.  
It uses [Vagrant](https://www.vagrantup.com/) to provision a small environment with two virtual machines and introduces Go-based tooling to explore system behavior.  

⚠️ **Note:** This project is still under active construction and is **not yet ready for use**.

---

## 📂 Project Structure

```

Vagrantfile
README.md
monitor/
├── breaks
│   └── broken\_boot\_loader.go
├── go.mod
├── go.sum
├── imports.json
├── monitor\_logic.go
└── shared
├── go.mod
├── library
│   ├── corrupt\_file.go
│   └── messages.go
└── types
└── shared\_types.go

````

---

## 🖥️ Virtual Machines

### **monitor**
- Provisions Go (`/usr/local/go`).
- Hosts and runs Go source files in the `monitor/` directory.
- Manages compilation and execution of break modules.
- Handles encrypted message passing and logging.

### **testenv**
- Minimal Linux environment.
- Currently only runs updates and upgrades.
- Serves as the target system for testing break modules.

---

## ⚙️ Go Components

### **monitor/monitor_logic.go**
- Opens a TCP listener on a random port.
- Generates a **token** and an **encryption key**.
- Randomly selects a break module from `breaks/`, compiles it, and deploys it to `testenv`.
- Transfers the binary to `/tmp` on `testenv` and executes it as root.
- Processes encrypted log messages:
  - Validates messages with token + encryption key.
  - Logs accepted messages (e.g., `chaos_report` → JSON log).
  - Closes the port on `"operation_complete"`.
- Intended to loop every **7–10 minutes**.

---

### **monitor/breaks/broken_boot_loader.go**
- Simulates a **boot failure** by corrupting a critical boot file.

---

### **monitor/shared/library/corrupt_file.go**
- Corrupts files at the byte level:
  - Input: file path + corruption percentage.
  - Randomly selects that % of bytes.
  - Flips bits (`0 ↔ 1`) and adds a random number for additional corruption.

---

### **monitor/shared/library/messages.go**
- Builds and encrypts messages:
  - Input: IP, port, token, type, payload, key.
  - Encrypts the message and sends it to the target.

---

### **monitor/shared/types/shared_types.go**
- Defines shared objects between modules.
- Currently includes only the `Message` struct.

---

## 🚀 Getting Started

### Requirements
- [Vagrant](https://developer.hashicorp.com/vagrant/downloads)
- [VirtualBox](https://www.virtualbox.org/) (or another provider)

### Setup
```bash
git clone https://github.com/<your-username>/<your-repo>.git
cd <your-repo>
vagrant up # run from within the root of the project where you see the Vagrantfile
````

### Usage

SSH into the **monitor** VM and run the Go logic:

```bash
vagrant ssh monitor
cd /vagrant/monitor
go run monitor_logic.go
```

SSH into the **testenv** VM fix the broken stuff:

```bash
vagrant ssh testenv
```
---

## ⚠️ Disclaimer

This project is **not production ready** and is part of a **Linux certification build**.
It may deliberately corrupt files or simulate destructive operations.
Run only inside the provided Vagrant environment.

---

## 📌 Roadmap

* Expand break modules (e.g., file system, networking, memory stress).
* Add tasks for the user to do in the `testenv` VM.
* Improve logging and reporting.
* Documentation for certification steps.

---

## 📝 License

This project is released under the MIT License. See [LICENSE](LICENSE) for details.

```

Do you also want me to add a **Mermaid diagram** in the README showing how `monitor` communicates with `testenv` and runs the breaks? That would make the architecture pop nicely on GitHub.
```
