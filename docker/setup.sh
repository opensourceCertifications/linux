#!/usr/bin/env bash
set -e
LOCATION=$1

# Define file paths
TODO_FILE="$LOCATION/todo.csv"
BREAK_FILE="$LOCATION/break.csv"
README_FILE="$HOME/README.md"

# Function to pick a random line from a CSV file
pick_random_line() {
    shuf -n 1 "$1"
}

# Select a random task and sabotage
TODO_LINE=$(pick_random_line "$TODO_FILE")
BREAK_LINE=$(pick_random_line "$BREAK_FILE")

# Assign values from CSV (assuming format: "task,time,difficulty")
IFS=',' read -r TODO_TASK TODO_TIME TODO_DIFFICULTY <<< "$TODO_LINE"
IFS=',' read -r BREAK_TASK BREAK_TIME BREAK_DIFFICULTY <<< "$BREAK_LINE"

# Print the selected tasks
echo "Selected Task: $TODO_TASK (Time: $TODO_TIME mins, Difficulty: $TODO_DIFFICULTY/10)"
echo "Selected Sabotage: $BREAK_TASK (Time: $BREAK_TIME mins, Difficulty: $BREAK_DIFFICULTY/10)"

# Create a README file for the user
cat <<EOF > "$HOME/$README_FILE"
# Exam Instructions

## Your Task:
- **Task:** $TODO_TASK
- **Estimated Time:** $TODO_TIME minutes
- **Difficulty:** $TODO_DIFFICULTY/10

## Challenge:
- One issue exists in the system. Identify and fix it.

Good luck!
EOF

# Display the README content
cat "$HOME/$README_FILE"

