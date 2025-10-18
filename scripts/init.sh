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
echo "║     🪸  ANEMONE - Initialisation      ║"
echo "╔═══════════════════════════════════════╗"
echo -e "${NC}"
echo ""

echo -e "${BLUE}[1/5]${NC} Vérification des dépendances..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker non installé${NC}"
    exit 1
fi
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose non installé${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker installé${NC}"
echo ""

echo -e "${BLUE}[2/5]${NC} Création de la structure..."
mkdir -p config/{wireguard,ssh,samba}
mkdir -p data backup backups logs services/{restic,api}
echo -e "${GREEN}✓ Structure créée${NC}"
echo ""

echo -e "${BLUE}[3/5]${NC} Génération clés WireGuard..."
if [ ! -f config/wireguard/private.key ]; then
    # Méthode 1: Utiliser wg si disponible sur le host (méthode officielle)
    if command -v wg &> /dev/null; then
        wg genkey | tee config/wireguard/private.key | wg pubkey > config/wireguard/public.key
        chmod 600 config/wireguard/private.key
        chmod 644 config/wireguard/public.key
        echo -e "${GREEN}✓ Clés WireGuard générées (via wg)${NC}"
    # Méthode 2: Utiliser l'image Docker WireGuard (toujours correct)
    else
        echo -e "${YELLOW}⚠ 'wg' non disponible sur le host, utilisation de Docker...${NC}"
        # Générer la clé privée
        docker run --rm linuxserver/wireguard:latest wg genkey > config/wireguard/private.key
        # Générer la clé publique correspondante
        cat config/wireguard/private.key | docker run --rm -i linuxserver/wireguard:latest wg pubkey > config/wireguard/public.key
        chmod 600 config/wireguard/private.key
        chmod 644 config/wireguard/public.key
        echo -e "${GREEN}✓ Clés WireGuard générées (via Docker)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Clés déjà présentes${NC}"
fi
echo ""

echo -e "${BLUE}[4/5]${NC} Génération clés SSH..."
if [ ! -f config/ssh/id_rsa ]; then
    ssh-keygen -t rsa -b 4096 -f config/ssh/id_rsa -N "" -C "restic@anemone" -q
    chmod 600 config/ssh/id_rsa
    chmod 644 config/ssh/id_rsa.pub
    touch config/ssh/authorized_keys
    chmod 600 config/ssh/authorized_keys
    echo -e "${GREEN}✓ Clés SSH générées${NC}"
else
    echo -e "${YELLOW}⚠ Clés déjà présentes${NC}"
fi
echo ""

echo -e "${BLUE}[5/5]${NC} Configuration..."
if [ ! -f config/config.yaml ]; then
    [ -f config/config.yaml.example ] && cp config/config.yaml.example config/config.yaml
    echo -e "${GREEN}✓ config.yaml créé${NC}"
else
    echo -e "${YELLOW}⚠ config.yaml présent${NC}"
fi

if [ ! -f .env ]; then
    [ -f .env.example ] && cp .env.example .env
    echo -e "${GREEN}✓ .env créé${NC}"
else
    echo -e "${YELLOW}⚠ .env présent${NC}"
fi
echo ""

echo -e "${GREEN}✅ Initialisation terminée !${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${CYAN}📋 INFORMATIONS À PARTAGER${NC}"
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
echo "2. Éditez config/config.yaml"
echo "3. Lancez: ${YELLOW}docker-compose up -d${NC}"
echo "4. Configurez la clé: ${YELLOW}http://localhost:3000/setup${NC}"
echo "5. ${RED}SAUVEGARDEZ LA CLÉ DANS BITWARDEN !${NC}"
echo ""
