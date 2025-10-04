# Helper: `library/append_to_results.yml` — merge lists into `/tmp/results.yml`

This helper consolidates results from **any check** by appending a list of items under a **key** in
`/tmp/results.yml` on the testenv. It deduplicates entries and writes a nicely formatted mapping.

- **Location**: `monitor/ansible/library/append_to_results.yml`
- **Called by**: any check that needs to merge results (`include_tasks`)

---

## Contract

**Inputs (required via `vars:`):**

- `result_key` *(string)* — the key to update in the results mapping, e.g., `restored_files`.
- `new_items` *(list[string])* — items to append under that key.

**Behavior:**

1. Reads existing `/tmp/results.yml` if present (base64 + YAML decode via `slurp`).
2. Builds a working dict (`_existing`) or `{}` if the file doesn’t exist yet.
3. Merges: `existing[result_key] + new_items` and applies `| unique` to deduplicate.
4. Writes the merged mapping back to `/tmp/results.yml` with `mode: 0644` and `backup: yes`.

The task avoids failures on missing files and is **idempotent**.

---

## Example usage

```yaml
- name: Append restored files
  ansible.builtin.include_tasks: "{{ playbook_dir }}/library/append_to_results.yml"
  vars:
    result_key: "restored_files"
    new_items: "{{ restored_files }}"
  when: restored_files | length > 0
```

**Result file shape:**

```yaml
restored_files:
  - /boot/grub2/grub.cfg
corrupted_files:
  - /boot/initramfs-5.14.0-...img
```

Multiple checks can append to different keys in the same run; this file is fetched to the repo by
`checks.yml` as `monitor/ansible/results.yml`.
