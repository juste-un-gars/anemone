#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

set -e

# Script de restauration des données depuis un peer en cas de crash serveur
# Usage: ./restore-from-peer.sh <peer_name> [--dry-run]

PEER_NAME="$1"
DRY_RUN=""

if [ "$2" = "--dry-run" ]; then
    DRY_RUN="--dry-run"
    echo "🔍 MODE DRY-RUN : Aucune modification ne sera effectuée"
fi

if [ -z "$PEER_NAME" ]; then
    echo "❌ Usage: $0 <peer_name> [--dry-run]"
    exit 1
fi

# Charger la configuration
CONFIG_FILE="${CONFIG_PATH:-/config/config.yaml}"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Fichier de configuration introuvable : $CONFIG_FILE"
    exit 1
fi

# Extraire l'IP du peer depuis la configuration
PEER_IP=$(python3 -c "
import yaml
import sys

try:
    with open('$CONFIG_FILE') as f:
        config = yaml.safe_load(f)

    peers = config.get('peers', [])
    peer = next((p for p in peers if p.get('name') == '$PEER_NAME'), None)

    if not peer:
        print('PEER_NOT_FOUND', file=sys.stderr)
        sys.exit(1)

    allowed_ips = peer.get('allowed_ips', '')
    if not allowed_ips:
        print('NO_IP', file=sys.stderr)
        sys.exit(1)

    # Extraire l'IP (format: 10.8.0.2/32)
    ip = allowed_ips.split('/')[0]
    print(ip)
except Exception as e:
    print(f'ERROR: {e}', file=sys.stderr)
    sys.exit(1)
")

if [ $? -ne 0 ]; then
    echo "❌ Erreur : Peer '$PEER_NAME' introuvable ou mal configuré"
    exit 1
fi

echo "🔄 Restauration depuis le peer : $PEER_NAME ($PEER_IP)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Vérifier la connectivité
echo "🔍 Test de connectivité..."
if ! ping -c 2 -W 3 "$PEER_IP" > /dev/null 2>&1; then
    echo "❌ Peer injoignable : $PEER_IP"
    exit 1
fi
echo "✅ Peer accessible"

# Vérifier l'accès SSH
echo "🔍 Test de connexion SSH..."
if ! ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes "restic@$PEER_IP" "echo OK" > /dev/null 2>&1; then
    echo "❌ Connexion SSH impossible. Vérifiez les clés SSH."
    exit 1
fi
echo "✅ Connexion SSH OK"

# Destination locale
LOCAL_BACKUP_PATH="/mnt/backup"

# Source distante (sur le peer)
# Le peer stocke nos données dans /backups/<notre_serveur_name>
SERVER_NAME=$(python3 -c "
import yaml
with open('$CONFIG_FILE') as f:
    config = yaml.safe_load(f)
print(config.get('server', {}).get('name', 'unknown'))
")

REMOTE_PATH="restic@${PEER_IP}:/mnt/backup"

echo ""
echo "📂 Source  : $REMOTE_PATH"
echo "📁 Destination : $LOCAL_BACKUP_PATH"
echo ""

if [ -n "$DRY_RUN" ]; then
    echo "⚠️  MODE DRY-RUN ACTIVÉ - Simulation uniquement"
    echo ""
fi

# Demander confirmation si pas en dry-run
if [ -z "$DRY_RUN" ]; then
    echo "⚠️  ATTENTION : Cette opération va écraser les données locales !"
    echo "   Les fichiers présents localement mais pas sur le peer seront conservés."
    echo "   Les fichiers différents seront mis à jour avec la version du peer."
    echo ""
    read -p "Continuer ? (yes/no) : " -r CONFIRM

    if [ "$CONFIRM" != "yes" ]; then
        echo "❌ Annulé par l'utilisateur"
        exit 0
    fi
fi

# Créer le répertoire de destination si nécessaire
mkdir -p "$LOCAL_BACKUP_PATH"

# Options rsync
RSYNC_OPTS=(
    -avz
    --progress
    --stats
    --exclude='.trash/'
    --exclude='*.tmp'
    --exclude='*.part'
    --exclude='~*'
    -e "ssh -o StrictHostKeyChecking=no"
)

if [ -n "$DRY_RUN" ]; then
    RSYNC_OPTS+=("$DRY_RUN")
fi

# Lancer la restauration
echo "🚀 Démarrage de la restauration..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if rsync "${RSYNC_OPTS[@]}" "$REMOTE_PATH/" "$LOCAL_BACKUP_PATH/"; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    if [ -n "$DRY_RUN" ]; then
        echo "✅ Simulation terminée avec succès"
        echo "   Relancez sans --dry-run pour effectuer la restauration réelle"
    else
        echo "✅ Restauration terminée avec succès !"
        echo "   Données restaurées depuis $PEER_NAME vers $LOCAL_BACKUP_PATH"
    fi
    exit 0
else
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "❌ Erreur lors de la restauration"
    exit 1
fi
