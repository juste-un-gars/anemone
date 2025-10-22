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

# Start Docker
echo ""
echo "🐳 Starting Docker..."
docker-compose up -d --build

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
echo "   • Logs: ${CYAN}docker-compose logs -f${NC}"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}⚠️  Reminder: Your temporary key is in /tmp/.restic-key-restore${NC}"
echo -e "${RED}   Delete it after completing setup!${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
