#!/bin/bash

# Kibaship Ansible Playbook Execution Script
# Executes Ansible playbook with SSH agent setup in the same shell
# Environment variable KIBASHIP_SSH_PRIVATE_KEY must be set

set -e  # Exit on any error

# Validate required environment variable exists
if [ -z "$KIBASHIP_SSH_PRIVATE_KEY" ]; then
    echo "ERROR: KIBASHIP_SSH_PRIVATE_KEY environment variable is not set" >&2
    exit 1
fi

echo "Starting SSH agent setup..."

# Start SSH agent and evaluate its output to set environment variables
eval "$(ssh-agent -s)" > /dev/null

if [ -z "$SSH_AUTH_SOCK" ] || [ -z "$SSH_AGENT_PID" ]; then
    echo "ERROR: Failed to start SSH agent" >&2
    exit 1
fi

echo "SSH agent started successfully (PID: $SSH_AGENT_PID)"

# Create cleanup function to kill SSH agent on exit
cleanup() {
    if [ -n "$SSH_AGENT_PID" ]; then
        echo "Cleaning up SSH agent (PID: $SSH_AGENT_PID)..."
        kill "$SSH_AGENT_PID" 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit, interrupt, or termination
trap cleanup EXIT INT TERM

# Load private key into SSH agent from environment variable
echo "$KIBASHIP_SSH_PRIVATE_KEY" | ssh-add - > /dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "ERROR: Failed to load SSH private key into agent" >&2
    exit 1
fi

# Verify key was loaded
KEY_COUNT=$(ssh-add -l 2>/dev/null | wc -l)
if [ "$KEY_COUNT" -eq 0 ]; then
    echo "ERROR: No SSH keys found in agent after loading" >&2
    exit 1
fi

echo "SSH key loaded successfully into agent ($KEY_COUNT key(s) available)"

# Execute ansible-playbook with all provided arguments
# The SSH agent environment variables will be inherited
echo "Executing ansible-playbook $*"

# Use the ansible-playbook from the virtual environment
# The path should be provided as the first argument or default to local venv
ANSIBLE_PLAYBOOK_PATH="${ANSIBLE_PLAYBOOK_BIN:-./venv/bin/ansible-playbook}"

if [ ! -f "$ANSIBLE_PLAYBOOK_PATH" ]; then
    echo "ERROR: ansible-playbook not found at $ANSIBLE_PLAYBOOK_PATH" >&2
    exit 1
fi

# Execute ansible-playbook and capture exit code
"$ANSIBLE_PLAYBOOK_PATH" "$@"
ANSIBLE_EXIT_CODE=$?

# Exit with the same code as ansible-playbook
exit $ANSIBLE_EXIT_CODE
