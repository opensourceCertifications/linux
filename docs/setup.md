# Setup & Usage

## Prerequisites
- VirtualBox
- Vagrant
- Go (for local development)
- pre-commit (optional, for local linting)

## Bring up the lab
```bash
vagrant up
# (first boot can take a while for box download)
```

## Access VMs
```bash
vagrant ssh monitor
vagrant ssh testenv
```

## Orchestrator (monitor)
On `monitor` VM, run the Go monitor (or build a binary):
```bash
cd /vagrant/monitor/go
# Run directly
GO111MODULE=on go run monitor_logic.go

# Or build and run
go build -o monitor ./monitor_logic.go
./monitor
```

## Ansible checks
From your host or monitor VM, execute the checks playbook:
```bash
cd monitor/ansible
ansible-playbook checks.yml
```
