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

# IMPORTANT: Pour ChrootDirectory, /home/restic DOIT être possédé par root
# avec permissions strictes (pas de write pour group/others)
chown root:root /home/restic
chmod 755 /home/restic

# Mais les sous-répertoires peuvent appartenir à restic
if [ -d /home/restic/backups ]; then
    chown -R restic:restic /home/restic/backups
fi

# Le répertoire .ssh doit aussi appartenir à restic
if [ -d /home/restic/.ssh ]; then
    chown -R restic:restic /home/restic/.ssh
    chmod 700 /home/restic/.ssh
fi

echo "✅ Environment configured"
echo "   User: restic (UID: $PUID)"
echo "   Timezone: $TZ"
echo ""

# Configurer le cron job pour le backup automatique de configuration (2h du matin)
echo "0 2 * * * /scripts/core/backup-config-auto.sh >> /logs/config-backup.log 2>&1" > /etc/crontabs/root
chmod 600 /etc/crontabs/root
echo "✅ Cron job configured (daily config backup at 2:00 AM)"
echo ""

# Lancer supervisord qui gérera WireGuard, SSHD, Restic et Cron
exec /usr/bin/supervisord -c /etc/supervisord.conf
