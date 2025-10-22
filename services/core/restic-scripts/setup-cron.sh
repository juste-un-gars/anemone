#!/bin/bash
set -e

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

# Lire la configuration
BACKUP_CONFIG=$(python3 -c "
import yaml
import json
try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        backup = config.get('backup', {})
        print(json.dumps({
            'type': backup.get('type', 'snapshot'),
            'schedule': backup.get('schedule', '0 2 * * *')
        }))
except:
    print(json.dumps({'type': 'snapshot', 'schedule': '0 2 * * *'}))
")

BACKUP_TYPE=$(echo "$BACKUP_CONFIG" | python3 -c "import sys, json; print(json.load(sys.stdin)['type'])")
SCHEDULE=$(echo "$BACKUP_CONFIG" | python3 -c "import sys, json; print(json.load(sys.stdin)['schedule'])")

# Choisir le script selon le type
if [ "$BACKUP_TYPE" = "sync" ]; then
    SCRIPT="/scripts/sync-now.sh"
    echo "🔄 Setting up cron for rsync synchronization"
else
    SCRIPT="/scripts/backup-now.sh"
    echo "📸 Setting up cron for Restic snapshots"
fi

echo "⏰ Schedule: $SCHEDULE"

# Créer le cron job
echo "$SCHEDULE $SCRIPT >> /logs/backup.log 2>&1" > /etc/crontabs/root
chmod 600 /etc/crontabs/root

echo "✅ Cron job configured"
