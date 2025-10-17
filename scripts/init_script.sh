#!/bin/bash
set -e

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║     🪸  ANEMONE - Initialisation      ║"
echo "╔═══════════════════════════════════════╗"
echo -e "${NC}"
echo ""

# Vérifier les dépendances
echo -e "${BLUE}[1/7]${NC} Vérification des dépendances..."

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker n'est pas installé${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose n'est pas installé${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker et Docker Compose installés${NC}"
echo ""

# Créer la structure de dossiers
echo -e "${BLUE}[2/7]${NC} Création de la structure..."

mkdir -p config/{wireguard,ssh,samba}
mkdir -p data
mkdir -p backups
mkdir -p logs
mkdir -p services/{restic,api}

echo -e "${GREEN}✓ Structure créée${NC}"
echo ""

# Générer les clés WireGuard
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

# Générer le mot de passe Restic
echo -e "${BLUE}[4/7]${NC} Génération du mot de passe Restic..."

if [ ! -f config/restic-password ]; then
    python3 -c "import secrets; print(secrets.token_urlsafe(32))" > config/restic-password
    chmod 600 config/restic-password
    echo -e "${GREEN}✓ Mot de passe Restic généré${NC}"
else
    echo -e "${YELLOW}⚠ Mot de passe Restic déjà présent${NC}"
fi
echo ""

# Générer les clés SSH pour Restic
echo -e "${BLUE}[5/7]${NC} Génération des clés SSH..."

if [ ! -f config/ssh/id_rsa ]; then
    ssh-keygen -t rsa -b 4096 -f config/ssh/id_rsa -N "" -C "restic@anemone" -q
    chmod 600 config/ssh/id_rsa
    chmod 644 config/ssh/id_rsa.pub
    
    # Créer le fichier authorized_keys vide
    touch config/ssh/authorized_keys
    chmod 600 config/ssh/authorized_keys
    
    echo -e "${GREEN}✓ Clés SSH générées${NC}"
else
    echo -e "${YELLOW}⚠ Clés SSH déjà présentes${NC}"
fi
echo ""

# Copier le fichier de configuration exemple
echo -e "${BLUE}[6/7]${NC} Configuration..."

if [ ! -f config/config.yaml ]; then
    if [ -f config/config.yaml.example ]; then
        cp config/config.yaml.example config/config.yaml
        echo -e "${GREEN}✓ Fichier config.yaml créé${NC}"
    else
        echo -e "${YELLOW}⚠ Fichier config.yaml.example non trouvé${NC}"
        echo -e "${YELLOW}  Créez manuellement config/config.yaml${NC}"
    fi
else
    echo -e "${YELLOW}⚠ config.yaml déjà présent${NC}"
fi

# Copier le fichier .env exemple
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}✓ Fichier .env créé${NC}"
    else
        echo -e "${YELLOW}⚠ Fichier .env.example non trouvé${NC}"
    fi
else
    echo -e "${YELLOW}⚠ .env déjà présent${NC}"
fi
echo ""

# Afficher les informations importantes
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
echo -e "${YELLOW}🔑 Clé publique SSH (pour backups) :${NC}"
cat config/ssh/id_rsa.pub
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${CYAN}📝 PROCHAINES ÉTAPES${NC}"
echo ""
echo "1. ${YELLOW}Configurez votre DNS dynamique${NC}"
echo "   → DuckDNS : https://www.duckdns.org"
echo "   → No-IP : https://www.noip.com"
echo ""
echo "2. ${YELLOW}Configurez le port-forwarding sur votre box${NC}"
echo "   → Protocole : UDP"
echo "   → Port externe : 51820"
echo "   → Port interne : 51820"
echo "   → IP : $(hostname -I | awk '{print $1}')"
echo ""
echo "3. ${YELLOW}Éditez les fichiers de configuration${NC}"
echo "   → .env (mots de passe SMB/WebDAV)"
echo "   → config/config.yaml (nom du nœud, DNS, pairs)"
echo ""
echo "4. ${YELLOW}Démarrez Anemone${NC}"
echo "   → docker-compose up -d"
echo ""
echo "5. ${YELLOW}Ajoutez vos pairs${NC}"
echo "   → ./scripts/add-peer.sh"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${GREEN}📖 Documentation complète :${NC}"
echo "   https://github.com/juste-un-gars/anemone"
echo ""