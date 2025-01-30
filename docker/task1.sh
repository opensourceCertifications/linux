#!/bin/bash

# Constants
TEST_DURATION=600  # Test duration in seconds (10 minutes)
BROKEN_TASKS=("network_issue" "ssh_misconfig" "cron_failure")
TASKS=("install_tools" "configure_firewall" "update_packages")
README_FILE=/README.txt

# Function to set up the environment
setup_environment() {
    echo "Setting up environment..."
    yum install -y vim sudo curl > /dev/null 2>&1
    echo "Environment setup complete."
}

# Randomly select tasks for the user
choose_tasks() {
    echo "Selecting tasks..."
    CHOSEN_TASKS=()
    while [[ ${#CHOSEN_TASKS[@]} -lt 2 ]]; do
        TASK=${TASKS[$RANDOM % ${#TASKS[@]}]}
        if [[ ! " ${CHOSEN_TASKS[@]} " =~ " ${TASK} " ]]; then
            CHOSEN_TASKS+=("$TASK")
        fi
    done
    echo "Selected tasks: ${CHOSEN_TASKS[@]}"
}

# Break a random system component
break_system() {
    BROKEN_COMPONENT=${BROKEN_TASKS[$RANDOM % ${#BROKEN_TASKS[@]}]}
    echo "Breaking the system: $BROKEN_COMPONENT"
    case $BROKEN_COMPONENT in
        "network_issue")
            nmcli networking off || true ;;
        "ssh_misconfig")
            mv /etc/ssh/sshd_config /etc/ssh/sshd_config.bak || true ;;
        "cron_failure")
            systemctl stop crond || true ;;
    esac
    echo "Broken component: $BROKEN_COMPONENT"
}

# Create a README file with instructions
create_readme() {
    echo "Creating README file..."
    cat <<EOF > $README_FILE
Welcome to the DevOps Challenge!

Tasks to Complete:
1. ${CHOSEN_TASKS[0]}
2. ${CHOSEN_TASKS[1]}

NOTE: One system component is broken. Diagnose and fix it!

Time Remaining: Use the timer output to track your progress.
EOF
    echo "README created at $README_FILE."
}

# Start the countdown timer
timer() {
    echo "Starting the timer..."
    SECONDS_LEFT=$TEST_DURATION
    while [[ $SECONDS_LEFT -gt 0 ]]; do
        echo -ne "Time left: $(($SECONDS_LEFT / 60)) minutes and $(($SECONDS_LEFT % 60)) seconds\\r"
        sleep 1
        ((SECONDS_LEFT--))
    done
    echo -e "\\nTime's up!"
}

# Verify if the user fixed the broken component
verify_fix() {
    echo "Verifying the fixes..."
    case $BROKEN_COMPONENT in
        "network_issue")
            nmcli networking connectivity check && echo "Network fixed!" || echo "Network still broken!" ;;
        "ssh_misconfig")
            [[ -f /etc/ssh/sshd_config ]] && echo "SSH config restored!" || echo "SSH config still broken!" ;;
        "cron_failure")
            systemctl is-active crond && echo "Cron is running!" || echo "Cron is still broken!" ;;
    esac
}

# Main function
main() {
    setup_environment
    choose_tasks
    break_system
    create_readme
    timer
    verify_fix
}

main

