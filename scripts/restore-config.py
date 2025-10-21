#!/usr/bin/env python3
"""
Script de restauration de configuration Anemone depuis un fichier chiffré
Usage: python3 restore-config.py <fichier.enc> <clé_restic>
"""

import sys
import tarfile
import io
from pathlib import Path
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend


def restore_configuration(encrypted_file: str, restic_key: str, config_dir: str = "./config"):
    """
    Restaure la configuration depuis un fichier chiffré

    Args:
        encrypted_file: Chemin vers le fichier .enc
        restic_key: Clé Restic pour déchiffrer
        config_dir: Répertoire de destination (par défaut ./config)
    """
    try:
        print(f"📦 Lecture du fichier chiffré: {encrypted_file}")

        # Lire le fichier chiffré
        with open(encrypted_file, 'rb') as f:
            encrypted_data = f.read()

        # Extraire IV (16 premiers bytes)
        iv = encrypted_data[:16]
        ciphertext = encrypted_data[16:]

        print("🔓 Déchiffrement de l'archive...")

        # Dériver la clé de déchiffrement
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=b"anemone-config-export",  # Même salt qu'à l'export
            iterations=100000,
            backend=default_backend()
        )
        decryption_key = kdf.derive(restic_key.encode())

        # Déchiffrer avec AES-256-CBC
        cipher = Cipher(
            algorithms.AES(decryption_key),
            modes.CBC(iv),
            backend=default_backend()
        )
        decryptor = cipher.decryptor()
        padded_data = decryptor.update(ciphertext) + decryptor.finalize()

        # Retirer le padding PKCS7
        padding_length = padded_data[-1]
        tar_data = padded_data[:-padding_length]

        print("📂 Extraction de l'archive...")

        # Créer le répertoire config s'il n'existe pas
        config_path = Path(config_dir)
        config_path.mkdir(parents=True, exist_ok=True)

        # Extraire l'archive tar.gz
        tar_buffer = io.BytesIO(tar_data)
        with tarfile.open(fileobj=tar_buffer, mode='r:gz') as tar:
            # Lister les fichiers
            members = tar.getmembers()
            print(f"   Fichiers à restaurer: {len(members)}")

            for member in members:
                print(f"   - {member.name}")

            # Extraire tout dans config/
            tar.extractall(path=config_path)

        # Fixer les permissions
        print("🔒 Application des permissions...")

        # WireGuard keys
        wg_dir = config_path / "wireguard"
        if wg_dir.exists():
            (wg_dir / "private.key").chmod(0o600) if (wg_dir / "private.key").exists() else None
            (wg_dir / "public.key").chmod(0o644) if (wg_dir / "public.key").exists() else None

        # SSH keys
        ssh_dir = config_path / "ssh"
        if ssh_dir.exists():
            ssh_dir.chmod(0o700)
            (ssh_dir / "id_rsa").chmod(0o600) if (ssh_dir / "id_rsa").exists() else None
            (ssh_dir / "id_rsa.pub").chmod(0o644) if (ssh_dir / "id_rsa.pub").exists() else None
            (ssh_dir / "authorized_keys").chmod(0o600) if (ssh_dir / "authorized_keys").exists() else None

        # Restic encrypted key
        restic_enc = config_path / ".restic.encrypted"
        restic_salt = config_path / ".restic.salt"
        if restic_enc.exists():
            restic_enc.chmod(0o600)
        if restic_salt.exists():
            restic_salt.chmod(0o600)

        # Créer le marqueur de setup complété
        setup_marker = config_path / ".setup-completed"
        setup_marker.touch()

        print(f"✅ Configuration restaurée avec succès dans {config_path}")
        print("")
        print("Fichiers restaurés:")
        print(f"  - Configuration: {config_path / 'config.yaml'}")
        print(f"  - Clés WireGuard: {wg_dir}")
        print(f"  - Clés SSH: {ssh_dir}")
        print(f"  - Clé Restic: {restic_enc}")
        print("")
        print("⚠️  IMPORTANT:")
        print("  1. Vérifiez le contenu de config/config.yaml")
        print("  2. Lancez Docker: docker compose up -d")
        print("  3. Les backups seront automatiquement synchronisés depuis les peers")

        return True

    except Exception as e:
        print(f"❌ Erreur lors de la restauration: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python3 restore-config.py <fichier.enc> <clé_restic>")
        print("")
        print("Exemple:")
        print("  python3 restore-config.py anemone-backup-FR1-20251021-095520.enc 'votre-clé-restic'")
        sys.exit(1)

    encrypted_file = sys.argv[1]
    restic_key = sys.argv[2]

    if not Path(encrypted_file).exists():
        print(f"❌ Fichier introuvable: {encrypted_file}", file=sys.stderr)
        sys.exit(1)

    success = restore_configuration(encrypted_file, restic_key)
    sys.exit(0 if success else 1)
