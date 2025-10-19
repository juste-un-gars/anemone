#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║  🔧  Régénération de wg0.conf         ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"
echo ""

# Vérifier que les fichiers nécessaires existent
if [ ! -f config/config.yaml ]; then
    echo -e "${RED}✗ config/config.yaml manquant${NC}"
    exit 1
fi

if [ ! -f config/wireguard/private.key ]; then
    echo -e "${RED}✗ config/wireguard/private.key manquant${NC}"
    echo -e "${YELLOW}Lancez d'abord : ./scripts/init.sh${NC}"
    exit 1
fi

# Créer le dossier wg_confs si nécessaire
mkdir -p config/wg_confs

# Backup de l'ancien wg0.conf si existant
if [ -f config/wg_confs/wg0.conf ]; then
    BACKUP_FILE="config/wg_confs/wg0.conf.backup.$(date +%Y%m%d_%H%M%S)"
    cp config/wg_confs/wg0.conf "$BACKUP_FILE"
    echo -e "${YELLOW}📦 Backup créé : $BACKUP_FILE${NC}"
    echo ""
fi

# Régénérer wg0.conf depuis config.yaml
echo -e "${BLUE}Génération de wg0.conf depuis config.yaml...${NC}"

if python3 scripts/generate-wireguard-config.py config/config.yaml config/wg_confs/wg0.conf; then
    echo ""
    echo -e "${GREEN}✅ wg0.conf régénéré avec succès !${NC}"
    echo ""
    echo -e "${CYAN}📋 Contenu généré :${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    cat config/wg_confs/wg0.conf
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo -e "${YELLOW}⚠  IMPORTANT : Redémarrez WireGuard pour appliquer :${NC}"
    echo -e "   ${CYAN}docker-compose restart wireguard${NC}"
    echo ""
else
    echo -e "${RED}✗ Erreur lors de la génération${NC}"
    exit 1
fi
