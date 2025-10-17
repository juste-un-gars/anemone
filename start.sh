#!/bin/bash
set -e

CYAN='\033[0;36m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}🪸 Démarrage d'Anemone...${NC}"
echo ""

# Vérifier si l'initialisation a été faite
NEED_INIT=false

if [ ! -f config/wireguard/private.key ]; then
    echo -e "${YELLOW}⚠  Clés WireGuard manquantes${NC}"
    NEED_INIT=true
fi

if [ ! -f config/ssh/id_rsa ]; then
    echo -e "${YELLOW}⚠  Clés SSH manquantes${NC}"
    NEED_INIT=true
fi

if [ ! -f config/config.yaml ]; then
    echo -e "${YELLOW}⚠  config.yaml manquant${NC}"
    NEED_INIT=true
fi

if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠  .env manquant${NC}"
    NEED_INIT=true
fi

if [ "$NEED_INIT" = true ]; then
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}   L'initialisation n'a pas été effectuée !${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${CYAN}Exécution de l'initialisation automatique...${NC}"
    echo ""

    ./scripts/init.sh

    echo ""
    echo -e "${GREEN}✅ Initialisation terminée !${NC}"
    echo ""
    echo -e "${YELLOW}⚠  N'oubliez pas d'éditer les fichiers suivants :${NC}"
    echo "   - .env (mots de passe SMB/WebDAV)"
    echo "   - config/config.yaml (configuration générale)"
    echo ""
    echo -e "${CYAN}Appuyez sur Entrée pour continuer le démarrage...${NC}"
    read -r
fi

echo ""
echo -e "${CYAN}🚀 Démarrage des conteneurs Docker...${NC}"
docker compose up -d

echo ""
echo -e "${GREEN}✅ Anemone démarré !${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${CYAN}📋 PROCHAINES ÉTAPES${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ ! -f config/.setup-completed ]; then
    echo -e "${YELLOW}⚠  Setup web non complété${NC}"
    echo ""
    echo "1. Accédez à : ${GREEN}http://localhost:3000/setup${NC}"
    echo "2. Suivez l'assistant de configuration"
    echo "3. ${RED}SAUVEGARDEZ VOTRE CLÉ DANS BITWARDEN !${NC}"
else
    echo -e "${GREEN}✓${NC} Setup complété"
    echo ""
    echo "Dashboard : ${GREEN}http://localhost:3000/${NC}"
    echo ""
    echo "Commandes utiles :"
    echo "  - Logs : ${CYAN}docker compose logs -f${NC}"
    echo "  - Status : ${CYAN}docker compose ps${NC}"
    echo "  - Arrêter : ${CYAN}docker compose down${NC}"
fi

echo ""
