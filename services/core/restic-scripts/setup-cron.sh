#!/bin/bash
set -e

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

# Lire le planning depuis la config
SCHEDULE=$(python3 -c "
import yaml
try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        print(config.get('backup', {}).get('schedule', '0 2 * * *'))
except:
    print('0 2 * * *')
")

echo "🔄 Setting up cron for rsync synchronization"
echo "⏰ Schedule: $SCHEDULE"

# Créer le cron job pour la synchronisation des données
echo "$SCHEDULE /scripts/sync-now.sh >> /logs/sync.log 2>&1" > /etc/crontabs/root
chmod 600 /etc/crontabs/root

echo "✅ Cron job configured"
