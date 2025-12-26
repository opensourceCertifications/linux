# 🧨 Linux Chaos Certification — Failure Families Catalog

The following table defines **40 parameterized chaos families** distributed across **10 domains**.
Each family represents a category of break the monitor can introduce during the exam.
Every family has multiple tunable parameters, allowing variant generation and randomization.

---

## **Chaos Families**

| **Family Name** | **What It Covers** | **Parameters** |
|-----------------|--------------------|----------------|
| **Bootloader Corruption** | Break GRUB/BLS so next boot fails or selects wrong kernel. | Target file (`/boot/grub2/grub.cfg`, `grubenv`, BLS `.conf`); corruption method (truncate, flip bits, inject invalid line); scope (% bytes); default entry toggle; timing (immediate, delayed) |
| **Kernel Image Tamper** | Corrupt `vmlinuz-*` so reboot panics or drops to rescue. | Kernel to target (default/current/old); method (overwrite segment, random span, zero tail); bytes to corrupt; hash mismatch injection; restore path (cached copy vs `dnf reinstall`) |
| **Initramfs Sabotage** | Damage `initramfs` modules/hooks so root mount fails on reboot. | Target initramfs; method (remove `xfs.ko`, corrupt `init` script, drop `dracut` hook); dracut conf edits; compression type change; regen suppression (mask `dracut` timer) |
| **GRUB Device Map Lies** | Mispoint GRUB root to wrong disk/UUID. | Edit type (UUID swap, PARTUUID swap, wrong `set root=`); scope (default entry only, all entries); UEFI vs BIOS path; persistence (also edit `/etc/default/grub`?) |
| **Root FS Read-Only Flip** | Remount `/` read-only or set fstab to ro next boot. | Live remount vs fstab edit; target (`/`, `/usr`); add `errors=remount-ro`; delay (timer); with/without `systemd-remount-fs` override |
| **Inode Starvation** | Exhaust inodes so writes fail despite free space. | Filesystem (`/`, `/var`, `/tmp`); file count to create; file size; cleanup persistence; hide via chattr +a |
| **LVM Mapping Drift** | Break LV mapping or hide PV so volumes vanish. | Action (rename VG, remove `/etc/lvm/cache`, add `filter` to hide PV); target LV; persistence across reboot; add udev rule to mask device |
| **RAID Degradation Trap** | Mark member failed or desync, performance tanks or mount degrades. | Array (`/dev/md*`); member to fail; resync speed limit; bitmap tweak; mdadm conf edits; simulate bad sector with `dmsetup` |
| **RPM DB Poisoning (soft)** | Cause rpmdb/read failures without total brick. | Action (lock file left, stale Berkeley DB pages, partial copy); path (`/var/lib/rpm`); inject missing provides; recovery hint toggles |
| **Critical Lib Tamper (glibc)** | Remove/corrupt `/lib64/libc.so.6` to stop new dynamic execs. | File target (`libc`, `libpthread`, loader `ld-linux`); method (rename, truncate, flip); stash clean copy path; pair with `noexec` on temp; delay |
| **`ld.so.preload` Trap** | Add invalid or malicious `.so` so all new processes fail. | Path to preload; content (nonexistent, incompatible arch, interposer); scope (root only, all users); timing; backup drop |
| **Symlink Bait & Switch** | Flip symlinks for core tools to wrong binaries. | Targets (`/bin/sh`, `/usr/bin/python`, `/usr/bin/systemctl`); destination (older ver, wrong arch, script); persistence (immutable bit on link) |
| **GPG/RPM Key Drift** | Remove/trust wrong keys so `dnf` fails or installs from wrong repo. | Action (delete key, add rogue key, expire time); repo target; `sslverify` toggle; mirror priority manipulation |
| **Systemd Default Target Twist** | Switch to `rescue`/`emergency` or isolate a partial target on boot. | Target (`rescue`, `multi-user`, custom); method (symlink default, `systemctl set-default`, edit unit alias); drop conflicting overrides |
| **Unit Drop-in Saboteur** | Inject drop-ins that intermittently fail services. | Target unit(s); `ExecStartPre` random gate; `RestartSec` inflation; environment poisoning; timer-based toggle |
| **Timer Bombs** | Hidden timers invoke disruptive scripts periodically. | Timer cadence (cron-like, random jitter); service payload; unit location (`~/.config/systemd/user`, `/etc/systemd/system`); persist enablement |
| **PID1 Interaction Limits** | Modify systemd rate limits / default limits to cause spurious failures. | `DefaultLimitNOFILE/CORE/NPROC`; `StartLimitBurst/Interval`; per-unit `TasksMax`; persistence |
| **PBR Blackhole** | Route specific CIDR/host through blackhole table. | CIDR target; table id/name; rule priority; IPv4/IPv6; add matching `ip neigh` poison |
| **Egress `tc netem` Pain** | Add delay/loss/reorder excluding ssh. | Interface; loss/delay/reorder %; exclude filter (tcp/22); attach point (root/clsact); persistence |
| **nftables Shadow Chain** | Earlier-priority base chain short-circuits real policy. | Family/table/chain names; hook/prio; action (accept/drop/reset); random handle ids; rule counters off |
| **DNS Chaos** | Probabilistic or conditional DNS failures. | Drop %; UDP vs TCP 53; per-resolver selection; `/etc/resolv.conf` tamper (rotate, timeout); `systemd-resolved` cache flush loop |
| **PAM/SSH Friction (future sessions)** | New sessions fail; current stays. | PAM module insert (`pam_time`, `pam_listfile`); limits.d caps; `DenyUsers` without reload; nftables NEW only |
| **Sudo Subtlety** | Sudo works sometimes, then fails via rules ordering. | Insert earlier `%wheel` rule with NOPASSWD conflict; timestamp_timeout; lecture file lock; per-TTY restriction |
| **Account State Gotchas** | Locked/expired users, shell changed, homedir perms. | `chage -E` date; `passwd -l`; shell to `/sbin/nologin`; home `0700`/ownership swap; `faillock` tripwire |
| **Known_hosts/Hostkey Drift** | Force strict hostkey mismatch to block automation. | Replace server hostkey; rotate client `known_hosts`; `HostKeyAlgorithms` change; revocation lists |
| **Chrony Skew & Jitter** | Time skew breaks TLS, Kerberos, caches. | Offset magnitude; source (fake NTP, manual step); chrony conf edits (minsources, makestep off); slew vs step |
| **Timer Misfires** | cron/systemd timer schedules drift or collide. | Modify timezones; `OnCalendar` skew; missed job handling; anacron toggles; inhibit locks |
| **Clocksource Swap** | Switch to unstable clocksource to cause weird timing. | `tsc`/`hpet`/`kvm-clock` selection; NTP on/off; `maxslewrate`; introduce CPU steal via cgroup |
| **journald Pressure** | Journald drops logs or rate-limits too aggressively. | `SystemMaxUse`, `RuntimeMaxUse`; `RateLimitInterval/Burst`; storage to `volatile`; fs quota on `/var/log/journal` |
| **rsyslog Pipeline Break** | Stop remote/local forwarding silently. | Drop imjournal/imuxsock rules; add discard rules; HUP suppression; TLS cert path swap; queue limits |
| **logrotate Saboteur** | Rotate too often or never; permissions wrong. | Frequency; `create` mode; `postrotate` removed; state file corrupt; compress options |
| **Metric Agent Blindness** | Mangle node exporter/collectd so graphs lie. | Bind to wrong iface; filter out labels; scrape path change; excessive `--collector.disable-defaults` |
| **cgroup v2 Memory Squeeze** | Throttle `user.slice`/services with `memory.high/max`. | Slice/Scope; thresholds; reclaim enablement; PSI monitoring off; drop-in persist |
| **CPU Quota Clamp** | Quietly cap CPUs for shells or critical services. | `CPUQuota` %; `AllowedCPUs` mask; `nice`/`ionice`; sched policy; attach/teardown timing |
| **FD/Proc Limits** | Exhaust file descriptors or NPROC to block new work. | Leak rate (FD/s); scope (per-user, system); limits.d values; hidden leaker unit; cleanup guard |
| **Disk Fill Patterns** | Fill disk subtly to trip journaling or tmp usage. | Target FS; file type (sparse vs real); hidden path; watermark %; throttle |
| **Env Poisoning** | Break programs by altering key env vars at session/service levels. | Inject in `/etc/environment`, unit `Environment=`, `/etc/profile.d`; targets (`LD_PRELOAD`, `PATH`, `SSL_CERT_FILE`, `NO_PROXY`) |
| **Socket Path Swaps** | Point services to wrong UNIX sockets/FIFO names. | Swap `/run/*.sock`; create look-alike sockets; adjust perms/SELinux type; stale pidfile confusion |
| **Tempdir/Sticky Bit Games** | Break apps relying on `/tmp` semantics. | Remove sticky bit; mount `noexec`/`nosuid`; quota on `/tmp`; symlink farm from `/tmp` to elsewhere |
| **PIDFile/Lock Contention** | Fake stale pid/lock files to block starts. | Target service; create pid with live PID; `flock` contention via background process; cleanup timing |

---

### 🧩 Next Steps

1. Pick a domain to focus on first (e.g., *Boot & Kernel*).
2. For each family in that domain, open an implementation ticket:
   - Define exact break command(s).
   - Implement a Go chaos module and matching Ansible verification.
   - Include rollback logic and randomness knobs.
3. Iterate and test with your two-VM harness (`monitor` + `testenv`).
