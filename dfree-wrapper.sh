#!/bin/bash
# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "[$(date)] Called with: $@" >> /tmp/dfree.log
"$SCRIPT_DIR/anemone-dfree" "$@" 2>> /tmp/dfree.log
