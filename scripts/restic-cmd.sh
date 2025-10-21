#!/bin/bash
# Script helper pour exécuter des commandes Restic avec le mot de passe chargé automatiquement
# Usage: ./scripts/restic-cmd.sh <peer-name> <restic-args>
# Exemple: ./scripts/restic-cmd.sh FR2 snapshots
# Exemple: ./scripts/restic-cmd.sh FR2 snapshots --latest 1
# Exemple: ./scripts/restic-cmd.sh FR2 ls latest

set -e

PEER_NAME="${1}"
shift

if [ -z "$PEER_NAME" ]; then
    echo "Usage: $0 <peer-name> <restic-args>"
    echo ""
    echo "Exemples:"
    echo "  $0 FR2 snapshots"
    echo "  $0 FR2 snapshots --latest 1"
    echo "  $0 FR2 stats"
    echo "  $0 FR2 ls latest"
    echo "  $0 FR2 check"
    exit 1
fi

# Récupérer le nom du serveur depuis la config
SERVER_NAME=$(docker exec anemone-core python3 -c "
import yaml
with open('/config/config.yaml') as f:
    config = yaml.safe_load(f)
print(config['server']['name'])
" 2>/dev/null)

# Récupérer l'IP du peer depuis la config
PEER_IP=$(docker exec anemone-core python3 -c "
import yaml
with open('/config/config.yaml') as f:
    config = yaml.safe_load(f)
for peer in config.get('peers', []):
    if peer['name'] == '${PEER_NAME}':
        print(peer['allowed_ips'].split('/')[0])
        break
" 2>/dev/null)

if [ -z "$PEER_IP" ]; then
    echo "❌ Peer '$PEER_NAME' not found in config"
    echo ""
    echo "Available peers:"
    docker exec anemone-core python3 -c "
import yaml
with open('/config/config.yaml') as f:
    config = yaml.safe_load(f)
for peer in config.get('peers', []):
    print(f\"  - {peer['name']} ({peer['allowed_ips']})\")
"
    exit 1
fi

# Repository path
REPO="sftp:restic@${PEER_IP}:/backups/${SERVER_NAME}"

echo "🔍 Querying repository: $REPO"
echo ""

# Exécuter la commande restic avec le mot de passe chargé
docker exec anemone-core sh -c "
  export RESTIC_PASSWORD=\$(python3 /scripts/decrypt_key.py 2>/dev/null)
  restic -r ${REPO} $*
"
