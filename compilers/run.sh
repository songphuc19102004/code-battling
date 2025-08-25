#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
# Treat unset variables as an error.
set -eu

# --- Configuration ---
# Name of the running worker container
CONTAINER_NAME="my-worker"

# Resource limits for the sandbox
TIME_LIMIT=2          # in seconds
WALL_TIME_LIMIT=5     # in seconds
MEMORY_LIMIT=128000   # in kilobytes (128 MB)

# --- Argument Validation ---
if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <language> <source_file> <input_file>"
    echo "Example: $0 python main.py input.txt"
    echo "Supported languages: python, go, node"
    exit 1
fi

LANGUAGE=$1
SOURCE_FILE=$2
INPUT_FILE=$3

# Check that the necessary files exist on the host
for f in "$SOURCE_FILE" "$INPUT_FILE"; do
    if [ ! -f "$f" ]; then
        echo "Error: File not found at '$f'"
        exit 1
    fi
done

# --- Prerequisite Check ---
if [ ! "$(docker ps -q -f name=^/${CONTAINER_NAME}$)" ]; then
    echo "Error: Worker container '${CONTAINER_NAME}' is not running. Please start it first."
    exit 1
fi

# --- Execution ---

# 1. Create a temporary, unique directory inside the container for staging files.
TEMP_DIR_IN_CONTAINER=$(docker exec "$CONTAINER_NAME" mktemp -d)

# 2. Ensure the temporary staging directory is cleaned up when this script exits.
trap 'docker exec "$CONTAINER_NAME" rm -rf "$TEMP_DIR_IN_CONTAINER"' EXIT

# 3. Copy source code and input file into the temporary staging directory.
docker cp "$SOURCE_FILE" "${CONTAINER_NAME}:${TEMP_DIR_IN_CONTAINER}/source"
docker cp "$INPUT_FILE" "${CONTAINER_NAME}:${TEMP_DIR_IN_CONTAINER}/input.txt"

# 4. Execute the sandbox workflow inside the container.
docker exec -i "$CONTAINER_NAME" /bin/bash -s \
    "$TIME_LIMIT" "$MEMORY_LIMIT" "$WALL_TIME_LIMIT" "$LANGUAGE" "$TEMP_DIR_IN_CONTAINER" <<'EOF'

# This entire block runs inside the 'my-worker' container
set -euo pipefail

# --- Container-Side Variables ---
TIME_LIMIT_VAR=$1
MEMORY_LIMIT_VAR=$2
WALL_TIME_LIMIT_VAR=$3
LANGUAGE_VAR=$4
TEMP_DIR_VAR=$5

# --- Isolate Setup ---
BOX_ID=$(shuf -i 0-999 -n 1)
trap 'isolate --box-id=${BOX_ID} --cleanup' EXIT

# --- Isolate Workflow ---
# 1. Initialize the sandbox.
BOX_PATH=$(isolate --box-id="${BOX_ID}" --init)

# 2. Move the staged files into the sandbox's writable 'box' directory.
mv "${TEMP_DIR_VAR}/source" "${BOX_PATH}/box/source"
mv "${TEMP_DIR_VAR}/input.txt" "${BOX_PATH}/box/input.txt"

# 3. Determine the command to run. This string will be executed by bash.
COMMAND_TO_RUN=""
case "$LANGUAGE_VAR" in
  "python")
    COMMAND_TO_RUN="/usr/local/python-3.13.6/bin/python script.py"
    ;;
  "go")
    # First, compile the Go program. We must provide the PATH here too.
    isolate --box-id="${BOX_ID}" \
            -E PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
            -E HOME=/tmp \
            -d /etc:noexec \
            --run -- /usr/local/go/bin/go build -o app_binary source
    # The command to run is now the compiled binary.
    COMMAND_TO_RUN="./app_binary"
    ;;
  "node")
    COMMAND_TO_RUN="/usr/local/bin/node source"
    ;;
  *)
    echo "Error: Unsupported language '$LANGUAGE_VAR'."
    exit 1
    ;;
esac

# 4. Run the user's code inside the sandbox.
#    *** THIS IS THE FIX, mimicking Judge0's method ***
#    Instead of --dir=/usr, we set the PATH environment variable inside the sandbox.
echo "--- Running Isolate ---"
isolate \
    --box-id="${BOX_ID}" \
    --time="${TIME_LIMIT_VAR}" \
    --wall-time="${WALL_TIME_LIMIT_VAR}" \
    --mem="${MEMORY_LIMIT_VAR}" \
    -E HOME=/tmp \
    -E PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    -d /etc:noexec \
    --stdin=input.txt \
    --stdout=output.txt \
    --stderr=errors.txt \
    --meta=meta.txt \
    --run -- /bin/bash -c "${COMMAND_TO_RUN}"

# 5. Display results.
echo
echo "--- Standard Output (output.txt) ---"
cat "${BOX_PATH}/output.txt"
echo

echo "--- Standard Error (errors.txt) ---"
cat "${BOX_PATH}/errors.txt"
echo

echo "--- Isolate Metadata (meta.txt) ---"
cat "${BOX_PATH}/meta.txt"
echo

echo "--- Isolate cleanup will be performed automatically ---"
EOF

echo "--- Host cleanup will be performed automatically ---"
