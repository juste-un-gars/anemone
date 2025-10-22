"""
Gestionnaire de quotas disque pour Anemone
Surveille et applique les limites d'espace disque par pair
"""

import os
import yaml
from pathlib import Path
from typing import Dict, List, Tuple


class QuotaManager:
    def __init__(self, config_path: str = "/config/config.yaml"):
        self.config_path = config_path
        self.config = self._load_config()

    def _load_config(self) -> dict:
        """Charge la configuration depuis le fichier YAML"""
        try:
            with open(self.config_path, 'r') as f:
                return yaml.safe_load(f)
        except Exception as e:
            print(f"Error loading config: {e}")
            return {}

    def _parse_size(self, size_str: str) -> int:
        """
        Convertit une taille en format humain (ex: "10GB") en bytes

        Args:
            size_str: Taille au format "5GB", "100MB", etc. ou "0" pour illimité

        Returns:
            Taille en bytes, ou 0 pour illimité
        """
        if not size_str or size_str == "0":
            return 0  # 0 = illimité

        size_str = size_str.strip().upper()
        units = {
            'KB': 1024,
            'MB': 1024 ** 2,
            'GB': 1024 ** 3,
            'TB': 1024 ** 4,
        }

        for unit, multiplier in units.items():
            if size_str.endswith(unit):
                try:
                    value = float(size_str[:-2])
                    return int(value * multiplier)
                except ValueError:
                    return 0

        # Si pas d'unité reconnue, considérer comme bytes
        try:
            return int(size_str)
        except ValueError:
            return 0

    def _format_size(self, size_bytes: int) -> str:
        """
        Convertit une taille en bytes en format humain

        Args:
            size_bytes: Taille en bytes

        Returns:
            Taille au format "10.5 GB"
        """
        if size_bytes == 0:
            return "0 B"

        units = ['B', 'KB', 'MB', 'GB', 'TB']
        unit_index = 0
        size = float(size_bytes)

        while size >= 1024 and unit_index < len(units) - 1:
            size /= 1024
            unit_index += 1

        return f"{size:.2f} {units[unit_index]}"

    def _get_directory_size(self, path: Path) -> int:
        """
        Calcule la taille totale d'un répertoire

        Args:
            path: Chemin du répertoire

        Returns:
            Taille en bytes
        """
        total_size = 0
        try:
            for entry in path.rglob('*'):
                if entry.is_file():
                    total_size += entry.stat().st_size
        except Exception as e:
            print(f"Error calculating size for {path}: {e}")

        return total_size

    def check_peer_quota(self, peer_name: str) -> Tuple[bool, Dict]:
        """
        Vérifie si un pair respecte son quota

        Args:
            peer_name: Nom du pair

        Returns:
            Tuple (within_quota, info_dict)
            - within_quota: True si dans les limites, False si dépassé
            - info_dict: Détails sur l'utilisation
        """
        # Récupérer le quota configuré
        max_size_str = self.config.get('restic_server', {}).get('max_size_per_peer', '10GB')
        max_size_bytes = self._parse_size(max_size_str)

        # Récupérer le chemin de stockage des backups reçus
        # Les backups sont reçus dans /home/restic/backups/{server_name}/
        backup_receive_path = '/home/restic/backups'
        peer_dir = Path(backup_receive_path) / peer_name

        # Vérifier si le répertoire existe
        if not peer_dir.exists():
            return True, {
                'peer_name': peer_name,
                'used_bytes': 0,
                'used_formatted': '0 B',
                'quota_bytes': max_size_bytes,
                'quota_formatted': max_size_str,
                'percentage': 0,
                'within_quota': True,
                'unlimited': max_size_bytes == 0
            }

        # Calculer la taille utilisée
        used_bytes = self._get_directory_size(peer_dir)

        # Vérifier le quota
        unlimited = max_size_bytes == 0
        within_quota = unlimited or used_bytes <= max_size_bytes
        percentage = 0 if unlimited or max_size_bytes == 0 else (used_bytes / max_size_bytes) * 100

        return within_quota, {
            'peer_name': peer_name,
            'used_bytes': used_bytes,
            'used_formatted': self._format_size(used_bytes),
            'quota_bytes': max_size_bytes,
            'quota_formatted': max_size_str if not unlimited else 'Unlimited',
            'percentage': round(percentage, 2),
            'within_quota': within_quota,
            'unlimited': unlimited
        }

    def check_all_quotas(self) -> List[Dict]:
        """
        Vérifie les quotas de tous les pairs

        Returns:
            Liste des informations de quota pour chaque pair
        """
        peers = self.config.get('peers', [])
        quota_info = []

        for peer in peers:
            peer_name = peer.get('name', 'Unknown')
            within_quota, info = self.check_peer_quota(peer_name)
            quota_info.append(info)

        return quota_info

    def enforce_quota(self, peer_name: str) -> bool:
        """
        Applique le quota pour un pair qui a dépassé sa limite
        En désactivant sa clé SSH dans authorized_keys

        Args:
            peer_name: Nom du pair

        Returns:
            True si l'action a été effectuée, False sinon
        """
        within_quota, info = self.check_peer_quota(peer_name)

        if within_quota:
            return False  # Pas besoin d'appliquer le quota

        # Désactiver la clé SSH du pair dans authorized_keys
        authorized_keys_path = "/config/ssh/authorized_keys"

        try:
            with open(authorized_keys_path, 'r') as f:
                lines = f.readlines()

            # Trouver les clés du pair et les commenter
            modified = False
            new_lines = []
            for line in lines:
                # Si la ligne contient le nom du pair et n'est pas déjà commentée
                if peer_name in line and not line.strip().startswith('#'):
                    new_lines.append(f"# QUOTA EXCEEDED - {line}")
                    modified = True
                else:
                    new_lines.append(line)

            if modified:
                with open(authorized_keys_path, 'w') as f:
                    f.writelines(new_lines)
                print(f"✓ SSH access disabled for peer '{peer_name}' (quota exceeded)")
                return True

        except Exception as e:
            print(f"ERROR enforcing quota for {peer_name}: {e}")

        return False

    def restore_access(self, peer_name: str) -> bool:
        """
        Restaure l'accès SSH pour un pair (décommente sa clé)

        Args:
            peer_name: Nom du pair

        Returns:
            True si l'action a été effectuée, False sinon
        """
        authorized_keys_path = "/config/ssh/authorized_keys"

        try:
            with open(authorized_keys_path, 'r') as f:
                lines = f.readlines()

            # Décommenter les clés du pair
            modified = False
            new_lines = []
            for line in lines:
                # Si la ligne est commentée pour quota et contient le nom du pair
                if '# QUOTA EXCEEDED' in line and peer_name in line:
                    # Retirer le commentaire
                    new_lines.append(line.replace('# QUOTA EXCEEDED - ', ''))
                    modified = True
                else:
                    new_lines.append(line)

            if modified:
                with open(authorized_keys_path, 'w') as f:
                    f.writelines(new_lines)
                print(f"✓ SSH access restored for peer '{peer_name}'")
                return True

        except Exception as e:
            print(f"ERROR restoring access for {peer_name}: {e}")

        return False
