#!/usr/bin/env python3
"""
Script de découverte automatique des backups de configuration sur les peers
Usage: python3 discover-backups.py [--json]
"""

import sys
import yaml
import subprocess
import re
from pathlib import Path
from datetime import datetime
import json

def log(message, error=False):
    """Affiche un message avec timestamp"""
    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    prefix = "❌" if error else "🔍"
    output = sys.stderr if error else sys.stdout
    print(f"[{timestamp}] {prefix} {message}", file=output)

def get_server_name(config_file="/config/config.yaml"):
    """Récupère le nom du serveur depuis la configuration"""
    try:
        with open(config_file, 'r') as f:
            config = yaml.safe_load(f)
            return config.get('server', {}).get('name', 'unknown')
    except Exception as e:
        log(f"Erreur lecture config: {e}", error=True)
        return 'unknown'

def get_peers(config_file="/config/config.yaml"):
    """Récupère la liste des peers actifs depuis la configuration"""
    try:
        with open(config_file, 'r') as f:
            config = yaml.safe_load(f)
            peers = []
            for peer in config.get('peers', []):
                if peer.get('enabled', True):
                    peers.append({
                        'name': peer.get('name', 'unknown'),
                        'vpn_ip': peer.get('vpn_ip', ''),
                        'endpoint': peer.get('endpoint', '')
                    })
            return peers
    except Exception as e:
        log(f"Erreur lecture peers: {e}", error=True)
        return []

def list_remote_backups(peer_ip, server_name, ssh_key="/root/.ssh/id_rsa"):
    """Liste les backups disponibles sur un peer via SFTP"""
    remote_path = f"/config-backups/{server_name}"

    # Créer la commande SFTP pour lister les fichiers
    sftp_commands = f"ls -l {remote_path}/*.enc"

    try:
        result = subprocess.run(
            [
                "ssh",
                "-o", "StrictHostKeyChecking=no",
                "-o", "ConnectTimeout=10",
                "-i", ssh_key,
                f"restic@{peer_ip}",
                sftp_commands
            ],
            capture_output=True,
            text=True,
            timeout=15
        )

        if result.returncode != 0:
            return []

        backups = []
        for line in result.stdout.strip().split('\n'):
            # Parser la sortie ls -l pour extraire nom et taille
            parts = line.split()
            if len(parts) >= 9:
                filename = parts[-1]
                size = int(parts[4]) if parts[4].isdigit() else 0

                # Extraire la date du nom de fichier (format: anemone-backup-HOSTNAME-YYYYMMDD-HHMMSS.enc)
                match = re.search(r'(\d{8})-(\d{6})\.enc$', filename)
                if match:
                    date_str = match.group(1)
                    time_str = match.group(2)
                    timestamp_str = f"{date_str[:4]}-{date_str[4:6]}-{date_str[6:8]} {time_str[:2]}:{time_str[2:4]}:{time_str[4:6]}"

                    backups.append({
                        'filename': filename,
                        'path': f"{remote_path}/{filename}",
                        'size': size,
                        'timestamp': timestamp_str,
                        'date': date_str,
                        'time': time_str
                    })

        return backups

    except subprocess.TimeoutExpired:
        log(f"Timeout lors de la connexion à {peer_ip}", error=True)
        return []
    except Exception as e:
        log(f"Erreur lors de la liste des backups sur {peer_ip}: {e}", error=True)
        return []

def download_backup(peer_ip, remote_path, local_path, ssh_key="/root/.ssh/id_rsa"):
    """Télécharge un backup depuis un peer via SFTP"""
    try:
        result = subprocess.run(
            [
                "scp",
                "-o", "StrictHostKeyChecking=no",
                "-o", "ConnectTimeout=10",
                "-i", ssh_key,
                f"restic@{peer_ip}:{remote_path}",
                local_path
            ],
            capture_output=True,
            text=True,
            timeout=60
        )

        return result.returncode == 0

    except Exception as e:
        log(f"Erreur lors du téléchargement: {e}", error=True)
        return False

def discover_backups(output_json=False):
    """Découvre tous les backups disponibles sur les peers"""
    server_name = get_server_name()
    peers = get_peers()

    if not output_json:
        log(f"Serveur: {server_name}")
        log(f"Recherche de backups sur {len(peers)} peer(s)...")
        log("")

    all_backups = []

    for peer in peers:
        if not peer['vpn_ip']:
            continue

        if not output_json:
            log(f"→ Connexion à {peer['name']} ({peer['vpn_ip']})...")

        backups = list_remote_backups(peer['vpn_ip'], server_name)

        if backups:
            for backup in backups:
                backup['peer_name'] = peer['name']
                backup['peer_ip'] = peer['vpn_ip']
                all_backups.append(backup)

            if not output_json:
                log(f"  ✓ {len(backups)} backup(s) trouvé(s)")
        else:
            if not output_json:
                log(f"  ✗ Aucun backup trouvé ou connexion impossible")

    if output_json:
        print(json.dumps({
            'server': server_name,
            'backups': all_backups,
            'total': len(all_backups)
        }, indent=2))
    else:
        log("")
        log(f"📊 Total: {len(all_backups)} backup(s) disponible(s)")

        if all_backups:
            log("")
            log("Backups trouvés:")
            # Trier par date (du plus récent au plus ancien)
            all_backups.sort(key=lambda x: x['timestamp'], reverse=True)

            for i, backup in enumerate(all_backups, 1):
                size_mb = backup['size'] / (1024 * 1024)
                log(f"  {i}. {backup['filename']}")
                log(f"     Peer: {backup['peer_name']} ({backup['peer_ip']})")
                log(f"     Date: {backup['timestamp']}")
                log(f"     Taille: {size_mb:.2f} MB")
                log("")

    return all_backups

def main():
    output_json = '--json' in sys.argv

    try:
        backups = discover_backups(output_json)
        sys.exit(0 if backups else 1)
    except KeyboardInterrupt:
        if not output_json:
            log("\nInterrompu par l'utilisateur")
        sys.exit(1)
    except Exception as e:
        log(f"Erreur: {e}", error=True)
        sys.exit(1)

if __name__ == "__main__":
    main()
