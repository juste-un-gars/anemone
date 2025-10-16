#!/bin/bash
set -e

echo "🪸 Anemone Restic Service starting..."

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}
RESTIC_PASSWORD_FILE=${RESTIC_PASSWORD_FILE:-/config/restic-password}

export RESTIC_PASSWORD_FILE

BACKUP_MODE=$(python3 -c "
import yaml
with open('$CONFIG_PATH') as f:
    config = yaml.safe_load(f)
    print(config.get('backup', {}).get('mode', 'scheduled'))
")

echo "📋 Backup mode: $BACKUP_MODE"

if [ -f /config/ssh/id_rsa ]; then
    cp /config/ssh/id_rsa /root/.ssh/id_rsa
    chmod 600 /root/.ssh/id_rsa
fi

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
