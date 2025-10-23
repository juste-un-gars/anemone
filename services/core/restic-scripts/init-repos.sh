#!/bin/bash

echo "🔧 Initializing Restic repositories..."

# Attendre que le réseau VPN soit prêt
echo "  Waiting for VPN network to be ready..."
sleep 5

# Vérifier que wg0 existe
if ip addr show wg0 &>/dev/null; then
    echo "  ✓ VPN interface wg0 is up"
else
    echo "  ⚠️  VPN interface wg0 not found (network may not be ready yet)"
fi

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}

# Lire et initialiser les repositories
python3 << 'PYTHON_SCRIPT'
import yaml
import subprocess
import sys
import os

config_path = os.getenv('CONFIG_PATH', '/config/config.yaml')
restic_password = os.getenv('RESTIC_PASSWORD')

if not restic_password:
    print('⚠️  RESTIC_PASSWORD not set, skipping repository initialization')
    sys.exit(0)

try:
    with open(config_path) as f:
        config = yaml.safe_load(f)
        targets = config.get('backup', {}).get('targets', [])
        enabled_targets = [t for t in targets if t.get('enabled', True)]

    if not enabled_targets:
        print('⚠️  No backup targets configured')
        sys.exit(0)

    print(f'  Found {len(enabled_targets)} target(s) to initialize')

    for target in enabled_targets:
        name = target.get('name', 'unknown')
        host = target.get('host')
        port = target.get('port', 22222)
        user = target.get('user', 'restic')
        path = target.get('path', '/backups')

        # Construire l'URL du repository
        # Format SFTP pour Restic : sftp:user@host:/path
        # Le port 22222 est configuré dans /root/.ssh/config (Host *)
        # Via VPN, on utilise le port 22222 (port interne du conteneur)
        repo_url = f'sftp:{user}@{host}:{path}'

        print(f'\\n  Checking repository: {name}')
        print(f'    URL: {repo_url}')

        # Vérifier si le repository existe déjà
        check_cmd = ['restic', '-r', repo_url, 'snapshots']

        try:
            result = subprocess.run(
                check_cmd,
                env={**os.environ, 'RESTIC_PASSWORD': restic_password},
                capture_output=True,
                text=True,
                timeout=30
            )

            if result.returncode == 0:
                print(f'    ✓ Repository already initialized')
            else:
                # Repository n'existe pas, l'initialiser
                print(f'    Initializing new repository...')
                init_cmd = ['restic', '-r', repo_url, 'init']

                result = subprocess.run(
                    init_cmd,
                    env={**os.environ, 'RESTIC_PASSWORD': restic_password},
                    capture_output=True,
                    text=True,
                    timeout=30
                )

                if result.returncode == 0:
                    print(f'    ✓ Repository initialized successfully')
                else:
                    print(f'    ⚠️  Failed to initialize: {result.stderr}')
                    print(f'    (will retry during backup)')

        except subprocess.TimeoutExpired:
            print(f'    ⏱️  Connection timeout (peer may not be reachable yet)')
        except Exception as e:
            print(f'    ⚠️  Error: {e}')

    print('\\n✅ Repository initialization complete')

except Exception as e:
    print(f'❌ Error reading config: {e}', file=sys.stderr)
    sys.exit(1)

PYTHON_SCRIPT
