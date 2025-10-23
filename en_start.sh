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
echo -e "${CYAN}  Step 4/5: Starting Docker${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "🐳 Building and starting containers..."
$DOCKER_COMPOSE_CMD up -d --build

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
