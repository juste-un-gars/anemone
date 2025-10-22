#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

set -e

# Variables
SMB_USER=${SMB_USER:-anemone}
SMB_PASSWORD=${SMB_PASSWORD:-changeme}
SMB_WORKGROUP=${SMB_WORKGROUP:-WORKGROUP}

# Créer l'utilisateur Samba s'il n'existe pas
if ! id "$SMB_USER" >/dev/null 2>&1; then
    adduser -D -H "$SMB_USER"
fi

# Configurer le mot de passe Samba
(echo "$SMB_PASSWORD"; echo "$SMB_PASSWORD") | smbpasswd -a -s "$SMB_USER"
smbpasswd -e "$SMB_USER"

# Générer smb.conf
cat > /etc/samba/smb.conf <<EOF
[global]
   workgroup = $SMB_WORKGROUP
   server string = Anemone File Server
   security = user
   map to guest = Bad User
   log file = /var/log/samba/%m.log
   max log size = 50

   # Performance
   socket options = TCP_NODELAY IPTOS_LOWDELAY SO_RCVBUF=131072 SO_SNDBUF=131072
   read raw = yes
   write raw = yes
   max xmit = 65536
   dead time = 15
   getwd cache = yes

[data]
   path = /mnt/data
   browseable = yes
   writable = yes
   guest ok = no
   valid users = $SMB_USER
   force user = $SMB_USER
   create mask = 0664
   directory mask = 0775
   comment = Local Data (not backed up)

[backup]
   path = /mnt/backup
   browseable = yes
   writable = yes
   guest ok = no
   valid users = $SMB_USER
   force user = $SMB_USER
   create mask = 0664
   directory mask = 0775
   comment = Backed Up Data

   # Corbeille (VFS recycle module)
   vfs objects = recycle
   recycle:repository = .trash
   recycle:keeptree = yes
   recycle:versions = yes
   recycle:touch = yes
   recycle:maxsize = 0
   recycle:exclude = *.tmp,*.temp,~*,*.swp,*.part
   recycle:exclude_dir = /tmp,/temp,.trash
EOF

# Permissions
chown -R "$SMB_USER:$SMB_USER" /mnt/data /mnt/backup 2>/dev/null || true

echo "✅ Samba configured"
