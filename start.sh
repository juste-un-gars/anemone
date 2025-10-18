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
fi

# Configuration interactive
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}   Configuration du serveur Anemone${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Demander le nom du node
echo -e "${BLUE}📛 Nom du serveur (node name)${NC}"
echo -e "${YELLOW}   Utilisé pour identifier ce serveur dans le réseau${NC}"
read -p "   Nom du node [anemone-home]: " NODE_NAME
NODE_NAME=${NODE_NAME:-anemone-home}
echo ""

# Demander le nom d'utilisateur
echo -e "${BLUE}👤 Nom d'utilisateur${NC}"
echo -e "${YELLOW}   Utilisé pour SMB et WebDAV${NC}"
read -p "   Nom d'utilisateur [anemone]: " USERNAME
USERNAME=${USERNAME:-anemone}
echo ""

# Demander le mot de passe
echo -e "${BLUE}🔐 Mot de passe${NC}"
echo -e "${YELLOW}   Utilisé pour SMB et WebDAV${NC}"
while true; do
    read -s -p "   Mot de passe: " PASSWORD
    echo ""
    if [ -z "$PASSWORD" ]; then
        echo -e "${RED}   ⚠  Le mot de passe ne peut pas être vide${NC}"
        continue
    fi
    read -s -p "   Confirmer le mot de passe: " PASSWORD_CONFIRM
    echo ""
    if [ "$PASSWORD" = "$PASSWORD_CONFIRM" ]; then
        break
    else
        echo -e "${RED}   ⚠  Les mots de passe ne correspondent pas${NC}"
    fi
done
echo ""

# Demander l'adresse IP VPN locale
echo -e "${BLUE}🔢 Adresse IP VPN locale${NC}"
echo -e "${YELLOW}   Chaque serveur Anemone doit avoir une IP unique dans le réseau VPN${NC}"
echo -e "${YELLOW}   Format: 10.X.Y.Z/24 (réseau privé en /24)${NC}"
echo -e "${YELLOW}   Exemples: 10.8.0.1/24, 10.9.0.1/24, 192.168.100.1/24${NC}"
while true; do
    read -p "   IP VPN [10.8.0.1/24]: " VPN_ADDRESS
    VPN_ADDRESS=${VPN_ADDRESS:-10.8.0.1/24}

    # Valider le format IP/masque (accepter 10.x.x.x/24, 192.168.x.x/24, 172.16-31.x.x/24)
    if [[ "$VPN_ADDRESS" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$ ]]; then
        # Extraire les octets
        IFS='.' read -r O1 O2 O3 O4_MASK <<< "$VPN_ADDRESS"
        O4=$(echo "$O4_MASK" | cut -d'/' -f1)
        MASK=$(echo "$O4_MASK" | cut -d'/' -f2)

        # Vérifier que c'est une IP privée valide
        if ([ "$O1" -eq 10 ] || \
            ([ "$O1" -eq 192 ] && [ "$O2" -eq 168 ]) || \
            ([ "$O1" -eq 172 ] && [ "$O2" -ge 16 ] && [ "$O2" -le 31 ])) && \
           [ "$O1" -ge 0 ] && [ "$O1" -le 255 ] && \
           [ "$O2" -ge 0 ] && [ "$O2" -le 255 ] && \
           [ "$O3" -ge 0 ] && [ "$O3" -le 255 ] && \
           [ "$O4" -ge 1 ] && [ "$O4" -le 254 ] && \
           [ "$MASK" -ge 8 ] && [ "$MASK" -le 30 ]; then
            break
        fi
    fi
    echo -e "${RED}   ⚠  Format invalide. Utilisez une IP privée avec masque (ex: 10.9.0.1/24)${NC}"
done
echo ""

# Demander l'endpoint public
echo -e "${BLUE}🌍 Endpoint public (optionnel)${NC}"
echo -e "${YELLOW}   Adresse pour que les autres serveurs puissent vous joindre${NC}"
echo -e "${YELLOW}   Exemples: monserveur.duckdns.org:51820 ou 192.168.1.100:51820${NC}"
echo -e "${YELLOW}   Laissez vide si vous n'avez pas d'IP publique/DynDNS${NC}"
read -p "   Endpoint [vide]: " ENDPOINT
ENDPOINT=${ENDPOINT:-""}
echo ""

# Demander le mode de backup
echo -e "${BLUE}💾 Mode de sauvegarde${NC}"
echo -e "${YELLOW}   1) live      - Sauvegarde immédiate à chaque modification (inotify)${NC}"
echo -e "${YELLOW}   2) periodic  - Sauvegarde toutes les 30 minutes${NC}"
echo -e "${YELLOW}   3) scheduled - Sauvegarde planifiée (tous les jours à 2h du matin)${NC}"
read -p "   Choix [2]: " BACKUP_MODE_CHOICE
BACKUP_MODE_CHOICE=${BACKUP_MODE_CHOICE:-2}
case $BACKUP_MODE_CHOICE in
    1) BACKUP_MODE="live" ;;
    3) BACKUP_MODE="scheduled" ;;
    *) BACKUP_MODE="periodic" ;;
esac
echo ""

# Demander la timezone
echo -e "${BLUE}🕐 Fuseau horaire (timezone)${NC}"
echo -e "${YELLOW}   Exemples courants:${NC}"
echo -e "${YELLOW}   - Europe/Paris (France)${NC}"
echo -e "${YELLOW}   - Europe/Brussels (Belgique)${NC}"
echo -e "${YELLOW}   - Europe/Zurich (Suisse)${NC}"
echo -e "${YELLOW}   - America/Montreal (Canada)${NC}"
echo -e "${YELLOW}   Liste complète: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones${NC}"
read -p "   Timezone [Europe/Paris]: " TIMEZONE
TIMEZONE=${TIMEZONE:-Europe/Paris}
echo ""

# Mettre à jour .env
echo -e "${BLUE}💾 Mise à jour de la configuration...${NC}"
if [ -f .env ]; then
    # Mettre à jour les valeurs existantes
    sed -i "s/^SMB_USER=.*/SMB_USER=${USERNAME}/" .env
    sed -i "s/^SMB_PASSWORD=.*/SMB_PASSWORD=${PASSWORD}/" .env
    sed -i "s/^WEBDAV_USER=.*/WEBDAV_USER=${USERNAME}/" .env
    sed -i "s/^WEBDAV_PASSWORD=.*/WEBDAV_PASSWORD=${PASSWORD}/" .env
    sed -i "s|^TIMEZONE=.*|TIMEZONE=${TIMEZONE}|" .env
    echo -e "${GREEN}✓ Fichier .env mis à jour${NC}"
else
    echo -e "${RED}✗ Fichier .env introuvable${NC}"
fi

# Mettre à jour config.yaml
if [ -f config/config.yaml ]; then
    # Mettre à jour le nom du node
    sed -i "s/^  name: .*/  name: \"${NODE_NAME}\"/" config/config.yaml

    # Mettre à jour l'adresse IP VPN
    sed -i "s|^  address: .*|  address: \"${VPN_ADDRESS}\"|" config/config.yaml

    # Mettre à jour l'endpoint public WireGuard
    if [ -n "$ENDPOINT" ]; then
        sed -i "s|^  public_endpoint: .*|  public_endpoint: \"${ENDPOINT}\"|" config/config.yaml
    fi

    # Mettre à jour les credentials SMB/WebDAV
    sed -i "/^services:/,/^wireguard:/ s/username: .*/username: \"${USERNAME}\"/" config/config.yaml
    sed -i "/^services:/,/^wireguard:/ s/password: .*/password: \"${PASSWORD}\"/" config/config.yaml

    # Mettre à jour le mode de backup
    sed -i "s/^  mode: .*/  mode: \"${BACKUP_MODE}\"/" config/config.yaml

    # Mettre à jour la timezone
    sed -i "s|^  timezone: .*|  timezone: \"${TIMEZONE}\"|" config/config.yaml

    echo -e "${GREEN}✓ Fichier config.yaml mis à jour${NC}"
else
    echo -e "${RED}✗ Fichier config.yaml introuvable${NC}"
fi

echo ""
echo -e "${GREEN}✅ Configuration terminée !${NC}"
echo ""
echo -e "${CYAN}Récapitulatif :${NC}"
echo -e "  Node name     : ${GREEN}${NODE_NAME}${NC}"
echo -e "  IP VPN        : ${GREEN}${VPN_ADDRESS}${NC}"
echo -e "  Utilisateur   : ${GREEN}${USERNAME}${NC}"
echo -e "  Mot de passe  : ${GREEN}********${NC}"
if [ -n "$ENDPOINT" ]; then
    echo -e "  Endpoint      : ${GREEN}${ENDPOINT}${NC}"
else
    echo -e "  Endpoint      : ${YELLOW}non configuré${NC}"
fi
echo -e "  Mode backup   : ${GREEN}${BACKUP_MODE}${NC}"
echo -e "  Timezone      : ${GREEN}${TIMEZONE}${NC}"
echo ""

echo ""
echo -e "${BLUE}🔧 Génération de la configuration WireGuard...${NC}"
# Vérifier si les clés existent
if [ -f config/wireguard/private.key ]; then
    # Générer wg0.conf depuis config.yaml
    if python3 scripts/generate-wireguard-config.py config/config.yaml config/wireguard/wg0.conf 2>/dev/null; then
        echo -e "${GREEN}✓ wg0.conf généré${NC}"
    else
        echo -e "${YELLOW}⚠  Erreur lors de la génération de wg0.conf (sera régénéré automatiquement)${NC}"
    fi
else
    echo -e "${YELLOW}⚠  Clés WireGuard manquantes, wg0.conf sera généré au premier ajout de pair${NC}"
fi

echo ""
echo -e "${CYAN}🔨 Construction des images Docker...${NC}"
docker compose build --no-cache

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
