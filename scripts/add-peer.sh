#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🪸 Anemone - Ajouter un pair${NC}"
echo ""

echo -e "${BLUE}Nom du pair :${NC}"
read -r PEER_NAME

echo -e "${BLUE}Clé publique WireGuard :${NC}"
read -r PEER_PUBKEY

echo -e "${BLUE}Endpoint (DNS:port) :${NC}"
read -r PEER_ENDPOINT

echo -e "${BLUE}IP VPN (ex: 10.8.0.2) :${NC}"
read -r PEER_IP

echo -e "${BLUE}Clé publique SSH :${NC}"
read -r PEER_SSH_KEY

echo ""
echo -e "${YELLOW}Continuer ? (o/N)${NC}"
read -r CONFIRM

if [[ ! "$CONFIRM" =~ ^[oO]$ ]]; then
    exit 0
fi

# Ajouter au config.yaml
cat >> config/config.yaml <<EOF

  - name: "$PEER_NAME"
    endpoint: "$PEER_ENDPOINT"
    public_key: "$PEER_PUBKEY"
    allowed_ips: "$PEER_IP/32"
    persistent_keepalive: 25
EOF

# Ajouter la clé SSH
if [ -n "$PEER_SSH_KEY" ]; then
    echo "$PEER_SSH_KEY" >> config/ssh/authorized_keys
fi

echo -e "${GREEN}✅ Pair ajouté !${NC}"
echo "Relancez: docker-compose restart"
