#!/usr/bin/env python3
"""
Decrypt Restic key using Python cryptography library
"""
import sys
import os
from pathlib import Path
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend


def get_system_key():
    """Get system key from HOSTNAME (persistent across container restarts)"""
    # IMPORTANT : Ne PAS utiliser UUID qui change à chaque redémarrage
    return os.getenv('HOSTNAME', 'anemone')


def decrypt_restic_key(encrypted_path: str, salt_path: str) -> str:
    """Decrypt the Restic key"""
    try:
        # Get system key
        system_key = get_system_key()

        # Read salt
        with open(salt_path, 'r') as f:
            salt_hex = f.read().strip()
        salt = bytes.fromhex(salt_hex)

        # Read encrypted data (IV + encrypted)
        with open(encrypted_path, 'rb') as f:
            encrypted_data = f.read()

        # Extract IV (first 16 bytes) and encrypted content
        iv = encrypted_data[:16]
        encrypted = encrypted_data[16:]

        # Derive decryption key using PBKDF2
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=salt,
            iterations=100000,
            backend=default_backend()
        )
        derived_key = kdf.derive(f"{system_key}".encode())

        # Decrypt using AES-256-CBC
        cipher = Cipher(
            algorithms.AES(derived_key),
            modes.CBC(iv),
            backend=default_backend()
        )
        decryptor = cipher.decryptor()

        # Decrypt
        decrypted_padded = decryptor.update(encrypted) + decryptor.finalize()

        # Remove padding
        padding_length = decrypted_padded[-1]
        decrypted = decrypted_padded[:-padding_length]

        return decrypted.decode('utf-8')

    except Exception as e:
        print(f"Error decrypting key: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    encrypted_path = "/config/.restic.encrypted"
    salt_path = "/config/.restic.salt"

    if not Path(encrypted_path).exists() or not Path(salt_path).exists():
        print("Error: Encrypted key or salt not found", file=sys.stderr)
        sys.exit(1)

    key = decrypt_restic_key(encrypted_path, salt_path)
    print(key, end='')  # No newline to avoid issues with environment variables
