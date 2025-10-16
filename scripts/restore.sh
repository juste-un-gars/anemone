#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║  🪸  ANEMONE - Restauration           ║"
echo "╔═══════════════════════════════════════╗"
echo -e "${NC}"
echo ""

# Vérifier setup
if [ ! -f config/.setup-completed ]; then
    echo -e "${RED}❌ Setup non terminé${NC}"
    echo ""
    echo "Pour restaurer :"
    echo "1. ${YELLOW}docker-compose up -d${NC}"
    echo "2. ${YELLOW}http://localhost:3000/setup${NC}"
    echo "3. Choisir 'Restauration'"
    echo "4. Saisir votre clé Restic"
    exit 1
fi

echo -e "${GREEN}✓ Configuration OK${NC}"
echo ""

# Vérifier conteneur
if ! docker ps | grep -q anemone-restic; then
    echo -e "${RED}❌ Conteneur non démarré${NC}"
    echo "   Lancez: ${YELLOW}docker-compose up -d${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Service actif${NC}"
echo ""

# Lire targets
echo -e "${BLUE}Destinations disponibles :${NC}"
echo ""

targets=$(grep -A 10 "^  targets:" config/config.yaml | grep "name:" | sed 's/.*name: "\(.*\)"/\1/' || true)

if [ -z "$targets" ]; then
    echo -e "${RED}❌ Aucune destination${NC}"
    exit 1
fi

i=1
declare -A target_map
while IFS= read -r target; do
    echo "  [$i] $target"
    target_map[$i]=$target
    ((i++))
done <<< "$targets"

echo ""
echo -e "${YELLOW}Choisissez (numéro) :${NC}"
read -r choice

selected_target="${target_map[$choice]}"

if [ -z "$selected_target" ]; then
    echo -e "${RED}❌ Choix invalide${NC}"
    exit 1
fi

echo -e "${GREEN}→ Source : $selected_target${NC}"
echo ""

# Extraire infos
host=$(grep -A 5 "name: \"$selected_target\"" config/config.yaml | grep "host:" | awk '{print $2}' | tr -d '"')
port=$(grep -A 5 "name: \"$selected_target\"" config/config.yaml | grep "port:" | awk '{print $2}')
user=$(grep -A 5 "name: \"$selected_target\"" config/config.yaml | grep "user:" | awk '{print $2}' | tr -d '"')
path=$(grep -A 5 "name: \"$selected_target\"" config/config.yaml | grep "path:" | awk '{print $2}' | tr -d '"')

repo="sftp:$user@$host:$path"

echo -e "${BLUE}Repository : $repo${NC}"
echo -e "${BLUE}Liste des snapshots...${NC}"
echo ""

if ! docker exec anemone-restic restic -r "$repo" snapshots; then
    echo ""
    echo -e "${RED}❌ Impossible de lister${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}ID du snapshot (ou 'latest') :${NC}"
read -r snapshot_id

echo -e "${YELLOW}Dossier de restauration (./restore) :${NC}"
read -r restore_path
restore_path=${restore_path:-./restore}

mkdir -p "$restore_path"

echo ""
echo -e "${BLUE}🔄 Restauration en cours...${NC}"
echo ""

if docker exec anemone-restic restic -r "$repo" restore "$snapshot_id" --target /tmp/restore; then
    echo ""
    echo -e "${BLUE}📦 Copie des fichiers...${NC}"
    docker cp anemone-restic:/tmp/restore/. "$restore_path/"
    docker exec anemone-restic rm -rf /tmp/restore
    
    echo ""
    echo -e "${GREEN}✅ Restauration réussie !${NC}"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${CYAN}📂 Fichiers dans :${NC} ${YELLOW}$restore_path${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
else
    echo ""
    echo -e "${RED}❌ Échec${NC}"
    exit 1
fi
