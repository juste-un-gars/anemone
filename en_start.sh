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

echo -e "${CYAN}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🪸 Anemone - New Server Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${NC}"

echo -e "${YELLOW}⚠️  Are you sure you want to create a NEW server?${NC}"
echo ""
echo "   If you want to RESTORE an existing server from backup,"
echo "   use instead: ${GREEN}./en_restore.sh backup.enc${NC}"
echo ""
read -p "Continue with a new server? (yes/no): " -r CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${RED}❌ Cancelled${NC}"
    exit 0
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Step 1/5: Prerequisites Check${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    echo "   Install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✅ Docker detected${NC}"

# Check Docker Compose and determine which command to use
DOCKER_COMPOSE_CMD=""
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
    echo -e "${GREEN}✅ Docker Compose v2 detected${NC}"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
    echo -e "${GREEN}✅ Docker Compose v1 detected${NC}"
    echo -e "${YELLOW}⚠️  Docker Compose v1 is deprecated, install the v2 plugin${NC}"
else
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Step 2/5: Initialization${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Run init.sh if config doesn't exist
if [ ! -d "config" ] || [ ! -f "config/wireguard/private.key" ]; then
    echo "🔑 Generating keys (WireGuard, SSH)..."
    ./scripts/init.sh
    echo -e "${GREEN}✅ Keys generated${NC}"
else
    echo -e "${YELLOW}⚠️  Existing configuration detected${NC}"
    read -p "   Regenerate keys? (yes/no): " -r REGEN
    if [ "$REGEN" = "yes" ]; then
        ./scripts/init.sh
        echo -e "${GREEN}✅ Keys regenerated${NC}"
    else
        echo "   Keeping existing keys"
    fi
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Step 3/5: Server Configuration${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

read -p "🏷️  Server name (e.g., SERVER1, PARIS, HOME): " SERVER_NAME
read -p "🌐 External VPN address (e.g., dyndns, public IP): " EXTERNAL_ADDR
read -p "🔌 WireGuard port (default 51820): " VPN_PORT
VPN_PORT=${VPN_PORT:-51820}

# Update config.yaml if needed
if [ -f "config/config.yaml" ]; then
    echo "📝 Updating config/config.yaml..."
    sed -i "s/name: .*/name: ${SERVER_NAME}/" config/config.yaml 2>/dev/null || true
    sed -i "s/endpoint: .*/endpoint: ${EXTERNAL_ADDR}:${VPN_PORT}/" config/config.yaml 2>/dev/null || true
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Step 3b/5: Storage Configuration${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Share and storage configuration
DOCKER_PROFILES=""
USE_NETWORK_SHARES="no"

echo ""
read -p "📂 Do you want to use integrated sharing (Samba + WebDAV)? (yes/no): " USE_INTEGRATED_SHARES

if [ "$USE_INTEGRATED_SHARES" = "yes" ]; then
    DOCKER_PROFILES="--profile shares"
    echo -e "${GREEN}✅ Integrated sharing will be enabled${NC}"
else
    echo -e "${YELLOW}ℹ️  Integrated sharing will not be enabled${NC}"
    echo ""
    read -p "🌐 Do you want to mount an existing network share? (yes/no): " USE_NETWORK_SHARES

    if [ "$USE_NETWORK_SHARES" = "yes" ]; then
        echo ""
        echo -e "${BLUE}Configuring network mount...${NC}"

        # Check if cifs-utils is installed
        if ! dpkg -l | grep -q cifs-utils 2>/dev/null && ! rpm -q cifs-utils &>/dev/null; then
            echo -e "${YELLOW}⚠️  cifs-utils is not installed${NC}"
            read -p "   Do you want to install it now? (yes/no): " INSTALL_CIFS
            if [ "$INSTALL_CIFS" = "yes" ]; then
                if command -v apt-get &> /dev/null; then
                    sudo apt-get update && sudo apt-get install -y cifs-utils
                elif command -v dnf &> /dev/null; then
                    sudo dnf install -y cifs-utils
                elif command -v yum &> /dev/null; then
                    sudo yum install -y cifs-utils
                else
                    echo -e "${RED}❌ Cannot install automatically. Please install cifs-utils manually.${NC}"
                    exit 1
                fi
            else
                echo -e "${RED}❌ cifs-utils is required to mount network shares${NC}"
                exit 1
            fi
        fi

        echo ""
        echo "Enter network share information for user data:"
        read -p "  Server/Share (e.g., //192.168.1.10/backup): " SMB_BACKUP_PATH
        echo ""
        echo "Enter network share information for received backups:"
        read -p "  Server/Share (e.g., //192.168.1.10/backups): " SMB_BACKUPS_PATH
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

        # Create mount script
        cat > mount-shares.sh << 'EOFMOUNT'
#!/bin/bash
# Anemone - Network share mount script
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0

set -e

CREDENTIALS="/root/.anemone-cifs-credentials"
MOUNT_OPTS="credentials=${CREDENTIALS},iocharset=utf8,file_mode=0777,dir_mode=0777"

# Mount backup (user data)
if ! mountpoint -q /mnt/anemone/backup; then
    echo "Mounting SMB_BACKUP_PATH_PLACEHOLDER..."
    sudo mount -t cifs "SMB_BACKUP_PATH_PLACEHOLDER" /mnt/anemone/backup -o ${MOUNT_OPTS}
    echo "✅ Mounted: /mnt/anemone/backup"
fi

# Mount backups (received from peers)
if ! mountpoint -q /mnt/anemone/backups; then
    echo "Mounting SMB_BACKUPS_PATH_PLACEHOLDER..."
    sudo mount -t cifs "SMB_BACKUPS_PATH_PLACEHOLDER" /mnt/anemone/backups -o ${MOUNT_OPTS}
    echo "✅ Mounted: /mnt/anemone/backups"
fi

echo "✅ All shares mounted"
EOFMOUNT

        # Replace placeholders
        sed -i "s|SMB_BACKUP_PATH_PLACEHOLDER|${SMB_BACKUP_PATH}|g" mount-shares.sh
        sed -i "s|SMB_BACKUPS_PATH_PLACEHOLDER|${SMB_BACKUPS_PATH}|g" mount-shares.sh
        chmod +x mount-shares.sh

        # Mount now
        echo ""
        echo "📌 Mounting network shares..."
        sudo ./mount-shares.sh

        # Create/modify .env to use mounts
        cat > .env << EOFENV
# Configuration generated by en_start.sh
BACKUP_DATA_PATH=/mnt/anemone/backup
BACKUP_RECEIVE_PATH=/mnt/anemone/backups
EOFENV

        echo -e "${GREEN}✅ Network shares mounted and configured${NC}"
        echo -e "${YELLOW}⚠️  To remount automatically at boot, add to /etc/fstab:${NC}"
        echo ""
        echo "${SMB_BACKUP_PATH} /mnt/anemone/backup cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0"
        echo "${SMB_BACKUPS_PATH} /mnt/anemone/backups cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0"
        echo ""
    fi
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Step 4/5: Starting Docker${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "🐳 Building and starting containers..."
$DOCKER_COMPOSE_CMD up -d --build $DOCKER_PROFILES

echo ""
echo -e "${GREEN}✅ Containers started successfully!${NC}"

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Step 5/5: Initial Setup${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✅ Installation completed!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📋 NEXT STEPS:${NC}"
echo ""
echo "1. 🌐 Go to: ${CYAN}http://localhost:3000/setup${NC}"
echo ""
echo "2. 🔐 Configure your Restic encryption key"
echo "   • Choose 'New server' to generate a new key"
echo "   • ${RED}⚠️  SAVE THE KEY IN BITWARDEN IMMEDIATELY!${NC}"
echo ""
echo "3. 👥 Add peers for redundancy"
echo "   • Web interface: http://localhost:3000/peers"
echo "   • Or use: ./scripts/add-peer.sh"
echo ""
echo "4. 📊 Monitor backups"
echo "   • Dashboard: http://localhost:3000/"
echo "   • Recovery: http://localhost:3000/recovery"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Logs: $DOCKER_COMPOSE_CMD logs -f${NC}"
echo -e "${CYAN}  Stop: $DOCKER_COMPOSE_CMD down${NC}"
echo -e "${CYAN}  Restart: $DOCKER_COMPOSE_CMD restart${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
