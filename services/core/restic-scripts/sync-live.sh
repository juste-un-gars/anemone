#!/bin/bash
set -e

echo "🔴 LIVE sync mode - Watching for file changes..."

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

# Charger la configuration
DEBOUNCE=$(python3 -c "
import yaml
try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        print(config.get('backup', {}).get('debounce', 30))
except:
    print('30')
")

BACKUP_DATA_PATH=$(python3 -c "
import yaml
try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        print(config.get('storage', {}).get('backup_data_path', '/mnt/backup'))
except:
    print('/mnt/backup')
")

echo "📂 Watching directory: $BACKUP_DATA_PATH"
echo "⏱️  Debounce delay: ${DEBOUNCE}s"

# Vérifier que le répertoire existe
if [ ! -d "$BACKUP_DATA_PATH" ]; then
    echo "⚠️  Backup directory does not exist: $BACKUP_DATA_PATH"
    echo "   Creating it..."
    mkdir -p "$BACKUP_DATA_PATH"
fi

# Timestamp du dernier sync
LAST_SYNC=0
SYNC_PENDING=0

# Fonction de sync
do_sync() {
    local now=$(date +%s)
    local elapsed=$((now - LAST_SYNC))

    if [ $elapsed -lt $DEBOUNCE ]; then
        echo "⏳ Sync too soon (${elapsed}s < ${DEBOUNCE}s), marking as pending..."
        SYNC_PENDING=1
        return
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🚀 Starting sync triggered by file changes..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if /scripts/sync-now.sh; then
        LAST_SYNC=$(date +%s)
        SYNC_PENDING=0
        echo "✅ Sync completed at $(date '+%Y-%m-%d %H:%M:%S')"
    else
        echo "❌ Sync failed at $(date '+%Y-%m-%d %H:%M:%S')"
    fi

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
}

# Fonction de vérification des syncs en attente
check_pending() {
    if [ $SYNC_PENDING -eq 1 ]; then
        local now=$(date +%s)
        local elapsed=$((now - LAST_SYNC))

        if [ $elapsed -ge $DEBOUNCE ]; then
            echo "⏰ Debounce delay passed, executing pending sync..."
            do_sync
        fi
    fi
}

# Sync initial au démarrage
echo "🔄 Performing initial sync..."
do_sync

echo ""
echo "👁️  Now watching for file changes..."
echo "   (Press Ctrl+C to stop)"
echo ""

# Surveiller les changements en arrière-plan
inotifywait -m -r -e modify,create,delete,move "$BACKUP_DATA_PATH" --exclude '(\.tmp$|\.swp$|\.part$|~$|\.trash)' | while read -r directory event filename; do
    echo "📝 Detected: $event $directory$filename"
    do_sync
done &

INOTIFY_PID=$!

# Vérifier périodiquement les syncs en attente
while true; do
    sleep 5
    check_pending

    # Vérifier que inotifywait tourne toujours
    if ! kill -0 $INOTIFY_PID 2>/dev/null; then
        echo "❌ inotifywait process died, restarting..."
        inotifywait -m -r -e modify,create,delete,move "$BACKUP_DATA_PATH" --exclude '(\.tmp$|\.swp$|\.part$|~$|\.trash)' | while read -r directory event filename; do
            echo "📝 Detected: $event $directory$filename"
            do_sync
        done &
        INOTIFY_PID=$!
    fi
done
