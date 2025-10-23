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

# Vérifier Docker Compose et déterminer la commande à utiliser
DOCKER_COMPOSE_CMD=""
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
    echo -e "${GREEN}✅ Docker Compose v2 détecté${NC}"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
    echo -e "${GREEN}✅ Docker Compose v1 détecté${NC}"
    echo -e "${YELLOW}⚠️  Docker Compose v1 est obsolète, installez le plugin v2${NC}"
else
    echo -e "${RED}❌ Docker Compose n'est pas installé${NC}"
    exit 1
fi

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
read -p "🌐 Adresse extérieure pour le VPN (ex: dyndns, IP publique) : " EXTERNAL_ADDR
read -p "🔌 Port WireGuard (par défaut 51820) : " VPN_PORT
VPN_PORT=${VPN_PORT:-51820}

# Mettre à jour config.yaml si nécessaire
if [ -f "config/config.yaml" ]; then
    echo "📝 Mise à jour de config/config.yaml..."
    sed -i "s/name: .*/name: ${SERVER_NAME}/" config/config.yaml 2>/dev/null || true
    sed -i "s/endpoint: .*/endpoint: ${EXTERNAL_ADDR}:${VPN_PORT}/" config/config.yaml 2>/dev/null || true
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 3b/5 : Configuration du stockage${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Configuration du partage et du stockage
DOCKER_PROFILES=""
USE_NETWORK_SHARES="non"
FSTAB_MODIFIED="non"

echo ""
read -p "📂 Voulez-vous utiliser le partage intégré (Samba + WebDAV) ? (oui/non) : " USE_INTEGRATED_SHARES

if [ "$USE_INTEGRATED_SHARES" = "oui" ]; then
    DOCKER_PROFILES="--profile shares"
    echo -e "${GREEN}✅ Le partage intégré sera activé${NC}"

    # Sauvegarder la configuration de stockage
    mkdir -p config
    cat > config/.anemone-storage-config << EOF
# Configuration de stockage Anemone
# Ce fichier est sauvegardé avec les backups de configuration
storage_type: integrated_shares
EOF
else
    echo -e "${YELLOW}ℹ️  Le partage intégré ne sera pas activé${NC}"
    echo ""
    read -p "🌐 Voulez-vous monter un partage réseau existant ? (oui/non) : " USE_NETWORK_SHARES

    if [ "$USE_NETWORK_SHARES" = "oui" ]; then
        echo ""
        echo -e "${BLUE}Configuration du montage réseau...${NC}"

        # Vérifier si cifs-utils est installé
        if ! dpkg -l | grep -q cifs-utils 2>/dev/null && ! rpm -q cifs-utils &>/dev/null; then
            echo -e "${YELLOW}⚠️  cifs-utils n'est pas installé${NC}"
            read -p "   Voulez-vous l'installer maintenant ? (oui/non) : " INSTALL_CIFS
            if [ "$INSTALL_CIFS" = "oui" ]; then
                if command -v apt-get &> /dev/null; then
                    sudo apt-get update && sudo apt-get install -y cifs-utils
                elif command -v dnf &> /dev/null; then
                    sudo dnf install -y cifs-utils
                elif command -v yum &> /dev/null; then
                    sudo yum install -y cifs-utils
                else
                    echo -e "${RED}❌ Impossible d'installer automatiquement. Installez cifs-utils manuellement.${NC}"
                    exit 1
                fi
            else
                echo -e "${RED}❌ cifs-utils est requis pour monter des partages réseau${NC}"
                exit 1
            fi
        fi

        echo ""
        echo "Entrez les informations du partage réseau pour les données utilisateur :"
        read -p "  Serveur/Partage (ex: //192.168.1.10/backup) : " SMB_BACKUP_PATH
        echo ""
        echo "Entrez les informations du partage réseau pour les backups reçus :"
        read -p "  Serveur/Partage (ex: //192.168.1.10/backups) : " SMB_BACKUPS_PATH
        echo ""
        read -p "👤 Nom d'utilisateur pour les montages : " SMB_USERNAME
        read -s -p "🔐 Mot de passe : " SMB_PASSWORD
        echo ""

        # Créer les répertoires de montage
        sudo mkdir -p /mnt/anemone/backup /mnt/anemone/backups

        # Créer le fichier credentials
        sudo bash -c "cat > /root/.anemone-cifs-credentials << EOF
username=${SMB_USERNAME}
password=${SMB_PASSWORD}
EOF"
        sudo chmod 600 /root/.anemone-cifs-credentials

        # Créer le script de montage
        cat > mount-shares.sh << 'EOFMOUNT'
#!/bin/bash
# Anemone - Script de montage des partages réseau
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0

set -e

CREDENTIALS="/root/.anemone-cifs-credentials"
MOUNT_OPTS="credentials=${CREDENTIALS},iocharset=utf8,file_mode=0777,dir_mode=0777"

# Monter backup (données utilisateur)
if ! mountpoint -q /mnt/anemone/backup; then
    echo "Montage de SMB_BACKUP_PATH_PLACEHOLDER..."
    sudo mount -t cifs "SMB_BACKUP_PATH_PLACEHOLDER" /mnt/anemone/backup -o ${MOUNT_OPTS}
    echo "✅ Monté : /mnt/anemone/backup"
fi

# Monter backups (backups reçus des pairs)
if ! mountpoint -q /mnt/anemone/backups; then
    echo "Montage de SMB_BACKUPS_PATH_PLACEHOLDER..."
    sudo mount -t cifs "SMB_BACKUPS_PATH_PLACEHOLDER" /mnt/anemone/backups -o ${MOUNT_OPTS}
    echo "✅ Monté : /mnt/anemone/backups"
fi

echo "✅ Tous les partages sont montés"
EOFMOUNT

        # Remplacer les placeholders
        sed -i "s|SMB_BACKUP_PATH_PLACEHOLDER|${SMB_BACKUP_PATH}|g" mount-shares.sh
        sed -i "s|SMB_BACKUPS_PATH_PLACEHOLDER|${SMB_BACKUPS_PATH}|g" mount-shares.sh
        chmod +x mount-shares.sh

        # Monter maintenant
        echo ""
        echo "📌 Montage des partages réseau..."
        sudo ./mount-shares.sh

        # Créer/modifier .env pour utiliser les montages
        cat > .env << EOFENV
# Configuration générée par fr_start.sh
BACKUP_DATA_PATH=/mnt/anemone/backup
BACKUP_RECEIVE_PATH=/mnt/anemone/backups
EOFENV

        echo -e "${GREEN}✅ Partages réseau montés et configurés${NC}"

        # Ajouter automatiquement à /etc/fstab
        echo ""
        echo "📝 Ajout des montages à /etc/fstab pour montage automatique au boot..."

        # Backup de fstab
        sudo cp /etc/fstab /etc/fstab.backup.$(date +%Y%m%d-%H%M%S)

        # Vérifier si les entrées existent déjà
        FSTAB_ENTRY_1="${SMB_BACKUP_PATH} /mnt/anemone/backup cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0"
        FSTAB_ENTRY_2="${SMB_BACKUPS_PATH} /mnt/anemone/backups cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0"

        if ! grep -qF "${SMB_BACKUP_PATH}" /etc/fstab; then
            echo "$FSTAB_ENTRY_1" | sudo tee -a /etc/fstab > /dev/null
            echo "  ✅ Ajouté : ${SMB_BACKUP_PATH} → /mnt/anemone/backup"
        else
            echo "  ⚠️  Entrée déjà présente : ${SMB_BACKUP_PATH}"
        fi

        if ! grep -qF "${SMB_BACKUPS_PATH}" /etc/fstab; then
            echo "$FSTAB_ENTRY_2" | sudo tee -a /etc/fstab > /dev/null
            echo "  ✅ Ajouté : ${SMB_BACKUPS_PATH} → /mnt/anemone/backups"
        else
            echo "  ⚠️  Entrée déjà présente : ${SMB_BACKUPS_PATH}"
        fi

        # Valider la configuration fstab (test à blanc)
        if sudo mount -a --fake 2>/dev/null; then
            echo -e "${GREEN}✅ Configuration /etc/fstab validée${NC}"
            FSTAB_MODIFIED="oui"
        else
            echo -e "${YELLOW}⚠️  Validation /etc/fstab : vérifiez manuellement avec 'sudo mount -a'${NC}"
            FSTAB_MODIFIED="oui"
        fi

        # Sauvegarder la configuration de stockage
        cat > config/.anemone-storage-config << EOF
# Configuration de stockage Anemone
# Ce fichier est sauvegardé avec les backups de configuration
storage_type: network_mount
network_backup_path: ${SMB_BACKUP_PATH}
network_backups_path: ${SMB_BACKUPS_PATH}
EOF
        echo ""
    else
        # Stockage local (ni partages intégrés, ni montage réseau)
        cat > config/.anemone-storage-config << EOF
# Configuration de stockage Anemone
# Ce fichier est sauvegardé avec les backups de configuration
storage_type: local
EOF
    fi
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Étape 4/5 : Démarrage de Docker${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "🐳 Construction et démarrage des conteneurs..."
$DOCKER_COMPOSE_CMD $DOCKER_PROFILES up -d --build

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

# Afficher info si fstab a été modifié
if [ "$FSTAB_MODIFIED" = "oui" ]; then
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  📝 Modification /etc/fstab${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${GREEN}✅ Les montages réseau ont été ajoutés à /etc/fstab${NC}"
    echo "   Les partages seront remontés automatiquement au redémarrage"
    echo ""
    echo -e "${YELLOW}ℹ️  Backup créé : /etc/fstab.backup.*${NC}"
    echo ""
fi

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
echo -e "${CYAN}  Logs : $DOCKER_COMPOSE_CMD logs -f${NC}"
echo -e "${CYAN}  Arrêter : $DOCKER_COMPOSE_CMD down${NC}"
echo -e "${CYAN}  Redémarrer : $DOCKER_COMPOSE_CMD restart${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
