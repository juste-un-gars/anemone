"""
Gestionnaire de pairs pour Anemone
Gestion des invitations, configuration WireGuard et SSH
"""

import yaml
import os
import subprocess
from pathlib import Path
from typing import Optional, List, Dict
from datetime import datetime
from crypto_utils import encrypt_invitation_with_pin, decrypt_invitation_with_pin


class PeerManager:
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

    def _save_config(self):
        """Sauvegarde la configuration dans le fichier YAML"""
        with open(self.config_path, 'w') as f:
            yaml.dump(self.config, f, default_flow_style=False)

    def get_local_info(self) -> dict:
        """Récupère les informations locales du serveur"""
        # Lire la clé publique WireGuard
        wg_pubkey_path = "/config/wireguard/public.key"
        with open(wg_pubkey_path, 'r') as f:
            wg_pubkey = f.read().strip()

        # Lire la clé publique SSH
        ssh_pubkey_path = "/config/ssh/id_rsa.pub"
        with open(ssh_pubkey_path, 'r') as f:
            ssh_pubkey = f.read().strip()

        # Informations depuis config.yaml
        node_name = self.config.get('node', {}).get('name', 'anemone')
        vpn_address = self.config.get('wireguard', {}).get('address', '10.8.0.1/24')
        endpoint = self.config.get('wireguard', {}).get('public_endpoint', '')

        # Extraire l'IP sans le masque
        vpn_ip = vpn_address.split('/')[0]

        return {
            "node_name": node_name,
            "vpn_ip": vpn_ip,
            "wireguard_pubkey": wg_pubkey,
            "ssh_pubkey": ssh_pubkey,
            "endpoint": endpoint,
            "version": "1.0"
        }

    def generate_invitation(self, pin: Optional[str] = None) -> dict:
        """
        Génère une invitation pour ce serveur

        Args:
            pin: PIN optionnel pour chiffrer l'invitation

        Returns:
            Dictionnaire contenant l'invitation (chiffrée ou non)
        """
        local_info = self.get_local_info()
        local_info["timestamp"] = datetime.now().isoformat()

        # Chiffrer avec le PIN si fourni
        return encrypt_invitation_with_pin(local_info, pin)

    def get_used_vpn_ips(self) -> List[str]:
        """Récupère la liste des IPs VPN déjà utilisées"""
        ips = []

        # IP locale
        local_ip = self.config.get('wireguard', {}).get('address', '10.8.0.1/24').split('/')[0]
        ips.append(local_ip)

        # IPs des pairs
        peers = self.config.get('peers', [])
        for peer in peers:
            allowed_ips = peer.get('allowed_ips', '')
            if allowed_ips:
                # Format: "10.8.0.2/32"
                ip = allowed_ips.split('/')[0]
                ips.append(ip)

        return ips

    def get_next_available_vpn_ip(self) -> str:
        """Trouve la prochaine IP VPN disponible dans le subnet 10.8.0.0/24"""
        used_ips = self.get_used_vpn_ips()
        base = "10.8.0."

        for i in range(2, 255):  # 10.8.0.1 réservé au premier serveur
            ip = f"{base}{i}"
            if ip not in used_ips:
                return ip

        raise Exception("Aucune IP disponible dans le subnet 10.8.0.0/24")

    def add_peer_from_invitation(self, invitation_data: dict, pin: Optional[str] = None) -> dict:
        """
        Ajoute un pair depuis une invitation (chiffrée ou non)

        Args:
            invitation_data: Données du QR code
            pin: PIN pour déchiffrer (si nécessaire)

        Returns:
            Informations du pair ajouté

        Raises:
            ValueError: Si le PIN est incorrect ou données invalides
        """
        # Déchiffrer si nécessaire
        print(f"DEBUG: invitation_data keys: {invitation_data.keys()}", flush=True)
        print(f"DEBUG: encrypted={invitation_data.get('encrypted')}, version={invitation_data.get('version')}", flush=True)

        if invitation_data.get("encrypted"):
            if not pin:
                raise ValueError("PIN requis pour déchiffrer cette invitation")
            peer_info = decrypt_invitation_with_pin(invitation_data, pin)
            print(f"DEBUG: Decrypted peer_info keys: {peer_info.keys()}", flush=True)
        elif invitation_data.get("version") == 2 and "data" in invitation_data:
            # Format v2 non chiffré - extraire les données JSON
            import json
            print(f"DEBUG: Parsing v2 non-encrypted format", flush=True)
            print(f"DEBUG: data field content: {invitation_data['data'][:100]}...", flush=True)
            peer_info = json.loads(invitation_data["data"])
            print(f"DEBUG: Parsed peer_info keys: {peer_info.keys()}", flush=True)
        else:
            # Format legacy ou déjà parsé
            print(f"DEBUG: Using legacy format or already parsed", flush=True)
            peer_info = invitation_data

        # Valider les champs requis
        required_fields = ["node_name", "vpn_ip", "wireguard_pubkey", "ssh_pubkey"]
        print(f"DEBUG: peer_info keys before validation: {peer_info.keys()}", flush=True)
        for field in required_fields:
            if field not in peer_info:
                print(f"DEBUG: Missing field '{field}' in peer_info. Available: {list(peer_info.keys())}", flush=True)
                raise ValueError(f"Champ manquant dans l'invitation: {field}")

        # Vérifier que l'IP VPN n'est pas déjà utilisée
        used_ips = self.get_used_vpn_ips()
        if peer_info["vpn_ip"] in used_ips:
            raise ValueError(f"L'IP VPN {peer_info['vpn_ip']} est déjà utilisée")

        # Ajouter le pair à la configuration WireGuard
        new_peer = {
            "name": peer_info["node_name"],
            "public_key": peer_info["wireguard_pubkey"],
            "allowed_ips": f"{peer_info['vpn_ip']}/32",
            "persistent_keepalive": 25
        }

        if peer_info.get("endpoint"):
            new_peer["endpoint"] = peer_info["endpoint"]

        # Ajouter à la liste des pairs
        if 'peers' not in self.config:
            self.config['peers'] = []

        self.config['peers'].append(new_peer)
        self._save_config()

        # Ajouter la clé SSH aux authorized_keys
        self._add_ssh_key(peer_info["ssh_pubkey"])

        # Auto-créer le target de backup pour ce pair
        self._add_backup_target(peer_info["node_name"], peer_info["vpn_ip"])

        # Redémarrer WireGuard pour appliquer les changements
        self._restart_wireguard()

        # Redémarrer Restic pour qu'il prenne en compte les nouveaux targets
        self._restart_restic()

        return {
            "name": peer_info["node_name"],
            "vpn_ip": peer_info["vpn_ip"],
            "endpoint": peer_info.get("endpoint", "N/A"),
            "status": "added"
        }

    def _add_ssh_key(self, ssh_pubkey: str):
        """Ajoute une clé SSH publique aux authorized_keys"""
        authorized_keys_path = "/config/ssh/authorized_keys"

        # Créer le fichier s'il n'existe pas
        Path(authorized_keys_path).touch(mode=0o600, exist_ok=True)

        # Lire les clés existantes
        with open(authorized_keys_path, 'r') as f:
            existing_keys = f.read()

        # Vérifier si la clé existe déjà
        if ssh_pubkey in existing_keys:
            return

        # Ajouter la nouvelle clé
        with open(authorized_keys_path, 'a') as f:
            f.write(f"\n{ssh_pubkey}\n")

    def _add_backup_target(self, peer_name: str, peer_vpn_ip: str):
        """
        Ajoute automatiquement un target de backup pour un pair

        Args:
            peer_name: Nom du pair
            peer_vpn_ip: IP VPN du pair
        """
        # Récupérer le nom local du serveur
        local_name = self.config.get('node', {}).get('name', 'anemone')

        # Créer l'entrée de backup target
        backup_target = {
            "name": f"{peer_name}-backup",
            "enabled": True,
            "type": "sftp",
            "host": peer_vpn_ip,
            "port": 2222,
            "user": "restic",
            "path": f"/backups/{local_name}"
        }

        # Initialiser la section backup si nécessaire
        if 'backup' not in self.config:
            self.config['backup'] = {}

        if 'targets' not in self.config['backup']:
            self.config['backup']['targets'] = []

        # Vérifier si le target existe déjà (éviter les doublons)
        existing_targets = self.config['backup']['targets']
        for target in existing_targets:
            if target.get('name') == backup_target['name']:
                print(f"✓ Backup target '{backup_target['name']}' already exists")
                return

        # Ajouter le nouveau target
        self.config['backup']['targets'].append(backup_target)
        self._save_config()

        # Créer le répertoire pour recevoir les backups de ce pair
        backup_receive_path = self.config.get('storage', {}).get('backup_receive_path', '/mnt/backups')
        peer_backup_dir = Path(backup_receive_path) / peer_name
        peer_backup_dir.mkdir(parents=True, exist_ok=True)

        print(f"✓ Backup target '{backup_target['name']}' added")
        print(f"✓ Backup directory created: {peer_backup_dir}")

        # S'assurer que le conteneur SFTP est démarré
        self._ensure_sftp_running()

    def _ensure_sftp_running(self):
        """Démarre le conteneur SFTP s'il n'est pas déjà en cours d'exécution"""
        try:
            # Vérifier si le conteneur SFTP existe et est en cours d'exécution
            result = subprocess.run(
                ["docker", "ps", "--filter", "name=anemone-sftp", "--format", "{{.Names}}"],
                check=True,
                capture_output=True,
                text=True
            )

            if "anemone-sftp" in result.stdout:
                print("✓ SFTP container already running")
                return

            # Le conteneur n'est pas en cours d'exécution, le démarrer
            print("Starting SFTP container...")
            subprocess.run(
                ["docker", "compose", "up", "-d", "sftp"],
                check=True,
                capture_output=True,
                cwd="/app"  # Le docker-compose.yml est à la racine du projet
            )
            print("✓ SFTP container started")

        except subprocess.CalledProcessError as e:
            print(f"ERROR starting SFTP container: {e}")
            print(f"stderr: {e.stderr.decode() if e.stderr else 'N/A'}")

    def _regenerate_wireguard_config(self):
        """Régénère le fichier wg0.conf depuis config.yaml"""
        try:
            subprocess.run(
                ["python3", "/scripts/generate-wireguard-config.py", self.config_path],
                check=True,
                capture_output=True
            )
            print("✓ WireGuard configuration regenerated")
        except subprocess.CalledProcessError as e:
            print(f"ERROR regenerating WireGuard config: {e}")
            print(f"stderr: {e.stderr.decode() if e.stderr else 'N/A'}")

    def _restart_wireguard(self):
        """Régénère la config et redémarre le conteneur WireGuard"""
        # Régénérer wg0.conf
        self._regenerate_wireguard_config()

        # Redémarrer le conteneur
        try:
            subprocess.run(
                ["docker", "restart", "anemone-wireguard"],
                check=True,
                capture_output=True
            )
            print("✓ WireGuard restarted")
        except subprocess.CalledProcessError as e:
            print(f"ERROR restarting WireGuard: {e}")

    def _restart_restic(self):
        """Redémarre le conteneur Restic pour qu'il prenne en compte les changements de config"""
        try:
            subprocess.run(
                ["docker", "restart", "anemone-restic"],
                check=True,
                capture_output=True
            )
            print("✓ Restic restarted")
        except subprocess.CalledProcessError as e:
            print(f"ERROR restarting Restic: {e}")

    def list_peers(self) -> List[dict]:
        """Liste tous les pairs configurés"""
        peers = self.config.get('peers', [])
        peer_list = []

        for peer in peers:
            peer_list.append({
                "name": peer.get("name", "Unknown"),
                "vpn_ip": peer.get("allowed_ips", "").split('/')[0],
                "endpoint": peer.get("endpoint", "N/A"),
                "public_key": peer.get("public_key", "")
            })

        return peer_list

    def remove_peer(self, peer_name: str) -> bool:
        """
        Supprime un pair par son nom

        Args:
            peer_name: Nom du pair à supprimer

        Returns:
            True si supprimé, False si non trouvé
        """
        peers = self.config.get('peers', [])
        initial_count = len(peers)

        # Filtrer pour retirer le pair
        self.config['peers'] = [p for p in peers if p.get('name') != peer_name]

        if len(self.config['peers']) < initial_count:
            self._save_config()
            self._restart_wireguard()
            return True

        return False

    def get_peer_status(self, peer_name: str) -> dict:
        """
        Récupère le statut d'un pair (connecté ou non)

        Args:
            peer_name: Nom du pair

        Returns:
            Dictionnaire avec le statut
        """
        # Trouver le pair
        peers = self.config.get('peers', [])
        peer = next((p for p in peers if p.get('name') == peer_name), None)

        if not peer:
            return {"status": "unknown", "error": "Pair non trouvé"}

        vpn_ip = peer.get("allowed_ips", "").split('/')[0]

        # Tester la connectivité via ping
        try:
            result = subprocess.run(
                ["docker", "exec", "anemone-wireguard", "ping", "-c", "1", "-W", "2", vpn_ip],
                capture_output=True,
                timeout=5
            )
            connected = result.returncode == 0
        except:
            connected = False

        return {
            "name": peer_name,
            "vpn_ip": vpn_ip,
            "status": "connected" if connected else "disconnected"
        }
