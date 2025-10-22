#!/bin/bash
set -e

echo "[$(date)] 🔄 Rsync synchronization starting..."

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}
BACKUP_SOURCE=${BACKUP_SOURCE:-/mnt/backup}

# Lire les targets depuis config.yaml
TARGETS_JSON=$(python3 -c "
import yaml
import json
import sys

try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        targets = config.get('backup', {}).get('targets', [])
        enabled_targets = [t for t in targets if t.get('enabled', True)]
        print(json.dumps(enabled_targets))
except Exception as e:
    print(f'Error reading config: {e}', file=sys.stderr)
    sys.exit(1)
")

# Vérifier qu'il y a des targets
TARGET_COUNT=$(echo "$TARGETS_JSON" | python3 -c "import sys, json; print(len(json.load(sys.stdin)))")

if [ "$TARGET_COUNT" -eq 0 ]; then
    echo "⚠️  No enabled sync targets configured"
    exit 0
fi

echo "📦 Found $TARGET_COUNT sync target(s)"

# Vérifier que rsync est installé
if ! command -v rsync &> /dev/null; then
    echo "❌ rsync is not installed"
    exit 1
fi

# Pour chaque target, exécuter la synchronisation
echo "$TARGETS_JSON" | python3 -c "
import json
import sys
import subprocess
import os

targets = json.load(sys.stdin)
backup_source = os.getenv('BACKUP_SOURCE', '/mnt/backup')

# S'assurer que le chemin source se termine par / pour rsync
if not backup_source.endswith('/'):
    backup_source += '/'

success_count = 0
failed_targets = []

for target in targets:
    name = target.get('name', 'unknown')
    host = target.get('host')
    port = target.get('port', 22)
    user = target.get('user', 'restic')
    path = target.get('path', '/backups')

    if not host:
        print(f'⚠️  Target {name}: no host configured', file=sys.stderr)
        failed_targets.append(name)
        continue

    # Construire la destination rsync
    # Format: user@host:path/server_name/
    # Le path doit correspondre au nom du serveur pour la structure
    server_name = os.getenv('HOSTNAME', 'anemone')
    dest = f'{user}@{host}:{path}'

    print(f'\\n📤 Syncing to: {name} ({dest})')

    # Options rsync:
    # -a : archive mode (recursive, preserve permissions, times, etc.)
    # -v : verbose
    # -z : compress during transfer
    # --delete : delete files on destination that don't exist on source (MIRROR)
    # --exclude : exclude patterns
    # -e : specify SSH with custom port
    rsync_cmd = [
        'rsync',
        '-avz',
        '--delete',
        '--exclude=.trash/',        # Ne pas synchroniser la corbeille
        '--exclude=*.tmp',
        '--exclude=*.swp',
        '--exclude=*.part',
        '--exclude=~*',
        '-e', f'ssh -p {port} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null',
        backup_source,
        dest
    ]

    try:
        result = subprocess.run(
            rsync_cmd,
            capture_output=True,
            text=True,
            timeout=600  # 10 minutes timeout
        )

        if result.returncode == 0:
            print(f'  ✅ Sync successful to {name}')

            # Afficher les statistiques de rsync
            for line in result.stdout.split('\\n'):
                if 'sent' in line or 'total size' in line or 'speedup' in line:
                    print(f'     {line.strip()}')

            success_count += 1
        else:
            print(f'  ❌ Sync failed to {name}')
            print(f'     Error: {result.stderr}', file=sys.stderr)
            failed_targets.append(name)

    except subprocess.TimeoutExpired:
        print(f'  ⏱️  Sync timeout to {name}', file=sys.stderr)
        failed_targets.append(name)
    except Exception as e:
        print(f'  ❌ Exception during sync to {name}: {e}', file=sys.stderr)
        failed_targets.append(name)

# Résumé
print(f'\\n📊 Sync summary: {success_count}/{len(targets)} successful')
if failed_targets:
    print(f'❌ Failed targets: {\", \".join(failed_targets)}', file=sys.stderr)
    sys.exit(1)
"

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "[$(date)] ✅ Sync completed successfully"
else
    echo "[$(date)] ❌ Sync completed with errors"
fi

exit $EXIT_CODE
