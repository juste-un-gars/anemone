#!/bin/bash
set -e

echo "🌐 Starting WireGuard VPN..."

# Vérifier que la configuration existe
if [ ! -f /config/wireguard/wg0.conf ]; then
    echo "❌ WireGuard configuration not found: /config/wireguard/wg0.conf"
    echo "   Please run the initialization script first"
    exit 1
fi

# Copier la configuration
cp /config/wireguard/wg0.conf /etc/wireguard/wg0.conf
chmod 600 /etc/wireguard/wg0.conf

# Activer le forwarding IP
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv4.conf.all.src_valid_mark=1

# Démarrer WireGuard
wg-quick up wg0

echo "✅ WireGuard started"

# Afficher le statut
wg show

# Garder le processus actif et surveiller l'interface
while true; do
    if ! ip link show wg0 >/dev/null 2>&1; then
        echo "❌ WireGuard interface down, restarting..."
        wg-quick down wg0 2>/dev/null || true
        sleep 2
        wg-quick up wg0
    fi
    sleep 30
done
