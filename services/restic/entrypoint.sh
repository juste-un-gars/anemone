#!/bin/bash
set -e

echo "🪸 Anemone Restic Service starting..."

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

if [ ! -f "$CONFIG_PATH" ]; then
    echo "❌ Configuration file not found: $CONFIG_PATH"
    exit 1
fi

# Vérifier si le setup est complété
if [ ! -f /config/.setup-completed ]; then
    echo "❌ Setup not completed"
    echo "   Please access http://localhost:3000/setup"
    sleep infinity
fi

# Déchiffrer la clé Restic
echo "🔓 Decrypting Restic key..."

if [ ! -f /config/.restic.encrypted ] || [ ! -f /config/.restic.salt ]; then
    echo "❌ Encrypted key or salt not found"
    exit 1
fi

# Déchiffrer avec Python cryptography
export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py)

if [ -z "$RESTIC_PASSWORD" ]; then
    echo "❌ Failed to decrypt key"
    exit 1
fi

echo "✅ Restic key decrypted"

# Copier clé SSH
if [ -f /config/ssh/id_rsa ]; then
    cp /config/ssh/id_rsa /root/.ssh/id_rsa
    chmod 600 /root/.ssh/id_rsa
fi

# Mode de backup
BACKUP_MODE=$(python3 -c "
import yaml
try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        print(config.get('backup', {}).get('mode', 'scheduled'))
except:
    print('scheduled')
")

echo "📋 Backup mode: $BACKUP_MODE"

# Initialiser repos
echo "🔧 Initializing repositories..."
/scripts/init-repos.sh

# Démarrer selon le mode
case "$BACKUP_MODE" in
    "live")
        echo "🔴 LIVE mode"
        exec /scripts/backup-live.sh
        ;;
    "periodic")
        echo "🟡 PERIODIC mode"
        exec /scripts/backup-periodic.sh
        ;;
    "scheduled")
        echo "🟢 SCHEDULED mode"
        /scripts/setup-cron.sh
        exec crond -f -l 2
        ;;
    *)
        echo "❌ Unknown mode: $BACKUP_MODE"
        exit 1
        ;;
esac
