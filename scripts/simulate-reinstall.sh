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

# Detect data directory
detect_data_dir() {
    # 1. Environment variable (explicit)
    if [[ -n "$ANEMONE_DATA_DIR" ]]; then
        echo "$ANEMONE_DATA_DIR"
        return
    fi

    # 2. Read from running systemd service
    if systemctl is-active --quiet anemone 2>/dev/null; then
        local dir=$(systemctl show anemone -p Environment 2>/dev/null | grep -oP 'ANEMONE_DATA_DIR=\K[^\s"]+')
        if [[ -n "$dir" ]]; then
            echo "$dir"
            return
        fi
    fi

    # 3. Read from env file
    if [[ -f /etc/anemone/anemone.env ]]; then
        local dir=$(grep -oP '^ANEMONE_DATA_DIR=\K.+' /etc/anemone/anemone.env 2>/dev/null)
        if [[ -n "$dir" ]]; then
            echo "$dir"
            return
        fi
    fi

    # 4. Read from service file
    if [[ -f /etc/systemd/system/anemone.service ]]; then
        local dir=$(grep -oP 'ANEMONE_DATA_DIR=\K[^"]+' /etc/systemd/system/anemone.service 2>/dev/null)
        if [[ -n "$dir" ]]; then
            echo "$dir"
            return
        fi
    fi

    # 5. Not found
    echo ""
}

DATA_DIR=$(detect_data_dir)

# Check if data directory was found
if [[ -z "$DATA_DIR" ]]; then
    echo -e "${RED}Error: Could not determine data directory.${NC}"
    echo ""
    echo "Specify it with:"
    echo "  sudo ANEMONE_DATA_DIR=/srv/anemone $0"
    exit 1
fi

# Verify database exists
if [[ ! -f "$DATA_DIR/db/anemone.db" ]]; then
    echo -e "${RED}Error: Database not found at: ${DATA_DIR}/db/anemone.db${NC}"
    echo ""
    echo "The service points to '$DATA_DIR' but the database is not there."
    echo "Specify the correct path with:"
    echo "  sudo ANEMONE_DATA_DIR=/path/to/data $0"
    exit 1
fi

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

# Show detected path
echo -e "Detected data directory: ${GREEN}${DATA_DIR}${NC}"
echo ""

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
