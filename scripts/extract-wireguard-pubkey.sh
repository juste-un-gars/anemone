#!/bin/bash
# Script pour extraire la clé publique WireGuard depuis le conteneur en cours d'exécution
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}Extraction de la clé publique WireGuard...${NC}"

# Vérifier que le conteneur tourne
if ! docker ps | grep -q anemone-wireguard; then
    echo -e "${RED}❌ Le conteneur anemone-wireguard n'est pas démarré${NC}"
    echo "   Lancez d'abord: docker-compose up -d wireguard"
    exit 1
fi

# Vérifier si la clé privée existe
if [ ! -f config/wireguard/private.key ]; then
    echo -e "${RED}❌ Clé privée introuvable dans config/wireguard/private.key${NC}"
    exit 1
fi

# Extraire la clé publique depuis le conteneur
echo -e "${YELLOW}→ Extraction via le conteneur WireGuard...${NC}"
PUBKEY=$(docker exec anemone-wireguard sh -c "cat /config/wireguard/private.key | wg pubkey" 2>/dev/null)

if [ -z "$PUBKEY" ]; then
    echo -e "${RED}❌ Impossible d'extraire la clé publique${NC}"
    exit 1
fi

# Sauvegarder la clé publique
echo "$PUBKEY" > config/wireguard/public.key
chmod 644 config/wireguard/public.key

echo -e "${GREEN}✅ Clé publique extraite et sauvegardée${NC}"
echo ""
echo -e "${YELLOW}Clé publique WireGuard :${NC}"
echo "$PUBKEY"
echo ""
