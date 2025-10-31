#!/bin/bash
# Anemone - Configure sudoers for SMB reload without password
# This allows the anemone service to reload smbd automatically

set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Get the user who will run anemone
ANEMONE_USER="${1:-anemone}"

echo "Configuring sudoers for automatic SMB reload..."
echo "User: $ANEMONE_USER"

# Create sudoers file for anemone
cat > /etc/sudoers.d/anemone-smb << EOF
# Allow anemone to manage SMB users and reload service without password
# Created by Anemone setup script
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smb
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smb.service
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smbd
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smbd.service
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/sbin/useradd -M -s /usr/sbin/nologin *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/sbin/userdel *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/smbpasswd
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/chown -R *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/chmod *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/cp * /etc/samba/smb.conf
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/mv *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/rm *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/rmdir *
$ANEMONE_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir *
EOF

# Set correct permissions (sudoers files must be 0440)
chmod 0440 /etc/sudoers.d/anemone-smb

# Test the sudoers file
if visudo -c -f /etc/sudoers.d/anemone-smb; then
    echo "✅ Sudoers configuration successful!"
    echo "   User '$ANEMONE_USER' can now reload smbd without password"
else
    echo "❌ Error in sudoers configuration"
    rm -f /etc/sudoers.d/anemone-smb
    exit 1
fi

echo ""
echo "Configuration complete. Anemone will now reload smbd automatically."
