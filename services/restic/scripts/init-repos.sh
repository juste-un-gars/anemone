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

# Lire la configuration
TARGETS=$(python3 -c "
import yaml
import sys

try:
    with open('$CONFIG_PATH') as f:
        config = yaml.safe_load(f)
        targets = config.get('backup', {}).get('targets', [])
        for target in targets:
            if target.get('enabled', True):
                print(target['repository'])
except Exception as e:
    print(f'Error reading config: {e}', file=sys.stderr)
    sys.exit(1)
")

if [ -z "$TARGETS" ]; then
    echo "⚠️  No backup targets configured"
    exit 0
fi

# Initialiser chaque repository
echo "$TARGETS" | while read -r REPO; do
    if [ -z "$REPO" ]; then
        continue
    fi

    echo "  Checking repository: $REPO"

    # Vérifier si le repository existe déjà
    if restic -r "$REPO" snapshots &>/dev/null; then
        echo "  ✓ Repository already initialized: $REPO"
    else
        echo "  Initializing new repository: $REPO"
        if restic -r "$REPO" init; then
            echo "  ✓ Repository initialized: $REPO"
        else
            echo "  ⚠️  Failed to initialize: $REPO (will retry during backup)"
        fi
    fi
done

echo "✅ Repository initialization complete"
