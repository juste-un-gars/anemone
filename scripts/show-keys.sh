#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║     🔑  ANEMONE - Clés publiques      ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"
echo ""

# Vérifier que les clés existent
if [ ! -f config/wireguard/public.key ]; then
    echo -e "${YELLOW}⚠ Clés WireGuard non générées. Lancez d'abord: ./scripts/init.sh${NC}"
    exit 1
fi

if [ ! -f config/ssh/id_rsa.pub ]; then
    echo -e "${YELLOW}⚠ Clés SSH non générées. Lancez d'abord: ./scripts/init.sh${NC}"
    exit 1
fi

echo -e "${YELLOW}🔑 Clé publique WireGuard :${NC}"

# Vérifier si c'est un placeholder
if grep -q "# Clé publique sera générée" config/wireguard/public.key 2>/dev/null; then
    echo -e "${RED}⚠ La clé publique n'a pas encore été extraite${NC}"
    echo ""
    echo -e "${YELLOW}→ Extraction automatique depuis le conteneur...${NC}"

    if docker ps | grep -q anemone-wireguard; then
        PUBKEY=$(docker exec anemone-wireguard sh -c "cat /config/wireguard/private.key | wg pubkey" 2>/dev/null || echo "")
        if [ -n "$PUBKEY" ]; then
            echo "$PUBKEY" > config/wireguard/public.key
            echo -e "${GREEN}✓ Clé extraite : ${PUBKEY}${NC}"
        else
            echo -e "${RED}❌ Impossible d'extraire. Le conteneur WireGuard doit être démarré.${NC}"
            exit 1
        fi
    else
        echo -e "${RED}❌ Le conteneur WireGuard n'est pas démarré${NC}"
        echo "   Lancez: docker-compose up -d wireguard"
        echo "   Puis: ./scripts/extract-wireguard-pubkey.sh"
        exit 1
    fi
else
    cat config/wireguard/public.key
fi

echo ""

echo -e "${YELLOW}🔑 Clé publique SSH :${NC}"
cat config/ssh/id_rsa.pub
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}💡 Ces clés sont à partager avec vos pairs${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
