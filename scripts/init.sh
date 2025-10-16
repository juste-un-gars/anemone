#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║     🪸  ANEMONE - Initialisation      ║"
echo "╔═══════════════════════════════════════╗"
echo -e "${NC}"
echo ""

echo -e "${BLUE}[1/7]${NC} Vérification des dépendances..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker n'est pas installé${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker installé${NC}"
echo ""

echo -e "${BLUE}[2/7]${NC} Création de la structure..."
mkdir -p config/{wireguard,ssh,samba}
mkdir -p data backups logs services/{restic,api}
echo -e "${GREEN}✓ Structure créée${NC}"
echo ""

echo -e "${BLUE}[3/7]${NC} Génération des clés WireGuard..."
if [ ! -f config/wireguard/private.key ]; then
    docker run --rm -v "$(pwd)/config/wireguard:/config" \
        linuxserver/wireguard:latest \
        sh -c "wg genkey | tee /config/private.key | wg pubkey > /config/public.key"
    chmod 600 config/wireguard/private.key
    chmod 644 config/wireguard/public.key
    echo -e "${GREEN}✓ Clés WireGuard générées${NC}"
else
    echo -e "${YELLOW}⚠ Clés WireGuard déjà présentes${NC}"
fi
echo ""

echo -e "${BLUE}[4/7]${NC} Génération du mot de passe Restic..."
if [ ! -f config/restic-password ]; then
    openssl rand -base64 32 > config/restic-password
    chmod 600 config/restic-password
    echo -e "${GREEN}✓ Mot de passe Restic généré${NC}"
else
    echo -e "${YELLOW}⚠ Mot de passe Restic déjà présent${NC}"
fi
echo ""

echo -e "${BLUE}[5/7]${NC} Génération des clés SSH..."
if [ ! -f config/ssh/id_rsa ]; then
    ssh-keygen -t rsa -b 4096 -f config/ssh/id_rsa -N "" -C "restic@anemone" -q
    chmod 600 config/ssh/id_rsa
    chmod 644 config/ssh/id_rsa.pub
    touch config/ssh/authorized_keys
    chmod 600 config/ssh/authorized_keys
    echo -e "${GREEN}✓ Clés SSH générées${NC}"
else
    echo -e "${YELLOW}⚠ Clés SSH déjà présentes${NC}"
fi
echo ""

echo -e "${BLUE}[6/17]${NC} Configuration..."
if [ ! -f config/config.yaml ]; then
    cp config/config.yaml.example config/config.yaml
    echo -e "${GREEN}✓ Fichier config.yaml créé${NC}"
else
    echo -e "${YELLOW}⚠ config.yaml déjà présent${NC}"
fi

if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${GREEN}✓ Fichier .env créé${NC}"
else
    echo -e "${YELLOW}⚠ .env déjà présent${NC}"
fi
echo ""

echo -e "${BLUE}[7/7]${NC} Récapitulatif..."
echo ""
echo -e "${GREEN}✅ Initialisation terminée !${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${CYAN}📋 INFORMATIONS À PARTAGER AVEC VOS PAIRS${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${YELLOW}🔑 Clé publique WireGuard :${NC}"
cat config/wireguard/public.key
echo ""
echo -e "${YELLOW}🔑 Clé publique SSH :${NC}"
cat config/ssh/id_rsa.pub
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${CYAN}📝 PROCHAINES ÉTAPES${NC}"
echo ""
echo "1. Éditez .env (mots de passe)"
echo "2. Éditez config/config.yaml (configuration)"
echo "3. Lancez: docker-compose up -d"
echo ""
