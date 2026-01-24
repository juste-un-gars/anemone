#!/bin/bash
#
# simulate-reinstall.sh
# Simulates an OS reinstall by removing system components while preserving data.
# Use this to test the repair mode of install.sh
#
# After running this script:
#   sudo ./install.sh  → Choose option 2 (Repair/Reinstall)
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
DATA_DIR="${ANEMONE_DATA_DIR:-/srv/anemone}"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  Anemone - Simulate OS Reinstall${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "This script will ${RED}REMOVE${NC}:"
echo "  - Anemone systemd service"
echo "  - Anemone binaries (/usr/local/bin/anemone*)"
echo "  - Anemone system user"
echo "  - Samba configuration for Anemone"
echo "  - Samba user entries for Anemone users"
echo ""
echo -e "This script will ${GREEN}KEEP${NC}:"
echo "  - Database: ${DATA_DIR}/db/"
echo "  - Shares: ${DATA_DIR}/shares/"
echo "  - Incoming: ${DATA_DIR}/incoming/"
echo "  - Certificates: ${DATA_DIR}/certs/"
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root (sudo)${NC}"
    exit 1
fi

# Check if data directory exists
if [[ ! -d "$DATA_DIR" ]]; then
    echo -e "${RED}Error: Data directory not found: ${DATA_DIR}${NC}"
    echo "Nothing to preserve. Use a fresh install instead."
    exit 1
fi

# Check if database exists
if [[ ! -f "$DATA_DIR/db/anemone.db" ]]; then
    echo -e "${RED}Error: Database not found: ${DATA_DIR}/db/anemone.db${NC}"
    echo "Nothing to restore from. Use a fresh install instead."
    exit 1
fi

# Confirmation
echo -e "${YELLOW}Are you sure you want to continue? [y/N]${NC}"
read -r response
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "Starting cleanup..."

# 1. Stop services
echo -n "Stopping services... "
systemctl stop anemone 2>/dev/null || true
systemctl stop smbd 2>/dev/null || true
echo -e "${GREEN}OK${NC}"

# 2. Disable and remove systemd service
echo -n "Removing systemd service... "
systemctl disable anemone 2>/dev/null || true
rm -f /etc/systemd/system/anemone.service
systemctl daemon-reload
echo -e "${GREEN}OK${NC}"

# 3. Remove binaries
echo -n "Removing binaries... "
rm -f /usr/local/bin/anemone
rm -f /usr/local/bin/anemone-dfree
echo -e "${GREEN}OK${NC}"

# 4. Remove Samba users (from smbpasswd)
echo -n "Removing Samba users... "
if command -v pdbedit &>/dev/null; then
    # Get list of Anemone users from database
    if command -v sqlite3 &>/dev/null; then
        users=$(sqlite3 "$DATA_DIR/db/anemone.db" "SELECT username FROM users WHERE is_admin = 0;" 2>/dev/null || true)
        for user in $users; do
            pdbedit -x "$user" 2>/dev/null || true
        done
    fi
fi
echo -e "${GREEN}OK${NC}"

# 5. Remove system users created by Anemone
echo -n "Removing system users... "
if command -v sqlite3 &>/dev/null; then
    users=$(sqlite3 "$DATA_DIR/db/anemone.db" "SELECT username FROM users WHERE is_admin = 0;" 2>/dev/null || true)
    for user in $users; do
        userdel "$user" 2>/dev/null || true
    done
fi
# Remove anemone system user
userdel anemone 2>/dev/null || true
echo -e "${GREEN}OK${NC}"

# 6. Remove Samba configuration (Anemone-generated)
echo -n "Removing Samba configuration... "
rm -f "${DATA_DIR}/smb/smb.conf"
# Remove include line from main smb.conf if present
if [[ -f /etc/samba/smb.conf ]]; then
    sed -i "\|include = ${DATA_DIR}/smb/smb.conf|d" /etc/samba/smb.conf 2>/dev/null || true
fi
echo -e "${GREEN}OK${NC}"

# 7. Remove generated files (but keep data)
echo -n "Removing generated files... "
rm -f "${DATA_DIR}/.setup_state"
rm -f "${DATA_DIR}/.needs-setup"
echo -e "${GREEN}OK${NC}"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Cleanup complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Data preserved in: ${DATA_DIR}"
echo "  - Database: ${DATA_DIR}/db/anemone.db"
echo "  - Shares: ${DATA_DIR}/shares/"
echo "  - Incoming: ${DATA_DIR}/incoming/"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Run: sudo ./install.sh"
echo "  2. Choose option 2 (Repair/Reinstall)"
echo "  3. After repair, users must reset their password via web UI for SMB access"
echo ""
