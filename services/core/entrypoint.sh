#!/bin/bash
set -e

echo "🪸 Anemone Core starting..."
echo "   WireGuard VPN + SFTP Server + Restic Backup"

# Vérifier les variables d'environnement
TZ=${TZ:-Europe/Paris}
PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Configurer le timezone
ln -snf /usr/share/zoneinfo/$TZ /etc/localtime
echo $TZ > /etc/timezone

# Créer les répertoires nécessaires
mkdir -p /config/wireguard /config/ssh /logs /var/run/wireguard

# Permissions
chmod 755 /logs
chmod 700 /config/ssh 2>/dev/null || true

# Si le fichier authorized_keys existe, le copier pour SSHD
if [ -f /config/ssh/authorized_keys ]; then
    mkdir -p /home/restic/.ssh
    cp /config/ssh/authorized_keys /home/restic/.ssh/authorized_keys
    chmod 600 /home/restic/.ssh/authorized_keys
    chown restic:restic /home/restic/.ssh/authorized_keys
fi

# Permissions du répertoire backups
if [ -d /home/restic/backups ]; then
    chown -R restic:restic /home/restic/backups
fi

echo "✅ Environment configured"
echo "   User: restic (UID: $PUID)"
echo "   Timezone: $TZ"
echo ""

# Lancer supervisord qui gérera WireGuard, SSHD et Restic
exec /usr/bin/supervisord -c /etc/supervisord.conf
