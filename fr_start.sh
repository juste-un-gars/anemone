#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🪸 Anemone - Configuration d'un nouveau serveur"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${NC}"

echo -e "${YELLOW}⚠️  Êtes-vous sûr de vouloir créer un NOUVEAU serveur ?${NC}"
echo ""
echo "   Si vous voulez RESTAURER un serveur existant depuis un backup,"
echo "   utilisez plutôt : ${GREEN}./fr_restore.sh backup.enc${NC}"
echo ""
read -p "Continuer avec un nouveau serveur ? (oui/non) : " -r CONFIRM

if [ "$CONFIRM" != "oui" ]; then
    echo -e "${RED}❌ Annulé${NC}"
    exit 0
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 1/5 : Vérification des prérequis${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Vérifier Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker n'est pas installé${NC}"
    echo "   Installez Docker : https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✅ Docker détecté${NC}"

# Vérifier Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose n'est pas installé${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker Compose détecté${NC}"

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 2/5 : Initialisation${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Lancer init.sh si config n'existe pas
if [ ! -d "config" ] || [ ! -f "config/wireguard/private.key" ]; then
    echo "🔑 Génération des clés (WireGuard, SSH)..."
    ./scripts/init.sh
    echo -e "${GREEN}✅ Clés générées${NC}"
else
    echo -e "${YELLOW}⚠️  Configuration existante détectée${NC}"
    read -p "   Régénérer les clés ? (oui/non) : " -r REGEN
    if [ "$REGEN" = "oui" ]; then
        ./scripts/init.sh
        echo -e "${GREEN}✅ Clés régénérées${NC}"
    else
        echo "   Clés existantes conservées"
    fi
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 3/5 : Configuration du serveur${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

read -p "🏷️  Nom de ce serveur (ex: FR1, PARIS, HOME) : " SERVER_NAME
read -p "🌐 Adresse DynDNS (ex: mon-serveur.duckdns.org) : " DYNDNS

# Mettre à jour config.yaml si nécessaire
if [ -f "config/config.yaml" ]; then
    echo "📝 Mise à jour de config/config.yaml..."
    sed -i "s/name: .*/name: ${SERVER_NAME}/" config/config.yaml 2>/dev/null || true
    sed -i "s/endpoint: .*/endpoint: ${DYNDNS}:51820/" config/config.yaml 2>/dev/null || true
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 4/5 : Démarrage de Docker${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "🐳 Construction et démarrage des conteneurs..."
docker-compose up -d --build

echo ""
echo -e "${GREEN}✅ Conteneurs démarrés avec succès !${NC}"

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 5/5 : Configuration initiale${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✅ Installation terminée !${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📋 PROCHAINES ÉTAPES :${NC}"
echo ""
echo "1. 🌐 Accédez à : ${CYAN}http://localhost:3000/setup${NC}"
echo ""
echo "2. 🔐 Configurez votre clé de chiffrement Restic"
echo "   • Choisissez 'Nouveau serveur' pour générer une nouvelle clé"
echo "   • ${RED}⚠️  SAUVEGARDEZ LA CLÉ DANS BITWARDEN IMMÉDIATEMENT !${NC}"
echo ""
echo "3. 👥 Ajoutez des pairs pour la redondance"
echo "   • Via l'interface web : http://localhost:3000/peers"
echo "   • Ou utilisez : ./scripts/add-peer.sh"
echo ""
echo "4. 📊 Surveillez les backups"
echo "   • Dashboard : http://localhost:3000/"
echo "   • Recovery : http://localhost:3000/recovery"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Logs : docker-compose logs -f${NC}"
echo -e "${CYAN}  Arrêter : docker-compose down${NC}"
echo -e "${CYAN}  Redémarrer : docker-compose restart${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
