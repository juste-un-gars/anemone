#!/bin/bash
set -e

echo "📁 Anemone Shares starting..."
echo "   Samba SMB + WebDAV File Sharing"

# Variables d'environnement
TZ=${TZ:-Europe/Paris}
SMB_USER=${SMB_USER:-anemone}
SMB_PASSWORD=${SMB_PASSWORD:-changeme}
SMB_WORKGROUP=${SMB_WORKGROUP:-WORKGROUP}
WEBDAV_USER=${WEBDAV_USER:-anemone}
WEBDAV_PASSWORD=${WEBDAV_PASSWORD:-changeme}

# Configurer le timezone
ln -snf /usr/share/zoneinfo/$TZ /etc/localtime
echo $TZ > /etc/timezone

# Créer les répertoires
mkdir -p /mnt/data /mnt/backup /logs /var/log/samba /run/samba

# Configurer Samba
echo "🔧 Configuring Samba..."
/scripts/configure-samba.sh

# Configurer WebDAV
echo "🔧 Configuring WebDAV..."
/scripts/configure-webdav.sh

echo "✅ Configuration completed"
echo ""

# Lancer supervisord
exec /usr/bin/supervisord -c /etc/supervisord.conf
