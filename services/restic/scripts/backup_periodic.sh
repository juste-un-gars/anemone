#!/bin/bash

# Mode PERIODIC : backup toutes les X minutes

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

# Extraire l'intervalle depuis le config
INTERVAL=$(python3 -c "
import yaml
with open('$CONFIG_PATH') as f:
    config = yaml.safe_load(f)
    print(config.get('backup', {}).get('interval', 30))
")

INTERVAL_SECONDS=$((INTERVAL * 60))

echo "🟡 Periodic backup mode active"
echo "   Interval: ${INTERVAL} minutes (${INTERVAL_SECONDS}s)"
echo ""

# Faire un premier backup au démarrage
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 🚀 Initial backup..."
/scripts/backup-now.sh

# Boucle infinie
while true; do
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 🔄 Starting backup..."
    /scripts/backup-now.sh
done-%m-%d %H:%M:%S')] ⏱️  Waiting ${INTERVAL} minutes..."
    sleep $INTERVAL_SECONDS
    
    echo "[$(date '+%Y