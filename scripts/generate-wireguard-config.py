#!/usr/bin/env python3
"""
Script pour générer /config/wireguard/wg0.conf depuis config.yaml
"""

import yaml
import sys
from pathlib import Path

def generate_wg_config(config_path="/config/config.yaml", output_path="/config/wireguard/wg0.conf"):
    """
    Génère le fichier wg0.conf pour WireGuard

    Args:
        config_path: Chemin vers config.yaml
        output_path: Chemin de sortie pour wg0.conf
    """
    # Vérifier que config.yaml existe
    config_path = Path(config_path)
    if not config_path.exists():
        print(f"ERROR: {config_path} does not exist", file=sys.stderr)
        sys.exit(1)

    # Charger la configuration
    with open(config_path, 'r') as f:
        config = yaml.safe_load(f)

    if not config:
        print("ERROR: config.yaml is empty or invalid", file=sys.stderr)
        sys.exit(1)

    wireguard_config = config.get('wireguard', {})
    peers = config.get('peers', [])

    # Lire la clé privée (chercher dans les chemins possibles)
    private_key_path = None
    possible_paths = [
        Path("config/wireguard/private.key"),
        Path("/config/wireguard/private.key"),
        Path("config/wg_confs/privatekey"),  # Parfois stocké ici aussi
        Path("/config/wg_confs/privatekey")
    ]

    for path in possible_paths:
        if path.exists():
            private_key_path = path
            break

    if not private_key_path:
        print(f"ERROR: private.key not found in config/wireguard/ or /config/wireguard/", file=sys.stderr)
        sys.exit(1)

    private_key = private_key_path.read_text().strip()

    # Générer la section [Interface]
    address = wireguard_config.get('address', '10.8.0.1/24')
    port = wireguard_config.get('port', 51820)

    config_lines = [
        "[Interface]",
        f"Address = {address}",
        f"ListenPort = {port}",
        f"PrivateKey = {private_key}",
        "MTU = 1420",
        ""
    ]

    # Ajouter chaque pair
    for peer in peers:
        config_lines.append("[Peer]")
        config_lines.append(f"# {peer.get('name', 'Unknown')}")
        config_lines.append(f"PublicKey = {peer.get('public_key', '')}")

        if peer.get('endpoint'):
            config_lines.append(f"Endpoint = {peer['endpoint']}")

        config_lines.append(f"AllowedIPs = {peer.get('allowed_ips', '')}")

        keepalive = peer.get('persistent_keepalive', 25)
        config_lines.append(f"PersistentKeepalive = {keepalive}")
        config_lines.append("")

    # Écrire le fichier
    output_path = Path(output_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    output_path.write_text("\n".join(config_lines))
    output_path.chmod(0o600)

    print(f"✓ WireGuard configuration generated: {output_path}")
    print(f"  Local IP: {address}")
    print(f"  Port: {port}")
    print(f"  Peers configured: {len(peers)}")

if __name__ == "__main__":
    config_file = sys.argv[1] if len(sys.argv) > 1 else "/config/config.yaml"
    # Architecture v2.0 : WireGuard natif (Alpine) lit depuis /config/wireguard/
    output_file = sys.argv[2] if len(sys.argv) > 2 else "/config/wireguard/wg0.conf"

    try:
        generate_wg_config(config_file, output_file)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)
