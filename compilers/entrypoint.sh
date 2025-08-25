#!/bin/sh

# Exit immediately if a command exits with a non-zero status.
set -e

# Run the isolate initialization command.
echo "Initializing isolate sandboxes..."
isolate --cg --init

# The `exec` command is important. It replaces the shell process with
# the tail command, which allows the container to receive signals
# (like `docker stop`) correctly.
echo "Initialization complete. Worker is ready."
exec tail -f /dev/null
