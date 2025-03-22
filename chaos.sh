#!/bin/bash

set -euo pipefail

ANSIBLE_DIR="/vagrant/ansible/BROKEN/breaks"

while true; do
    # Pick a random delay between 180 and 600 seconds (3–10 minutes)
    DELAY=$((RANDOM % (600 - 180 + 1) + 180))
    echo "[chaos] Sleeping for $DELAY seconds..."
    sleep "$DELAY"

    # Find all .yml and .yaml Ansible files
    MAPFILE=()
    while IFS= read -r -d $'\0' file; do
        MAPFILE+=("$file")
    done < <(find "$ANSIBLE_DIR" -type f \( -name "*.yml" -o -name "*.yaml" \) -print0)

    if [ "${#MAPFILE[@]}" -eq 0 ]; then
        echo "[chaos] No Ansible playbooks found in $ANSIBLE_DIR. Skipping this round."
        continue
    fi

    # Pick a random file from the list
    INDEX=$((RANDOM % ${#MAPFILE[@]}))
    PLAYBOOK="${MAPFILE[$INDEX]}"

    echo "[chaos] Running Ansible playbook: $PLAYBOOK"
    ansible-playbook "$PLAYBOOK" || echo "[chaos] Failed to run: $PLAYBOOK"
done
