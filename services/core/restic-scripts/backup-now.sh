#!/bin/bash
set -e

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}
BACKUP_SOURCE=${BACKUP_SOURCE:-/mnt/backup}

# Charger le mot de passe Restic s'il n'est pas déjà défini
if [ -z "$RESTIC_PASSWORD" ]; then
    export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py 2>/dev/null)
    if [ -z "$RESTIC_PASSWORD" ]; then
        echo "❌ Failed to decrypt Restic password" >&2
        exit 1
    fi
fi

echo "[$(date)] 🔄 Backup starting..."

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
    echo "⚠️  No enabled backup targets configured"
    exit 0
fi

echo "📦 Found $TARGET_COUNT backup target(s)"

# Pour chaque target, exécuter le backup
echo "$TARGETS_JSON" | python3 -c "
import json
import sys
import subprocess
import os

targets = json.load(sys.stdin)
backup_source = os.getenv('BACKUP_SOURCE', '/mnt/backup')
restic_password = os.getenv('RESTIC_PASSWORD')

if not restic_password:
    print('❌ RESTIC_PASSWORD not set', file=sys.stderr)
    sys.exit(1)

success_count = 0
failed_targets = []

for target in targets:
    name = target.get('name', 'unknown')
    host = target.get('host')
    port = target.get('port', 22)
    user = target.get('user', 'restic')
    path = target.get('path', '/backups')

    # Construire l'URL du repository Restic
    # Format SFTP pour Restic : sftp:user@host:/path (port 22 par défaut)
    # Note : Restic ne supporte pas le port dans l'URL SFTP
    # Pour un port non-standard, il faudrait configurer SSH
    # Via VPN, on utilise toujours le port 22 (port interne du conteneur)
    repo_url = f'sftp:{user}@{host}:{path}'

    print(f'\\n📤 Backing up to: {name} ({repo_url})')

    # Préparer la commande restic
    cmd = [
        'restic',
        '-r', repo_url,
        'backup',
        backup_source,
        '--host', os.getenv('HOSTNAME', 'anemone'),
        '--tag', 'auto'
    ]

    # Ajouter les exclusions depuis la config
    # TODO: Lire les exclusions depuis config.yaml

    try:
        result = subprocess.run(
            cmd,
            env={**os.environ, 'RESTIC_PASSWORD': restic_password},
            capture_output=True,
            text=True,
            timeout=300  # 5 minutes timeout
        )

        if result.returncode == 0:
            print(f'  ✅ Backup successful to {name}')
            success_count += 1
        else:
            print(f'  ❌ Backup failed to {name}')
            print(f'     Error: {result.stderr}', file=sys.stderr)
            failed_targets.append(name)

    except subprocess.TimeoutExpired:
        print(f'  ⏱️  Backup timeout to {name}', file=sys.stderr)
        failed_targets.append(name)
    except Exception as e:
        print(f'  ❌ Exception during backup to {name}: {e}', file=sys.stderr)
        failed_targets.append(name)

# Résumé
print(f'\\n📊 Backup summary: {success_count}/{len(targets)} successful')
if failed_targets:
    print(f'❌ Failed targets: {\", \".join(failed_targets)}', file=sys.stderr)
    sys.exit(1)
"

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "[$(date)] ✅ Backup completed successfully"
else
    echo "[$(date)] ❌ Backup completed with errors"
fi

exit $EXIT_CODE
