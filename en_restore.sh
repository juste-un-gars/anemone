#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

BACKUP_FILE="$1"

# Check Docker Compose and determine which command to use
DOCKER_COMPOSE_CMD=""
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
else
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    echo "   Install Docker Compose v2: https://docs.docker.com/compose/install/"
    exit 1
fi

echo -e "${CYAN}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🪸 Anemone - Restore from Backup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${NC}"

# Check backup file
if [ -z "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ Error: Backup file not specified${NC}"
    echo ""
    echo "Usage: $0 <backup_file.enc>"
    echo ""
    echo "Example: $0 anemone-backup-SERVER1-20250122-143000.enc"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ File not found: $BACKUP_FILE${NC}"
    exit 1
fi

# Display file info
FILE_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
FILE_DATE=$(stat -c %y "$BACKUP_FILE" 2>/dev/null | cut -d' ' -f1 || date -r "$BACKUP_FILE" "+%Y-%m-%d" 2>/dev/null || echo "unknown")

echo ""
echo -e "${BLUE}📄 Backup file: ${CYAN}$BACKUP_FILE${NC}"
echo -e "${BLUE}   Size: ${CYAN}$FILE_SIZE${NC}"
echo -e "${BLUE}   Date: ${CYAN}$FILE_DATE${NC}"
echo ""

# Ask for Restic key
echo -e "${YELLOW}🔑 Restic Decryption Key${NC}"
echo "   Enter the key you saved in Bitwarden"
echo ""
read -s -p "Restic key: " RESTIC_KEY
echo ""

if [ -z "$RESTIC_KEY" ]; then
    echo -e "${RED}❌ Key cannot be empty${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Backup Verification${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "🔍 Decrypting and verifying..."

# Use Python to decrypt and verify
python3 << EOF
import sys
import tarfile
import io
from pathlib import Path
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2
from cryptography.hazmat.backends import default_backend

try:
    # Read encrypted file
    with open('$BACKUP_FILE', 'rb') as f:
        encrypted_data = f.read()

    # Extract IV and data
    iv = encrypted_data[:16]
    ciphertext = encrypted_data[16:]

    # Derive encryption key from Restic key
    kdf = PBKDF2(
        algorithm=hashes.SHA256(),
        length=32,
        salt=b'anemone-config-backup',
        iterations=100000,
        backend=default_backend()
    )
    encryption_key = kdf.derive('$RESTIC_KEY'.encode())

    # Decrypt
    cipher = Cipher(
        algorithms.AES(encryption_key),
        modes.CBC(iv),
        backend=default_backend()
    )
    decryptor = cipher.decryptor()
    padded_data = decryptor.update(ciphertext) + decryptor.finalize()

    # Remove PKCS7 padding
    padding_length = padded_data[-1]
    data = padded_data[:-padding_length]

    # Verify it's a valid tar
    tar_file = tarfile.open(fileobj=io.BytesIO(data))
    members = tar_file.getnames()

    print(f'✅ Valid backup ({len(members)} files)')

    # Extract to temporary directory
    tar_file.extractall('$TEMP_DIR')
    tar_file.close()

    sys.exit(0)

except Exception as e:
    print(f'❌ Error: {e}', file=sys.stderr)
    sys.exit(1)
EOF

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Decryption failed${NC}"
    echo "   Check that the key is correct"
    exit 1
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Backup Content${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Analyze content
if [ -f "$TEMP_DIR/config.yaml" ]; then
    SERVER_NAME=$(grep "^  name:" "$TEMP_DIR/config.yaml" | awk '{print $2}' || echo "unknown")
    ENDPOINT=$(grep "^  endpoint:" "$TEMP_DIR/config.yaml" | awk '{print $2}' || echo "unknown")
    PEER_COUNT=$(grep -c "^  - name:" "$TEMP_DIR/config.yaml" || echo "0")

    echo "  📝 Server: $SERVER_NAME"
    echo "  🌐 Endpoint: $ENDPOINT"
    echo "  👥 Configured peers: $PEER_COUNT"
else
    echo "  ${YELLOW}⚠️  config.yaml not found in backup${NC}"
fi

echo ""
echo -e "${YELLOW}⚠️  This operation will:${NC}"
echo "   • Overwrite current configuration"
echo "   • Restore WireGuard and SSH keys"
echo "   • Restore peer configuration"
echo ""
read -p "Continue? (yes/no): " -r CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${RED}❌ Cancelled${NC}"
    exit 0
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Restoration in Progress${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Create config directories if they don't exist
mkdir -p config/wireguard config/ssh

# Restore files
echo "📦 Copying configuration files..."
cp -r "$TEMP_DIR"/* config/ 2>/dev/null || true

# Save Restic key to temporary file for setup
echo "$RESTIC_KEY" > /tmp/.restic-key-restore
chmod 600 /tmp/.restic-key-restore

echo -e "${GREEN}✅ Configuration restored${NC}"

# Check if storage configuration exists
if [ -f "config/.anemone-storage-config" ]; then
    STORAGE_TYPE=$(grep "storage_type:" config/.anemone-storage-config | cut -d: -f2 | tr -d ' ')

    if [ "$STORAGE_TYPE" = "network_mount" ]; then
        echo ""
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}  🌐 Network mount configuration detected${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""

        # Get share paths
        NETWORK_BACKUP=$(grep "network_backup_path:" config/.anemone-storage-config | cut -d: -f2- | tr -d ' ')
        NETWORK_BACKUPS=$(grep "network_backups_path:" config/.anemone-storage-config | cut -d: -f2- | tr -d ' ')

        echo "Old server used:"
        echo "  • Backup  : ${CYAN}$NETWORK_BACKUP${NC}"
        echo "  • Backups : ${CYAN}$NETWORK_BACKUPS${NC}"
        echo ""
        read -p "Do you want to remount these network shares? (yes/no): " REMOUNT_SHARES

        if [ "$REMOUNT_SHARES" = "yes" ]; then
            echo ""
            read -p "👤 Username for mounts: " SMB_USERNAME
            read -s -p "🔐 Password: " SMB_PASSWORD
            echo ""

            # Create mount directories
            sudo mkdir -p /mnt/anemone/backup /mnt/anemone/backups

            # Create credentials file
            sudo bash -c "cat > /root/.anemone-cifs-credentials << EOF
username=${SMB_USERNAME}
password=${SMB_PASSWORD}
EOF"
            sudo chmod 600 /root/.anemone-cifs-credentials

            # Mount shares
            echo "📌 Mounting network shares..."
            sudo mount -t cifs "$NETWORK_BACKUP" /mnt/anemone/backup -o credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 || {
                echo -e "${RED}❌ Failed to mount $NETWORK_BACKUP${NC}"
            }

            sudo mount -t cifs "$NETWORK_BACKUPS" /mnt/anemone/backups -o credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 || {
                echo -e "${RED}❌ Failed to mount $NETWORK_BACKUPS${NC}"
            }

            # Add to fstab
            sudo cp /etc/fstab /etc/fstab.backup.$(date +%Y%m%d-%H%M%S)

            if ! grep -qF "$NETWORK_BACKUP" /etc/fstab; then
                echo "$NETWORK_BACKUP /mnt/anemone/backup cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0" | sudo tee -a /etc/fstab > /dev/null
            fi

            if ! grep -qF "$NETWORK_BACKUPS" /etc/fstab; then
                echo "$NETWORK_BACKUPS /mnt/anemone/backups cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0" | sudo tee -a /etc/fstab > /dev/null
            fi

            # Create .env
            cat > .env << EOFENV
# Configuration generated by en_restore.sh
BACKUP_DATA_PATH=/mnt/anemone/backup
BACKUP_RECEIVE_PATH=/mnt/anemone/backups
EOFENV

            echo -e "${GREEN}✅ Network shares remounted${NC}"
            echo ""
            read -p "Do you want to restore data from a peer? (yes/no): " RESTORE_FROM_PEER
        else
            echo ""
            echo -e "${YELLOW}ℹ️  Switching to local storage mode${NC}"
            echo "   You will probably need to restore your data from a peer"
            echo ""
            RESTORE_FROM_PEER="yes"
        fi
    fi
fi

# Start Docker
echo ""
echo "🐳 Starting Docker..."
$DOCKER_COMPOSE_CMD up -d --build

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✅ Restoration Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📋 NEXT STEPS:${NC}"
echo ""
echo "1. 🔐 Finalize Restic configuration"
echo "   • Go to: ${CYAN}http://localhost:3000/setup${NC}"
echo "   • Choose 'Restore'"
echo "   • Paste your Restic key (same as used for restoration)"
echo ""
echo "2. 🔄 Restore your data from a peer"
echo "   • ${CYAN}http://localhost:3000/recovery${NC} → Restore from peer"
echo "   • Choose source peer (${SERVER_NAME} is now reconnected)"
echo "   • Simulation mode then restoration"
echo ""
echo "3. ✅ Verify everything works"
echo "   • Dashboard: ${CYAN}http://localhost:3000/${NC}"
echo "   • Logs: ${CYAN}$DOCKER_COMPOSE_CMD logs -f${NC}"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}⚠️  Reminder: Your temporary key is in /tmp/.restic-key-restore${NC}"
echo -e "${RED}   Delete it after completing setup!${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
