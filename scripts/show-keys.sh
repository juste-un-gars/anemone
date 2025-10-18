#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
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
cat config/wireguard/public.key
echo ""

echo -e "${YELLOW}🔑 Clé publique SSH :${NC}"
cat config/ssh/id_rsa.pub
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}💡 Ces clés sont à partager avec vos pairs${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
