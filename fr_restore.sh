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

BACKUP_FILE="$1"

# Vérifier Docker Compose et déterminer la commande à utiliser
DOCKER_COMPOSE_CMD=""
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
else
    echo -e "${RED}❌ Docker Compose n'est pas installé${NC}"
    echo "   Installez Docker Compose v2 : https://docs.docker.com/compose/install/"
    exit 1
fi

echo -e "${CYAN}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🪸 Anemone - Restauration depuis backup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${NC}"

# Vérifier le fichier de backup
if [ -z "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ Erreur : Fichier de backup non spécifié${NC}"
    echo ""
    echo "Usage : $0 <fichier_backup.enc>"
    echo ""
    echo "Exemple : $0 anemone-backup-FR1-20250122-143000.enc"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ Fichier introuvable : $BACKUP_FILE${NC}"
    exit 1
fi

# Afficher infos fichier
FILE_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
FILE_DATE=$(stat -c %y "$BACKUP_FILE" 2>/dev/null | cut -d' ' -f1 || date -r "$BACKUP_FILE" "+%Y-%m-%d" 2>/dev/null || echo "inconnu")

echo ""
echo -e "${BLUE}📄 Fichier de backup : ${CYAN}$BACKUP_FILE${NC}"
echo -e "${BLUE}   Taille : ${CYAN}$FILE_SIZE${NC}"
echo -e "${BLUE}   Date   : ${CYAN}$FILE_DATE${NC}"
echo ""

# Demander la clé Restic
echo -e "${YELLOW}🔑 Clé de déchiffrement Restic${NC}"
echo "   Entrez la clé que vous avez sauvegardée dans Bitwarden"
echo ""
read -s -p "Clé Restic : " RESTIC_KEY
echo ""

if [ -z "$RESTIC_KEY" ]; then
    echo -e "${RED}❌ La clé ne peut pas être vide${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Vérification du backup${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Créer un répertoire temporaire
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "🔍 Déchiffrement et vérification..."

# Utiliser Python pour déchiffrer et vérifier
python3 << EOF
import sys
import tarfile
import io
from pathlib import Path
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2
from cryptography.hazmat.backends import default_backend

try:
    # Lire le fichier chiffré
    with open('$BACKUP_FILE', 'rb') as f:
        encrypted_data = f.read()

    # Extraire IV et données
    iv = encrypted_data[:16]
    ciphertext = encrypted_data[16:]

    # Dériver la clé de chiffrement depuis la clé Restic
    kdf = PBKDF2(
        algorithm=hashes.SHA256(),
        length=32,
        salt=b'anemone-config-backup',
        iterations=100000,
        backend=default_backend()
    )
    encryption_key = kdf.derive('$RESTIC_KEY'.encode())

    # Déchiffrer
    cipher = Cipher(
        algorithms.AES(encryption_key),
        modes.CBC(iv),
        backend=default_backend()
    )
    decryptor = cipher.decryptor()
    padded_data = decryptor.update(ciphertext) + decryptor.finalize()

    # Retirer le padding PKCS7
    padding_length = padded_data[-1]
    data = padded_data[:-padding_length]

    # Vérifier que c'est un tar valide
    tar_file = tarfile.open(fileobj=io.BytesIO(data))
    members = tar_file.getnames()

    print(f'✅ Backup valide ({len(members)} fichiers)')

    # Extraire dans le répertoire temporaire
    tar_file.extractall('$TEMP_DIR')
    tar_file.close()

    sys.exit(0)

except Exception as e:
    print(f'❌ Erreur : {e}', file=sys.stderr)
    sys.exit(1)
EOF

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Échec du déchiffrement${NC}"
    echo "   Vérifiez que la clé est correcte"
    exit 1
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Contenu du backup${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Analyser le contenu
if [ -f "$TEMP_DIR/config.yaml" ]; then
    SERVER_NAME=$(grep "^  name:" "$TEMP_DIR/config.yaml" | awk '{print $2}' || echo "inconnu")
    ENDPOINT=$(grep "^  endpoint:" "$TEMP_DIR/config.yaml" | awk '{print $2}' || echo "inconnu")
    PEER_COUNT=$(grep -c "^  - name:" "$TEMP_DIR/config.yaml" || echo "0")

    echo "  📝 Serveur : $SERVER_NAME"
    echo "  🌐 Endpoint : $ENDPOINT"
    echo "  👥 Pairs configurés : $PEER_COUNT"
else
    echo "  ${YELLOW}⚠️  Fichier config.yaml non trouvé dans le backup${NC}"
fi

echo ""
echo -e "${YELLOW}⚠️  Cette opération va :${NC}"
echo "   • Écraser la configuration actuelle"
echo "   • Restaurer les clés WireGuard et SSH"
echo "   • Restaurer la configuration des pairs"
echo ""
read -p "Continuer ? (oui/non) : " -r CONFIRM

if [ "$CONFIRM" != "oui" ]; then
    echo -e "${RED}❌ Annulé${NC}"
    exit 0
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Restauration en cours${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Créer les répertoires config s'ils n'existent pas
mkdir -p config/wireguard config/ssh

# Restaurer les fichiers
echo "📦 Copie des fichiers de configuration..."
cp -r "$TEMP_DIR"/* config/ 2>/dev/null || true

# Sauvegarder la clé Restic dans un fichier temporaire pour setup
echo "$RESTIC_KEY" > /tmp/.restic-key-restore
chmod 600 /tmp/.restic-key-restore

echo -e "${GREEN}✅ Configuration restaurée${NC}"

# Vérifier et valider l'adresse VPN
if [ -f "config/config.yaml" ]; then
    # Extraire l'adresse avec guillemets
    CURRENT_VPN_ADDRESS_RAW=$(grep "address:" config/config.yaml | head -1 | awk '{print $2}')
    # Extraire l'adresse sans guillemets pour l'affichage
    CURRENT_VPN_ADDRESS=$(echo "$CURRENT_VPN_ADDRESS_RAW" | tr -d '"')

    if [ -n "$CURRENT_VPN_ADDRESS" ]; then
        echo ""
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}  🌐 Vérification de l'adresse VPN${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "Adresse VPN dans le fichier de configuration : ${CYAN}$CURRENT_VPN_ADDRESS${NC}"
        echo ""
        echo -e "${YELLOW}⚠️  Rappel : Chaque serveur Anemone doit avoir une adresse IP VPN unique !${NC}"
        echo ""
        read -p "Souhaitez-vous garder cette adresse ? (oui/non) : " KEEP_VPN_ADDRESS

        if [ "$KEEP_VPN_ADDRESS" != "oui" ]; then
            echo ""
            read -p "Nouvelle adresse VPN (ex: 10.8.0.2/24) : " NEW_VPN_ADDRESS

            if [ -n "$NEW_VPN_ADDRESS" ]; then
                # Mettre à jour config.yaml (en préservant les guillemets)
                sed -i "s|address: \"$CURRENT_VPN_ADDRESS\"|address: \"$NEW_VPN_ADDRESS\"|g" config/config.yaml

                # Mettre à jour wg0.conf si existant
                if [ -f "config/wg_confs/wg0.conf" ]; then
                    sed -i "s|Address = $CURRENT_VPN_ADDRESS|Address = $NEW_VPN_ADDRESS|g" config/wg_confs/wg0.conf
                fi

                echo -e "${GREEN}✅ Adresse VPN mise à jour : $NEW_VPN_ADDRESS${NC}"
            fi
        else
            echo -e "${GREEN}✅ Adresse VPN conservée : $CURRENT_VPN_ADDRESS${NC}"
        fi
    fi
fi

# Vérifier si la configuration de stockage existe
DOCKER_PROFILES=""
if [ -f "config/.anemone-storage-config" ]; then
    STORAGE_TYPE=$(grep "storage_type:" config/.anemone-storage-config | cut -d: -f2 | tr -d ' ')

    if [ "$STORAGE_TYPE" = "integrated_shares" ]; then
        echo ""
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}  📂 Partage intégré détecté${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "L'ancien serveur utilisait le partage intégré (Samba + WebDAV)."
        echo ""
        read -p "Voulez-vous continuer à utiliser le partage intégré ? (oui/non) : " USE_INTEGRATED_SHARES

        if [ "$USE_INTEGRATED_SHARES" = "oui" ]; then
            DOCKER_PROFILES="--profile shares"

            echo ""
            echo -e "${BLUE}Configuration des identifiants de partage...${NC}"
            read -p "👤 Nom d'utilisateur (par défaut: anemone) : " SHARE_USERNAME
            SHARE_USERNAME=${SHARE_USERNAME:-anemone}

            while true; do
                read -s -p "🔐 Mot de passe pour ${SHARE_USERNAME} : " SHARE_PASSWORD
                echo ""
                read -s -p "🔐 Confirmez le mot de passe : " SHARE_PASSWORD_CONFIRM
                echo ""

                if [ "$SHARE_PASSWORD" = "$SHARE_PASSWORD_CONFIRM" ]; then
                    break
                else
                    echo -e "${RED}❌ Les mots de passe ne correspondent pas. Réessayez.${NC}"
                fi
            done

            # Mettre à jour config.yaml avec les identifiants
            if [ -f config/config.yaml ]; then
                sed -i "/services:/,/smb:/{s/username: .*/username: \"${SHARE_USERNAME}\"/}" config/config.yaml
                sed -i "/services:/,/smb:/{s/password: .*/password: \"${SHARE_PASSWORD}\"/}" config/config.yaml
                sed -i "/webdav:/,/ssl:/{s/username: .*/username: \"${SHARE_USERNAME}\"/}" config/config.yaml
                sed -i "/webdav:/,/ssl:/{s/password: .*/password: \"${SHARE_PASSWORD}\"/}" config/config.yaml
                echo -e "${GREEN}✅ Identifiants configurés${NC}"
            fi
        else
            echo -e "${YELLOW}ℹ️  Le partage intégré ne sera pas activé${NC}"
        fi
    elif [ "$STORAGE_TYPE" = "network_mount" ]; then
        echo ""
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}  🌐 Configuration de montage réseau détectée${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""

        # Récupérer les chemins des partages
        OLD_NETWORK_BACKUP=$(grep "network_backup_path:" config/.anemone-storage-config | cut -d: -f2- | tr -d ' ')
        OLD_NETWORK_BACKUPS=$(grep "network_backups_path:" config/.anemone-storage-config | cut -d: -f2- | tr -d ' ')

        echo "Ancien serveur utilisait :"
        echo "  • Backup  : ${CYAN}$OLD_NETWORK_BACKUP${NC}"
        echo "  • Backups : ${CYAN}$OLD_NETWORK_BACKUPS${NC}"
        echo ""
        read -p "Voulez-vous remonter ces partages réseau ? (oui/non) : " REMOUNT_SHARES

        if [ "$REMOUNT_SHARES" = "oui" ]; then
            echo ""
            read -p "Les chemins sont-ils toujours corrects ? (oui/non) : " PATHS_CORRECT

            if [ "$PATHS_CORRECT" = "oui" ]; then
                # Utiliser les anciens chemins
                NETWORK_BACKUP="$OLD_NETWORK_BACKUP"
                NETWORK_BACKUPS="$OLD_NETWORK_BACKUPS"
            else
                # Demander les nouveaux chemins
                echo ""
                echo -e "${YELLOW}Veuillez entrer les nouveaux chemins réseau :${NC}"
                echo ""
                echo "Entrez les informations du partage réseau pour les données utilisateur :"
                read -p "  Serveur/Partage (ex: //192.168.1.10/backup) : " NETWORK_BACKUP
                echo ""
                echo "Entrez les informations du partage réseau pour les backups reçus :"
                read -p "  Serveur/Partage (ex: //192.168.1.10/backups) : " NETWORK_BACKUPS
            fi
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

            # Monter les partages
            echo "📌 Montage des partages réseau..."
            sudo mount -t cifs "$NETWORK_BACKUP" /mnt/anemone/backup -o credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 || {
                echo -e "${RED}❌ Échec du montage de $NETWORK_BACKUP${NC}"
            }

            sudo mount -t cifs "$NETWORK_BACKUPS" /mnt/anemone/backups -o credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 || {
                echo -e "${RED}❌ Échec du montage de $NETWORK_BACKUPS${NC}"
            }

            # Ajouter à fstab
            sudo cp /etc/fstab /etc/fstab.backup.$(date +%Y%m%d-%H%M%S)

            if ! grep -qF "$NETWORK_BACKUP" /etc/fstab; then
                echo "$NETWORK_BACKUP /mnt/anemone/backup cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0" | sudo tee -a /etc/fstab > /dev/null
            fi

            if ! grep -qF "$NETWORK_BACKUPS" /etc/fstab; then
                echo "$NETWORK_BACKUPS /mnt/anemone/backups cifs credentials=/root/.anemone-cifs-credentials,iocharset=utf8,file_mode=0777,dir_mode=0777 0 0" | sudo tee -a /etc/fstab > /dev/null
            fi

            # Créer .env
            cat > .env << EOFENV
# Configuration générée par fr_restore.sh
BACKUP_DATA_PATH=/mnt/anemone/backup
BACKUP_RECEIVE_PATH=/mnt/anemone/backups
EOFENV

            # Mettre à jour .anemone-storage-config avec les chemins utilisés
            cat > config/.anemone-storage-config << EOF
# Configuration de stockage Anemone
# Ce fichier est sauvegardé avec les backups de configuration
storage_type: network_mount
network_backup_path: ${NETWORK_BACKUP}
network_backups_path: ${NETWORK_BACKUPS}
EOF

            echo -e "${GREEN}✅ Partages réseau remontés${NC}"
            echo ""
            read -p "Voulez-vous restaurer les données depuis un pair ? (oui/non) : " RESTORE_FROM_PEER
        else
            echo ""
            echo -e "${YELLOW}ℹ️  Passage en mode stockage local${NC}"
            echo "   Vous devrez probablement restaurer vos données depuis un pair"
            echo ""
            RESTORE_FROM_PEER="oui"
        fi
    fi
fi

# Démarrer Docker
echo ""
echo "🐳 Démarrage de Docker..."
$DOCKER_COMPOSE_CMD $DOCKER_PROFILES up -d --build

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✅ Restauration terminée !${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📋 PROCHAINES ÉTAPES :${NC}"
echo ""
echo "1. 🔐 Finaliser la configuration Restic"
echo "   • Accédez à : ${CYAN}http://localhost:3000/setup${NC}"
echo "   • Choisissez 'Restauration'"
echo "   • Collez votre clé Restic (même clé que pour la restauration)"
echo ""
echo "2. 🔄 Restaurer vos données depuis un pair"
echo "   • ${CYAN}http://localhost:3000/recovery${NC} → Restaurer depuis peer"
echo "   • Choisissez le peer source (${SERVER_NAME} est maintenant reconnecté)"
echo "   • Mode simulation puis restauration"
echo ""
echo "3. ✅ Vérifier que tout fonctionne"
echo "   • Dashboard : ${CYAN}http://localhost:3000/${NC}"
echo "   • Logs : ${CYAN}$DOCKER_COMPOSE_CMD logs -f${NC}"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}⚠️  Rappel : Votre clé temporaire est dans /tmp/.restic-key-restore${NC}"
echo -e "${RED}   Supprimez-la après avoir finalisé le setup !${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
