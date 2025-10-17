"""
Utilitaires cryptographiques pour Anemone
Gestion du chiffrement des invitations avec PIN
"""

import secrets
import base64
import json
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2
from cryptography.hazmat.primitives import hashes


def encrypt_invitation_with_pin(data: dict, pin: str = None) -> dict:
    """
    Chiffre une invitation avec un PIN optionnel

    Args:
        data: Dictionnaire contenant les informations d'invitation
        pin: PIN à 4-6 chiffres (optionnel)

    Returns:
        Dictionnaire avec données chiffrées ou en clair
    """
    json_data = json.dumps(data, separators=(',', ':'))  # Compact JSON

    if not pin:
        return {
            "version": 2,
            "encrypted": False,
            "data": json_data
        }

    # Générer un sel aléatoire
    salt = secrets.token_bytes(16)

    # Dériver une clé du PIN avec PBKDF2
    kdf = PBKDF2(
        algorithm=hashes.SHA256(),
        length=32,
        salt=salt,
        iterations=100000,
    )
    key = kdf.derive(pin.encode('utf-8'))

    # Chiffrer avec AES-256-GCM
    aesgcm = AESGCM(key)
    nonce = secrets.token_bytes(12)
    ciphertext = aesgcm.encrypt(nonce, json_data.encode('utf-8'), None)

    return {
        "version": 2,
        "encrypted": True,
        "salt": base64.b64encode(salt).decode('ascii'),
        "nonce": base64.b64encode(nonce).decode('ascii'),
        "data": base64.b64encode(ciphertext).decode('ascii'),
        "hint": f"PIN {len(pin)} chiffres"
    }


def decrypt_invitation_with_pin(payload: dict, pin: str) -> dict:
    """
    Déchiffre une invitation avec un PIN

    Args:
        payload: Données chiffrées du QR code
        pin: PIN pour déchiffrer

    Returns:
        Dictionnaire avec les données d'invitation

    Raises:
        ValueError: Si le PIN est incorrect ou les données corrompues
    """
    # Vérifier la version
    if payload.get("version") != 2:
        raise ValueError("Format d'invitation non supporté")

    # Si non chiffré, retourner directement
    if not payload.get("encrypted"):
        return json.loads(payload["data"])

    try:
        # Décoder les données
        salt = base64.b64decode(payload["salt"])
        nonce = base64.b64decode(payload["nonce"])
        ciphertext = base64.b64decode(payload["data"])

        # Dériver la clé du PIN
        kdf = PBKDF2(
            algorithm=hashes.SHA256(),
            length=32,
            salt=salt,
            iterations=100000,
        )
        key = kdf.derive(pin.encode('utf-8'))

        # Déchiffrer
        aesgcm = AESGCM(key)
        plaintext = aesgcm.decrypt(nonce, ciphertext, None)

        return json.loads(plaintext.decode('utf-8'))

    except Exception as e:
        raise ValueError(f"PIN incorrect ou données corrompues: {str(e)}")


def generate_random_pin(length: int = 6) -> str:
    """
    Génère un PIN aléatoire

    Args:
        length: Longueur du PIN (4-8 chiffres)

    Returns:
        PIN sous forme de chaîne
    """
    if length < 4 or length > 8:
        raise ValueError("La longueur du PIN doit être entre 4 et 8")

    # Générer un nombre aléatoire sécurisé
    max_value = 10 ** length - 1
    min_value = 10 ** (length - 1)

    pin_number = secrets.randbelow(max_value - min_value + 1) + min_value
    return str(pin_number)


def validate_pin(pin: str) -> bool:
    """
    Valide un PIN

    Args:
        pin: PIN à valider

    Returns:
        True si valide, False sinon
    """
    if not pin:
        return False

    # Doit être numérique
    if not pin.isdigit():
        return False

    # Longueur entre 4 et 8
    if len(pin) < 4 or len(pin) > 8:
        return False

    return True
