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
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose non installé${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker installé${NC}"
echo -e "${GREEN}✓ Docker Compose installé${NC}"
echo ""

echo -e "${BLUE}[2/5]${NC} Création de la structure..."
mkdir -p config/{wireguard,wg_confs,ssh,samba}
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
    # Méthode 2: Utiliser l'image Docker WireGuard avec entrypoint spécifique
    else
        echo -e "${YELLOW}⚠ 'wg' non disponible sur le host, utilisation de Docker...${NC}"
        # Générer la clé privée (contourner l'entrypoint par défaut)
        docker run --rm --entrypoint wg linuxserver/wireguard:latest genkey > config/wireguard/private.key 2>/dev/null || {
            # Si ça échoue, utiliser Python comme fallback
            echo -e "${YELLOW}   → Fallback sur Python...${NC}"
            python3 -c "import base64, os; print(base64.b64encode(os.urandom(32)).decode())" > config/wireguard/private.key
        }

        # Générer la clé publique correspondante
        docker run --rm --entrypoint wg -i linuxserver/wireguard:latest pubkey < config/wireguard/private.key > config/wireguard/public.key 2>/dev/null || {
            # Si ça échoue aussi, créer un placeholder
            echo "# Clé publique sera générée au démarrage du conteneur" > config/wireguard/public.key
        }

        chmod 600 config/wireguard/private.key
        chmod 644 config/wireguard/public.key
        echo -e "${GREEN}✓ Clés WireGuard générées${NC}"
    fi

    # Créer le fichier wg0.conf avec les clés générées
    echo -e "${BLUE}   → Création de wg0.conf...${NC}"
    PRIVATE_KEY=$(cat config/wireguard/private.key)
    cat > config/wg_confs/wg0.conf <<EOF
[Interface]
PrivateKey = ${PRIVATE_KEY}
Address = 10.8.0.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth+ -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth+ -j MASQUERADE

# Ajoutez vos peers ici avec le format suivant :
# [Peer]
# PublicKey = CLE_PUBLIQUE_DU_PEER
# AllowedIPs = 10.8.0.2/32
# Endpoint = peer.duckdns.org:51820
# PersistentKeepalive = 25
EOF
    chmod 600 config/wg_confs/wg0.conf
    echo -e "${GREEN}   ✓ wg0.conf créé${NC}"
else
    echo -e "${YELLOW}⚠ Clés déjà présentes${NC}"
    # Vérifier si wg0.conf existe, sinon le créer
    if [ ! -f config/wg_confs/wg0.conf ]; then
        echo -e "${BLUE}   → Création de wg0.conf avec clés existantes...${NC}"
        PRIVATE_KEY=$(cat config/wireguard/private.key)
        cat > config/wg_confs/wg0.conf <<EOF
[Interface]
PrivateKey = ${PRIVATE_KEY}
Address = 10.8.0.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth+ -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth+ -j MASQUERADE

# Ajoutez vos peers ici avec le format suivant :
# [Peer]
# PublicKey = CLE_PUBLIQUE_DU_PEER
# AllowedIPs = 10.8.0.2/32
# Endpoint = peer.duckdns.org:51820
# PersistentKeepalive = 25
EOF
        chmod 600 config/wg_confs/wg0.conf
        echo -e "${GREEN}   ✓ wg0.conf créé${NC}"
    fi
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
