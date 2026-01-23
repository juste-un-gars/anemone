#!/bin/bash
# cleanup-keep-data.sh
# Simulates an OS reinstall while preserving Anemone data
# Used to test the "Import existing installation" feature
#
# Usage: sudo ./scripts/cleanup-keep-data.sh [data_dir]
# Default data_dir: /srv/anemone

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Data directory (default or from argument)
DATA_DIR="${1:-/srv/anemone}"

echo -e "${YELLOW}=== Anemone Cleanup (preserving data) ===${NC}"
echo ""
echo -e "Data directory: ${GREEN}$DATA_DIR${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (sudo)${NC}"
    exit 1
fi

# Check if data directory exists
if [ ! -d "$DATA_DIR" ]; then
    echo -e "${RED}Error: Data directory not found: $DATA_DIR${NC}"
    exit 1
fi

# Check if database exists
if [ ! -f "$DATA_DIR/db/anemone.db" ]; then
    echo -e "${RED}Error: Database not found: $DATA_DIR/db/anemone.db${NC}"
    exit 1
fi

# Confirm
echo -e "${YELLOW}This will remove Anemone installation but KEEP the data in $DATA_DIR${NC}"
echo ""
read -p "Continue? [y/N] " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "--- Retrieving users from database ---"
USERS=$(sqlite3 "$DATA_DIR/db/anemone.db" "SELECT username FROM users WHERE is_admin=0;" 2>/dev/null || true)
if [ -n "$USERS" ]; then
    echo "Found users: $USERS"
else
    echo "No regular users found"
fi

echo ""
echo "--- Stopping services ---"
systemctl stop anemone 2>/dev/null || echo "Service anemone not running"
systemctl disable anemone 2>/dev/null || echo "Service anemone not enabled"

echo ""
echo "--- Removing binaries ---"
rm -f /usr/local/bin/anemone
rm -f /usr/local/bin/anemone-dfree
echo "Removed /usr/local/bin/anemone*"

echo ""
echo "--- Removing systemd service ---"
rm -f /etc/systemd/system/anemone.service
systemctl daemon-reload
echo "Removed systemd service"

echo ""
echo "--- Removing configuration ---"
rm -rf /etc/anemone
echo "Removed /etc/anemone/"

rm -f /etc/sudoers.d/anemone
echo "Removed /etc/sudoers.d/anemone"

echo ""
echo "--- Removing system users ---"
for user in $USERS; do
    if id "$user" &>/dev/null; then
        userdel "$user" 2>/dev/null && echo "Removed user: $user" || echo "Failed to remove: $user"
    else
        echo "User not found (already removed?): $user"
    fi
done

echo ""
echo "--- Cleaning Samba configuration ---"
# Remove generated smb.conf
if [ -f "$DATA_DIR/smb/smb.conf" ]; then
    rm -f "$DATA_DIR/smb/smb.conf"
    echo "Removed $DATA_DIR/smb/smb.conf"
fi

# Restart Samba to apply changes
systemctl restart smbd 2>/dev/null || echo "Could not restart smbd"

echo ""
echo -e "${GREEN}=== Cleanup complete ===${NC}"
echo ""
echo "Data preserved in: $DATA_DIR"
echo ""
ls -la "$DATA_DIR/"
echo ""
echo -e "${YELLOW}To test the import feature:${NC}"
echo "  1. Rebuild: go build -o anemone cmd/anemone/main.go"
echo "  2. Install: sudo ./install.sh"
echo "  3. In the wizard, choose 'Import existing installation'"
echo "  4. Enter path: $DATA_DIR"
echo ""
