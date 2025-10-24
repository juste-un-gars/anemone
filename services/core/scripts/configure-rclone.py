#!/usr/bin/env python3
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

"""
Configure rclone for encrypted mirroring to peers.
Uses the Restic encryption key to encrypt data before syncing.
"""

import os
import sys
import yaml
import subprocess
import hashlib
from pathlib import Path

CONFIG_PATH = os.getenv('CONFIG_PATH', '/config/config.yaml')
RCLONE_CONFIG_DIR = Path('/root/.config/rclone')
RCLONE_CONFIG_FILE = RCLONE_CONFIG_DIR / 'rclone.conf'
SSH_KEY = '/config/ssh/id_rsa'


def obscure_password(password):
    """Use rclone's obscure command to encrypt password for config."""
    try:
        result = subprocess.run(
            ['rclone', 'obscure', password],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        print(f"❌ Failed to obscure password: {e.stderr}", file=sys.stderr)
        sys.exit(1)


def generate_salt_from_key(restic_key):
    """Generate a deterministic salt from the Restic key."""
    # Use SHA256 to derive a salt password from the main key
    salt_hash = hashlib.sha256(f"{restic_key}-salt".encode()).hexdigest()
    return salt_hash[:32]  # Use first 32 chars


def configure_rclone():
    """Generate rclone configuration for all backup targets."""

    # Get Restic password from environment
    restic_password = os.getenv('RESTIC_PASSWORD')
    if not restic_password:
        print("❌ RESTIC_PASSWORD not set in environment", file=sys.stderr)
        sys.exit(1)

    print("🔧 Configuring rclone for encrypted mirroring...")

    # Load config.yaml
    try:
        with open(CONFIG_PATH) as f:
            config = yaml.safe_load(f)
    except Exception as e:
        print(f"❌ Failed to read config: {e}", file=sys.stderr)
        sys.exit(1)

    targets = config.get('backup', {}).get('targets', [])
    enabled_targets = [t for t in targets if t.get('enabled', True)]

    if not enabled_targets:
        print("⚠️  No enabled backup targets configured")
        return

    print(f"📦 Configuring {len(enabled_targets)} target(s)")

    # Create rclone config directory
    RCLONE_CONFIG_DIR.mkdir(parents=True, exist_ok=True)

    # Obscure passwords for rclone config
    print("🔐 Encrypting passwords...")
    obscured_password = obscure_password(restic_password)
    salt = generate_salt_from_key(restic_password)
    obscured_salt = obscure_password(salt)

    # Generate rclone.conf
    config_lines = []

    for target in enabled_targets:
        name = target.get('name', 'unknown')
        host = target.get('host')
        port = target.get('port', 22222)
        user = target.get('user', 'restic')
        path = target.get('path', 'backups')

        if not host:
            print(f"⚠️  Target {name}: no host configured, skipping", file=sys.stderr)
            continue

        # Normalize path (remove leading /)
        if path.startswith('/'):
            path = path[1:]

        # Remote name without "-backup" suffix
        remote_base = name.replace('-backup', '')
        sftp_remote = f"{remote_base}-sftp"
        crypt_remote = f"{remote_base}-crypt"

        print(f"  ✓ Configuring {name}: {host}:{path}")

        # SFTP remote configuration
        config_lines.extend([
            f"[{sftp_remote}]",
            f"type = sftp",
            f"host = {host}",
            f"user = {user}",
            f"port = {port}",
            f"key_file = {SSH_KEY}",
            f"shell_type = unix",
            f"md5sum_command = md5sum",
            f"sha1sum_command = sha1sum",
            "",
        ])

        # Crypt remote configuration (wraps SFTP)
        config_lines.extend([
            f"[{crypt_remote}]",
            f"type = crypt",
            f"remote = {sftp_remote}:{path}",
            f"password = {obscured_password}",
            f"password2 = {obscured_salt}",
            f"filename_encryption = standard",
            f"directory_name_encryption = true",
            f"filename_encoding = base32768",
            "",
        ])

    # Write config file
    try:
        with open(RCLONE_CONFIG_FILE, 'w') as f:
            f.write('\n'.join(config_lines))

        # Secure permissions
        os.chmod(RCLONE_CONFIG_FILE, 0o600)

        print(f"✅ Rclone configuration created: {RCLONE_CONFIG_FILE}")
        print("🔒 All data will be encrypted with AES-256 before syncing")
        print("🔒 Filenames will be encrypted (unreadable on remote)")

    except Exception as e:
        print(f"❌ Failed to write rclone config: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    configure_rclone()
