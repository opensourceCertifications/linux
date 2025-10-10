# Chaos Certification Prompt

You are helping me design a **Linux chaos engineering certification**.

The exam runs on small Linux VMs (t3.small size: 2 vCPU, 2 GiB RAM). The exam monitor randomly executes “breaks” every 7–10 minutes. Candidates must detect and repair them.

---

## Important Context
- Breaks are grouped into **families**.
- A *family* = a general category of failure (e.g., “bootloader corruption”).
- Each family can have multiple **variants/parameters** (e.g., corrupt `grub.cfg`, corrupt `grubenv`, break BLS entry).
- The goal is breadth and unpredictability: candidates shouldn’t be able to pre-script watchdogs cheaply.

---

## Your Task
Scope out **40 parameterized failure families** in a **table format**.

For each family, include:
1. **Family Name** – short, descriptive.
2. **What It Covers** – a concise description of the type of break and its impact.
3. **Parameters** – 3–6 knobs or variants I can use to generate multiple concrete breaks within the family (e.g., target path, corruption intensity, scope, timing).

The table should be structured so I can create tickets for each family with a clear “definition of done” for implementation. Keep each row tight and clear enough that an engineer could immediately implement scripts based on it.

---

## 10 Domains to Distribute Families Across
1. **Boot & Kernel** – kernel images, initramfs, GRUB, BLS
2. **Filesystem & Storage** – disk space, inodes, LVM, RAID
3. **Packages & Libraries** – rpmdb, critical libs, symlinks, preload
4. **Services & Init** – systemd targets, units, timers
5. **Networking** – routes, DNS, MTU, firewall rules
6. **Authentication & Users** – PAM, sudoers, SSH, passwd/shadow
7. **Time & Scheduling** – NTP, clock skew, cron/systemd timers
8. **Observability & Logging** – journald, rsyslog, logrotate
9. **Resource Limits** – cgroups, ulimits, CPU/mem/fd exhaustion
10. **Application/Runtime Environment** – env vars, sockets, tempdirs, pidfiles

---

## Format Example (first row)

| Family Name           | What It Covers                                                                 | Parameters                                                                                      |
|-----------------------|---------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| Bootloader Corruption | Breaks the bootloader so the VM cannot boot cleanly. Covers GRUB and BLS logic. | - Target file (`/boot/grub2/grub.cfg`, `/boot/grub2/grubenv`, BLS `.conf`) <br> - Corruption method (truncate, invert bits, random offset) <br> - Corruption scope (% of file) <br> - Timing (immediate vs delayed execution) |

---

## Request
**Now, please generate a table with 40 rows (families) distributed across the 10 domains above, in this format.**
