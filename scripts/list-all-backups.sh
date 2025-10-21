#!/bin/bash
# Script pour lister tous les snapshots sur tous les peers configurés
# Usage: ./scripts/list-all-backups.sh

set -e

echo "🔍 Listing backups on all configured peers..."
echo ""

# Récupérer le nom du serveur
SERVER_NAME=$(docker exec anemone-core python3 -c "
import yaml
with open('/config/config.yaml') as f:
    config = yaml.safe_load(f)
print(config['server']['name'])
" 2>/dev/null)

echo "📦 Server: $SERVER_NAME"
echo ""

# Récupérer la liste des peers
PEERS=$(docker exec anemone-core python3 -c "
import yaml
import json
with open('/config/config.yaml') as f:
    config = yaml.safe_load(f)
peers = []
for peer in config.get('peers', []):
    peers.append({
        'name': peer['name'],
        'ip': peer['allowed_ips'].split('/')[0]
    })
print(json.dumps(peers))
" 2>/dev/null)

# Pour chaque peer
echo "$PEERS" | python3 -c "
import sys
import json
import subprocess

peers = json.load(sys.stdin)

for peer in peers:
    name = peer['name']
    ip = peer['ip']
    repo = f\"sftp:restic@{ip}:/backups/$SERVER_NAME\"

    print(f\"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\")
    print(f\"📡 Peer: {name} ({ip})\")
    print(f\"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\")

    # Exécuter restic snapshots
    cmd = f\"\"\"
export RESTIC_PASSWORD=\$(python3 /scripts/decrypt_key.py 2>/dev/null)
restic -r {repo} snapshots --compact 2>/dev/null || echo '  ❌ Unable to access repository'
\"\"\"

    result = subprocess.run(
        ['docker', 'exec', 'anemone-core', 'sh', '-c', cmd],
        capture_output=True,
        text=True
    )

    if result.returncode == 0 and result.stdout.strip():
        print(result.stdout)
    else:
        print('  ❌ No snapshots or repository not accessible')

    print()
"

echo "✅ Done"
