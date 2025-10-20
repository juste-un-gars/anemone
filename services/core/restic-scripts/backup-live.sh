#!/bin/bash
set -e

echo "🔴 LIVE backup mode - Watching for file changes..."

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

# Timestamp du dernier backup
LAST_BACKUP=0
BACKUP_PENDING=0

# Fonction de backup
do_backup() {
    local now=$(date +%s)
    local elapsed=$((now - LAST_BACKUP))

    if [ $elapsed -lt $DEBOUNCE ]; then
        echo "⏳ Backup too soon (${elapsed}s < ${DEBOUNCE}s), marking as pending..."
        BACKUP_PENDING=1
        return
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🚀 Starting backup triggered by file changes..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if /scripts/backup-now.sh; then
        LAST_BACKUP=$(date +%s)
        BACKUP_PENDING=0
        echo "✅ Backup completed at $(date '+%Y-%m-%d %H:%M:%S')"
    else
        echo "❌ Backup failed at $(date '+%Y-%m-%d %H:%M:%S')"
    fi

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
}

# Fonction de vérification des backups en attente
check_pending() {
    if [ $BACKUP_PENDING -eq 1 ]; then
        local now=$(date +%s)
        local elapsed=$((now - LAST_BACKUP))

        if [ $elapsed -ge $DEBOUNCE ]; then
            echo "⏰ Debounce delay passed, executing pending backup..."
            do_backup
        fi
    fi
}

# Backup initial au démarrage
echo "🔄 Performing initial backup..."
do_backup

echo ""
echo "👁️  Now watching for file changes..."
echo "   (Press Ctrl+C to stop)"
echo ""

# Surveiller les changements en arrière-plan
inotifywait -m -r -e modify,create,delete,move "$BACKUP_DATA_PATH" --exclude '(\.tmp$|\.swp$|\.part$|~$)' | while read -r directory event filename; do
    echo "📝 Detected: $event $directory$filename"
    do_backup
done &

INOTIFY_PID=$!

# Vérifier périodiquement les backups en attente
while true; do
    sleep 5
    check_pending

    # Vérifier que inotifywait tourne toujours
    if ! kill -0 $INOTIFY_PID 2>/dev/null; then
        echo "❌ inotifywait process died, restarting..."
        inotifywait -m -r -e modify,create,delete,move "$BACKUP_DATA_PATH" --exclude '(\.tmp$|\.swp$|\.part$|~$)' | while read -r directory event filename; do
            echo "📝 Detected: $event $directory$filename"
            do_backup
        done &
        INOTIFY_PID=$!
    fi
done
