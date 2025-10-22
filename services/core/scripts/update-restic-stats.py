#!/usr/bin/env python3
"""
Script pour mettre à jour les statistiques Restic dans un fichier JSON
Ce fichier est lu par l'API web pour afficher l'état des backups
"""

import os
import sys
import json
import yaml
import subprocess
from datetime import datetime, timezone

STATS_FILE = "/var/stats/restic-status.json"
CONFIG_FILE = "/config/config.yaml"

def get_restic_password():
    """Récupère le mot de passe Restic déchiffré"""
    try:
        result = subprocess.run(
            ["python3", "/scripts/decrypt_key.py"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except Exception as e:
        print(f"Failed to get Restic password: {e}", file=sys.stderr)
    return None


def query_target_snapshots(target_name, repo_url, restic_password):
    """
    Interroge un repository de backup pour récupérer ses snapshots
    Retourne le dernier snapshot avec métadonnées
    """

    try:
        # Récupérer tous les snapshots en JSON
        result = subprocess.run(
            ["restic", "-r", repo_url, "snapshots", "--json"],
            capture_output=True,
            text=True,
            timeout=15,
            env={**os.environ, "RESTIC_PASSWORD": restic_password}
        )

        if result.returncode != 0:
            return {
                "status": "error",
                "message": "Failed to query repository",
                "error": result.stderr
            }

        snapshots = json.loads(result.stdout)

        if not snapshots:
            return {
                "status": "no_snapshots",
                "message": "No snapshots found"
            }

        # Trier par date (le plus récent en premier) et prendre le premier
        snapshots.sort(key=lambda s: s['time'], reverse=True)
        snapshot = snapshots[0]

        # Calculer l'âge du snapshot
        snapshot_time = datetime.fromisoformat(snapshot['time'].replace('Z', '+00:00'))
        now = datetime.now(timezone.utc)
        age_seconds = (now - snapshot_time).total_seconds()
        age_hours = age_seconds / 3600

        # Déterminer le statut
        if age_hours < 25:
            status = "ok"
        elif age_hours < 48:
            status = "warning"
        else:
            status = "error"

        return {
            "status": status,
            "last_snapshot": {
                "id": snapshot.get('short_id', 'unknown'),
                "time": snapshot['time'],
                "time_formatted": snapshot_time.strftime("%Y-%m-%d %H:%M:%S"),
                "age_hours": round(age_hours, 1),
                "hostname": snapshot.get('hostname', 'unknown'),
                "paths": snapshot.get('paths', [])
            }
        }

    except subprocess.TimeoutExpired:
        return {
            "status": "timeout",
            "message": "Query timeout (peer unreachable?)"
        }
    except json.JSONDecodeError:
        return {
            "status": "error",
            "message": "Invalid JSON response from Restic"
        }
    except Exception as e:
        return {
            "status": "error",
            "message": str(e)
        }


def update_stats():
    """Met à jour le fichier de statistiques"""
    try:
        # Charger la configuration
        with open(CONFIG_FILE) as f:
            config = yaml.safe_load(f)

        server_name = config.get('node', {}).get('name', 'unknown')
        targets = config.get('backup', {}).get('targets', [])

        # Récupérer le mot de passe Restic
        restic_password = get_restic_password()
        if not restic_password:
            print("ERROR: Could not get Restic password", file=sys.stderr)
            return False

        # Construire les stats
        stats = {
            "last_update": datetime.now(timezone.utc).isoformat(),
            "server_name": server_name,
            "targets": []
        }

        # Pour chaque target de backup, récupérer les stats
        for target in targets:
            target_name = target.get('name', 'unknown')

            # Ne traiter que les targets activés
            if not target.get('enabled', True):
                stats["targets"].append({
                    "name": target_name,
                    "status": "disabled",
                    "message": "Target disabled in config"
                })
                continue

            # Construire l'URL du repository (même logique que backup-now.sh)
            host = target.get('host')
            port = target.get('port', 22)
            user = target.get('user', 'restic')
            path = target.get('path', '/backups')

            if not host:
                stats["targets"].append({
                    "name": target_name,
                    "status": "error",
                    "message": "No host configured"
                })
                continue

            repo_url = f"sftp:{user}@{host}:{path}"

            target_stats = query_target_snapshots(target_name, repo_url, restic_password)
            target_stats["name"] = target_name
            target_stats["host"] = host

            stats["targets"].append(target_stats)

        # Déterminer le statut global
        active_targets = [t for t in stats["targets"] if t["status"] not in ["disabled"]]

        if not active_targets:
            stats["global_status"] = "no_targets"
        elif all(t["status"] == "ok" for t in active_targets):
            stats["global_status"] = "ok"
        elif any(t["status"] in ["error", "timeout"] for t in active_targets):
            stats["global_status"] = "error"
        elif any(t["status"] == "warning" for t in active_targets):
            stats["global_status"] = "warning"
        else:
            stats["global_status"] = "partial"

        # Créer le répertoire si nécessaire
        os.makedirs(os.path.dirname(STATS_FILE), exist_ok=True)

        # Écrire le fichier JSON
        with open(STATS_FILE, 'w') as f:
            json.dump(stats, f, indent=2)

        print(f"✅ Stats updated: {stats['global_status']}")
        return True

    except Exception as e:
        print(f"ERROR updating stats: {e}", file=sys.stderr)
        return False


if __name__ == "__main__":
    success = update_stats()
    sys.exit(0 if success else 1)
