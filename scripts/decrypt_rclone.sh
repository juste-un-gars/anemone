#!/bin/bash
# decrypt_rclone.sh - Decrypt files backed up via rclone crypt
# Usage: decrypt_rclone.sh <crypt_password> <encrypted_directory> [output_directory]

set -euo pipefail

if [ $# -lt 2 ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo "Usage: $0 <crypt_password> <encrypted_directory> [output_directory]"
    echo ""
    echo "  crypt_password       Rclone crypt obscured password (from DB provider_config)"
    echo "  encrypted_directory  Directory containing rclone-encrypted files"
    echo "  output_directory     Where to write decrypted files (default: <encrypted_directory>_decrypted)"
    echo ""
    echo "Example:"
    echo "  $0 'obscured_password' /home/franck/pcloud-backup"
    echo "  $0 'obscured_password' /home/franck/pcloud-backup /tmp/restored"
    exit 1
fi

# Obscure the password for rclone crypt format
PASSWORD="$(rclone obscure "$1")"
SRC_DIR="$(realpath "$2")"
OUT_DIR="${3:-${SRC_DIR}_decrypted}"

if [ ! -d "$SRC_DIR" ]; then
    echo "Error: directory not found: $SRC_DIR"
    exit 1
fi

if ! command -v rclone &>/dev/null; then
    echo "Error: rclone is not installed"
    exit 1
fi

mkdir -p "$OUT_DIR"

echo "Source:  $SRC_DIR"
echo "Output:  $OUT_DIR"
echo ""

# Use rclone crypt inline remote to decrypt
rclone copy \
    ":crypt,remote=$SRC_DIR,password=$PASSWORD,filename_encryption=standard:" \
    "$OUT_DIR" \
    --progress

echo ""
echo "Done. Decrypted files in: $OUT_DIR"
