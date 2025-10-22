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

# Démarrer Docker
echo ""
echo "🐳 Démarrage de Docker..."
docker-compose up -d --build

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
echo "   • Logs : ${CYAN}docker-compose logs -f${NC}"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}⚠️  Rappel : Votre clé temporaire est dans /tmp/.restic-key-restore${NC}"
echo -e "${RED}   Supprimez-la après avoir finalisé le setup !${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
