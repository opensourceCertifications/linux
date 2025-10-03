# Check template — authoring a new validation task

This template shows the recommended **structure and conventions** for creating a new check under
`monitor/ansible/checks/`. Checks should be **read‑only** where possible; put remediation into a separate
task file (e.g., `fix_*.yml`).

> Save a copy of this file as `monitor/ansible/checks/YOUR_CHECK_NAME.yml` and adapt the example code.

---

## Naming conventions

- **File name**: `verify_something.yml` for validation; `fix_something.yml` for remediation.
- **Result keys**: use plural, descriptive names (`things_ok`, `things_bad`). These are merged into `/tmp/results.yml`.
- **Variables**: prefer lower_snake_case for facts you set in a play.

---

## Skeleton

```yaml
---
# 0) Define targets (often from ansible_vars.yml), initialize buckets

- name: Set targets (customize this)
  ansible.builtin.set_fact:
    targets: "{{ corruptedBootFiles | default([]) }}"

- name: Init buckets
  ansible.builtin.set_fact:
    good_items: []
    bad_items: []

- name: Nothing to check?
  ansible.builtin.debug:
    msg: "No targets found; skipping YOUR_CHECK_NAME."
  when: targets | length == 0

# 1) Collect evidence / probe (choose the right module: command, stat, package_facts, etc.)

- name: Probe each target
  ansible.builtin.command:
    argv: ["/usr/bin/true"]   # <— replace with your probe
  register: probe
  changed_when: false
  failed_when: false
  loop: "{{ targets }}"
  loop_control:
    label: "{{ item }}"

# 2) Bucket results based on probe output

- name: Bucket by status
  ansible.builtin.set_fact:
    good_items: >-
      {{ good_items + [ item.item ] if (item.rc|default(1)) == 0 else good_items }}
    bad_items: >-
      {{ bad_items + [ item.item ] if (item.rc|default(1)) != 0 else bad_items }}
  loop: "{{ probe.results | default([]) }}"

# 3) Merge buckets into /tmp/results.yml via helper

- name: Append good_items
  ansible.builtin.include_tasks: "{{ playbook_dir }}/library/append_to_results.yml"
  vars:
    result_key: "good_items"
    new_items: "{{ good_items }}"
  when: good_items | length > 0

- name: Append bad_items
  ansible.builtin.include_tasks: "{{ playbook_dir }}/library/append_to_results.yml"
  vars:
    result_key: "bad_items"
    new_items: "{{ bad_items }}"
  when: bad_items | length > 0
```

---

## Tips

- Use `changed_when: false` for pure probes to keep play recap meaningful.
- Use `failed_when: false` so one item’s failure doesn’t stop the check; bucket it instead.
- Avoid shells unless necessary; prefer `command`, `stat`, `package_facts`, etc.
- If you create per‑check artifacts, consider writing under `/tmp/{{ _cid }}/` when included from `checks.yml`.
