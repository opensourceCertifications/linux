# Vagrant Snapshots (Multi-VM: `testenv` + `monitor`)

This repo uses a multi-machine Vagrantfile (e.g., `testenv` and `monitor`). **Snapshots are per-VM**, so you save/restore each machine separately.

> **Tip:** If you want the whole lab to return to a consistent baseline, snapshot/restore **both** VMs with the same snapshot name (e.g., `baseline`).

---

## 1) Create a baseline snapshot (first time)

Bring everything up and let provisioning finish:

```bash
vagrant up
```

Optionally stop the VMs (not required for all providers, but often cleaner):

```bash
vagrant halt
```

Save snapshots (one per VM):

```bash
vagrant snapshot save testenv  baseline
vagrant snapshot save monitor  baseline
```

List snapshots:

```bash
vagrant snapshot list
```

---

## 2) Restore snapshots

### Restore **only** `testenv` (most common)
Use this if `testenv` gets borked and you just want it back:

```bash
vagrant snapshot restore testenv baseline
```

### Restore the **whole lab** (`testenv` + `monitor`)
Use this if you want both machines back to a consistent state:

```bash
vagrant snapshot restore testenv baseline
vagrant snapshot restore monitor baseline
```

### Restore without auto-starting (optional)
```bash
vagrant snapshot restore --no-start testenv baseline
vagrant snapshot restore --no-start monitor baseline
```

---

## 3) Delete snapshots

Delete a snapshot from a specific VM:

```bash
vagrant snapshot delete testenv  baseline
vagrant snapshot delete monitor  baseline
```

---

## 4) Quick “stack” workflow (push/pop)

Good for quick experiments:

```bash
vagrant snapshot push testenv
# ...experiment...
vagrant snapshot pop testenv
```

Repeat for `monitor` if needed.

> Don’t mix `push/pop` heavily with `save/restore` unless you know exactly what you’re doing; treat `push/pop` as a separate workflow.

---

## 5) Gotchas

### Synced folders aren’t rolled back
If you use a synced folder like `/vagrant`, those files typically live on your host machine.
Snapshots restore the VM’s disk/state, **not** your host working directory.

### Provider support varies
If `vagrant snapshot ...` says snapshots aren’t supported by your provider, you’ll need to use the provider’s native snapshot tooling (e.g., `virsh snapshot-*` for libvirt/KVM).

---

## 6) Handy one-liners

Save `baseline` for both VMs:

```bash
for m in testenv monitor; do vagrant snapshot save "$m" baseline; done
```

Restore `baseline` for both VMs:

```bash
for m in testenv monitor; do vagrant snapshot restore "$m" baseline; done
```
