#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║     🪸  ANEMONE - Ajouter un pair    ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"
echo ""

echo -e "${BLUE}📝 Informations du pair à ajouter${NC}"
echo ""

echo -e "${BLUE}Nom du pair (ex: alice) :${NC}"
read -r PEER_NAME

echo -e "${BLUE}Clé publique WireGuard :${NC}"
read -r PEER_PUBKEY

echo -e "${BLUE}Endpoint public (ex: alice.duckdns.org:51820) :${NC}"
read -r PEER_ENDPOINT

echo -e "${BLUE}IP VPN du pair (ex: 10.8.0.2) :${NC}"
read -r PEER_IP

echo -e "${BLUE}Clé publique SSH (pour autoriser les backups chez vous) :${NC}"
read -r PEER_SSH_KEY

echo ""
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo -e "${YELLOW}Récapitulatif :${NC}"
echo -e "Nom      : ${GREEN}${PEER_NAME}${NC}"
echo -e "Endpoint : ${GREEN}${PEER_ENDPOINT}${NC}"
echo -e "IP VPN   : ${GREEN}${PEER_IP}${NC}"
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Confirmer l'ajout ? (o/N)${NC}"
read -r CONFIRM

if [[ ! "$CONFIRM" =~ ^[oO]$ ]]; then
    echo -e "${RED}❌ Annulé${NC}"
    exit 0
fi

# Vérifier que config.yaml existe
if [ ! -f config/config.yaml ]; then
    echo -e "${RED}❌ config/config.yaml introuvable${NC}"
    exit 1
fi

# Ajouter au config.yaml (section peers)
echo ""
echo -e "${BLUE}[1/4]${NC} Ajout dans config.yaml..."
cat >> config/config.yaml <<EOF

  - name: "${PEER_NAME}"
    endpoint: "${PEER_ENDPOINT}"
    public_key: "${PEER_PUBKEY}"
    allowed_ips: "${PEER_IP}/32"
    persistent_keepalive: 25
    description: "Serveur de ${PEER_NAME}"
EOF
echo -e "${GREEN}✓ Pair ajouté dans config.yaml${NC}"

# Ajouter la clé SSH aux authorized_keys
if [ -n "$PEER_SSH_KEY" ]; then
    echo -e "${BLUE}[2/4]${NC} Ajout de la clé SSH..."
    mkdir -p config/ssh
    touch config/ssh/authorized_keys

    # Vérifier si la clé existe déjà
    if grep -qF "$PEER_SSH_KEY" config/ssh/authorized_keys 2>/dev/null; then
        echo -e "${YELLOW}⚠ Clé SSH déjà présente${NC}"
    else
        echo "$PEER_SSH_KEY" >> config/ssh/authorized_keys
        echo -e "${GREEN}✓ Clé SSH ajoutée${NC}"
    fi
else
    echo -e "${YELLOW}[2/4] Pas de clé SSH fournie (backup vers ce pair désactivé)${NC}"
fi

# Créer le dossier de backup pour ce pair
echo -e "${BLUE}[3/4]${NC} Création du dossier de backup..."
mkdir -p backups/${PEER_NAME}
echo -e "${GREEN}✓ Dossier backups/${PEER_NAME} créé${NC}"

# Afficher les prochaines étapes
echo -e "${BLUE}[4/4]${NC} Finalisation..."
echo ""
echo -e "${GREEN}✅ Pair ${PEER_NAME} ajouté avec succès !${NC}"
echo ""
echo -e "${CYAN}📋 Prochaines étapes :${NC}"
echo ""
echo "1. ${YELLOW}Ajouter une target de backup dans config/config.yaml${NC} :"
echo ""
echo "   backup:"
echo "     targets:"
echo "       - name: \"${PEER_NAME}-backup\""
echo "         enabled: true"
echo "         type: \"sftp\""
echo "         host: \"${PEER_IP}\""
echo "         port: 2222"
echo "         user: \"restic\""
echo "         path: \"/backups/$(whoami)\""
echo ""
echo "2. ${YELLOW}Redémarrer les services${NC} :"
echo "   docker compose down && docker compose up -d"
echo ""
echo "3. ${YELLOW}Tester la connexion VPN${NC} :"
echo "   docker exec anemone-wireguard wg show"
echo "   docker exec anemone-restic ping ${PEER_IP}"
echo ""
echo "4. ${YELLOW}Tester le backup${NC} :"
echo "   docker exec anemone-restic /scripts/backup-now.sh"
echo ""
