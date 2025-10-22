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

# Mode de synchronisation des données utilisateur
BACKUP_MODE=$(python3 -c "
import yaml
try:
    with open('/config/config.yaml') as f:
        config = yaml.safe_load(f)
        print(config.get('backup', {}).get('mode', 'scheduled'))
except:
    print('scheduled')
")

echo "📋 Backup mode: $BACKUP_MODE"
echo "🔄 User data: rsync synchronization (mirror mode)"
echo "📸 Server config: Restic snapshots (handled separately by cron)"

# Démarrer la synchronisation des données selon le mode
case "$BACKUP_MODE" in
    "live")
        echo "🔴 LIVE mode - watching for file changes"
        exec /scripts/sync-live.sh
        ;;
    "periodic")
        echo "🟡 PERIODIC mode - syncing every N minutes"
        exec /scripts/sync-periodic.sh
        ;;
    "scheduled")
        echo "🟢 SCHEDULED mode - syncing on cron schedule"
        /scripts/setup-cron.sh
        exec crond -f -l 2
        ;;
    *)
        echo "❌ Unknown mode: $BACKUP_MODE"
        exit 1
        ;;
esac
