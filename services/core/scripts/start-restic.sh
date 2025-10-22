#!/bin/bash
set -e

echo "🪸 Starting Anemone Restic Service..."

# Attendre que WireGuard soit prêt
echo "⏳ Waiting for WireGuard to be ready..."
max_wait=30
waited=0
while [ $waited -lt $max_wait ]; do
    if ip link show wg0 >/dev/null 2>&1; then
        echo "✅ WireGuard is ready"
        break
    fi
    sleep 1
    waited=$((waited + 1))
done

if [ $waited -ge $max_wait ]; then
    echo "⚠️  WireGuard not ready after ${max_wait}s, continuing anyway..."
fi

# Vérifier si le setup est complété
if [ ! -f /config/.setup-completed ]; then
    echo "❌ Setup not completed"
    echo "   Please access http://localhost:3000/setup"
    sleep infinity
fi

# Déchiffrer la clé Restic
echo "🔓 Loading Restic password..."

# Option 1: Mot de passe en clair (pour tests ou migration)
if [ -f /config/restic-password ]; then
    echo "📄 Using plaintext password file (legacy/test mode)"
    export RESTIC_PASSWORD=$(cat /config/restic-password)

    if [ -z "$RESTIC_PASSWORD" ]; then
        echo "❌ Password file is empty"
        exit 1
    fi

    echo "✅ Restic password loaded from plaintext file"

# Option 2: Clé chiffrée (mode normal)
elif [ -f /config/.restic.encrypted ] && [ -f /config/.restic.salt ]; then
    echo "🔐 Decrypting encrypted password..."
    export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py)

    if [ -z "$RESTIC_PASSWORD" ]; then
        echo "❌ Failed to decrypt key"
        echo "   If you need to use plaintext password, create: /config/restic-password"
        exit 1
    fi

    echo "✅ Restic password decrypted"

# Aucune méthode disponible
else
    echo "❌ No password found"
    echo "   - Encrypted: /config/.restic.encrypted (not found)"
    echo "   - Plaintext: /config/restic-password (not found)"
    echo "   Please complete setup at http://localhost:3000/setup"
    exit 1
fi

# Copier clé SSH
if [ -f /config/ssh/id_rsa ]; then
    cp /config/ssh/id_rsa /root/.ssh/id_rsa
    chmod 600 /root/.ssh/id_rsa
fi

# Type et mode de backup
BACKUP_CONFIG=$(python3 -c "
import yaml
import json
try:
    with open('/config/config.yaml') as f:
        config = yaml.safe_load(f)
        backup = config.get('backup', {})
        print(json.dumps({
            'type': backup.get('type', 'snapshot'),
            'mode': backup.get('mode', 'scheduled')
        }))
except:
    print(json.dumps({'type': 'snapshot', 'mode': 'scheduled'}))
")

BACKUP_TYPE=$(echo "$BACKUP_CONFIG" | python3 -c "import sys, json; print(json.load(sys.stdin)['type'])")
BACKUP_MODE=$(echo "$BACKUP_CONFIG" | python3 -c "import sys, json; print(json.load(sys.stdin)['mode'])")

echo "📋 Backup type: $BACKUP_TYPE"
echo "📋 Backup mode: $BACKUP_MODE"

# Exporter le type pour les scripts enfants
export BACKUP_TYPE

# Initialiser repos
echo "🔧 Initializing repositories..."
/scripts/init-repos.sh

# Déterminer le script à lancer selon le type
if [ "$BACKUP_TYPE" = "sync" ]; then
    SCRIPT_PREFIX="/scripts/sync"
    echo "🔄 Using rsync synchronization (mirror mode)"
else
    SCRIPT_PREFIX="/scripts/backup"
    echo "📸 Using Restic snapshots (history mode)"
fi

# Démarrer selon le mode
case "$BACKUP_MODE" in
    "live")
        echo "🔴 LIVE mode"
        exec ${SCRIPT_PREFIX}-live.sh
        ;;
    "periodic")
        echo "🟡 PERIODIC mode"
        exec ${SCRIPT_PREFIX}-periodic.sh
        ;;
    "scheduled")
        echo "🟢 SCHEDULED mode"
        # Pour scheduled, on utilise setup-cron.sh qui doit être adapté
        /scripts/setup-cron.sh
        exec crond -f -l 2
        ;;
    *)
        echo "❌ Unknown mode: $BACKUP_MODE"
        exit 1
        ;;
esac
