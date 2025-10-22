#!/bin/bash
set -e

echo "🟡 PERIODIC sync mode"

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

# Lire l'intervalle depuis la config
INTERVAL=$(python3 -c "
import yaml
try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        print(config.get('backup', {}).get('interval', 30))
except:
    print('30')
")

echo "⏱️  Sync interval: ${INTERVAL} minutes"
echo ""

# Boucle infinie avec sleep
while true; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🚀 Starting sync at $(date '+%Y-%m-%d %H:%M:%S')"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    if /scripts/sync-now.sh; then
        echo "✅ Sync completed successfully"
    else
        echo "❌ Sync failed"
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "⏳ Next sync in ${INTERVAL} minutes..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    sleep ${INTERVAL}m
done
