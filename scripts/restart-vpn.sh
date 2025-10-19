#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║   🔄  ANEMONE - Redémarrage VPN       ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"
echo ""

echo -e "${YELLOW}Ce script redémarre WireGuard puis Restic${NC}"
echo -e "${YELLOW}pour reconnecter le VPN après une modification.${NC}"
echo ""
echo -e "${YELLOW}Pourquoi ?${NC} Restic partage le namespace réseau de WireGuard."
echo -e "Après un restart de WireGuard, Restic doit redémarrer pour"
echo -e "se reconnecter au nouveau namespace."
echo ""

read -p "Continuer ? (o/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[OoYy]$ ]]; then
    echo -e "${RED}❌ Annulé${NC}"
    exit 0
fi

echo ""
echo -e "${CYAN}[1/3]${NC} Redémarrage de WireGuard..."
docker compose restart wireguard

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Erreur lors du redémarrage de WireGuard${NC}"
    exit 1
fi

echo -e "${GREEN}✓ WireGuard redémarré${NC}"
echo ""

echo -e "${CYAN}[2/3]${NC} Attente de 5 secondes que WireGuard soit prêt..."
sleep 5

echo -e "${CYAN}[3/3]${NC} Redémarrage de Restic..."
docker compose restart restic

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Erreur lors du redémarrage de Restic${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Restic redémarré${NC}"
echo ""

echo -e "${GREEN}✅ VPN redémarré avec succès !${NC}"
echo ""
echo -e "${CYAN}📋 Vérification recommandée :${NC}"
echo -e "  docker exec anemone-wireguard wg show"
echo -e "  docker exec anemone-restic ping -c 3 10.8.0.X"
echo ""
