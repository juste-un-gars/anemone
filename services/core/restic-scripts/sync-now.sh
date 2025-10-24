#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

set -e

echo "[$(date)] 🔄 Encrypted synchronization starting..."

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

# Vérifier que rclone est installé
if ! command -v rclone &> /dev/null; then
    echo "❌ rclone is not installed"
    exit 1
fi

# S'assurer que le répertoire source se termine par /
if [ ! -d "$BACKUP_SOURCE" ]; then
    echo "⚠️  Backup source does not exist: $BACKUP_SOURCE"
    echo "   Creating it..."
    mkdir -p "$BACKUP_SOURCE"
fi

# Vérifier que la configuration rclone existe
if [ ! -f /root/.config/rclone/rclone.conf ]; then
    echo "❌ Rclone configuration not found"
    echo "   Run: python3 /scripts/configure-rclone.py"
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

success_count = 0
failed_targets = []

for target in targets:
    name = target.get('name', 'unknown')

    # Remote name: remove '-backup' suffix and add '-crypt'
    remote_base = name.replace('-backup', '')
    crypt_remote = f'{remote_base}-crypt'

    print(f'\\n📤 Syncing to: {name} (encrypted)')

    # Options rclone sync:
    # --delete-during : supprime les fichiers manquants pendant le transfert (miroir)
    # --progress : affiche la progression
    # --stats : affiche les statistiques
    # --stats-one-line : stats sur une ligne
    # --exclude : exclut les patterns
    # --checksum : vérifie les checksums pour détecter les changements
    # --retries : nombre de tentatives en cas d'erreur
    rclone_cmd = [
        'rclone',
        'sync',
        backup_source,
        f'{crypt_remote}:',
        '--delete-during',
        '--progress',
        '--stats', '5s',
        '--stats-one-line',
        '--exclude', '.trash/**',
        '--exclude', '*.tmp',
        '--exclude', '*.swp',
        '--exclude', '*.part',
        '--exclude', '~*',
        '--checksum',
        '--retries', '3',
        '--low-level-retries', '10'
    ]

    try:
        result = subprocess.run(
            rclone_cmd,
            capture_output=True,
            text=True,
            timeout=3600  # 1 heure timeout
        )

        if result.returncode == 0:
            print(f'  ✅ Sync successful to {name}')

            # Afficher les statistiques
            for line in result.stderr.split('\\n'):
                if line.strip() and ('Transferred' in line or 'Checks' in line or 'Deleted' in line or 'Elapsed' in line):
                    print(f'     {line.strip()}')

            success_count += 1
        else:
            print(f'  ❌ Sync failed to {name}')
            print(f'     Error: {result.stderr}', file=sys.stderr)
            failed_targets.append(name)

    except subprocess.TimeoutExpired:
        print(f'  ⏱️  Sync timeout to {name} (>1h)', file=sys.stderr)
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
    echo "[$(date)] ✅ Encrypted sync completed successfully"
else
    echo "[$(date)] ❌ Encrypted sync completed with errors"
fi

exit $EXIT_CODE
