#!/bin/bash
set -e

# Variables
WEBDAV_USER=${WEBDAV_USER:-anemone}
WEBDAV_PASSWORD=${WEBDAV_PASSWORD:-changeme}
WEBDAV_PORT=${WEBDAV_PORT:-8080}

# Créer le répertoire WebDAV
mkdir -p /var/www/webdav/data /var/www/webdav/backup

# Créer des liens symboliques vers les vrais répertoires
ln -sfn /mnt/data /var/www/webdav/data
ln -sfn /mnt/backup /var/www/webdav/backup

# Créer le fichier de mots de passe WebDAV
htpasswd -bc /etc/apache2/.htpasswd "$WEBDAV_USER" "$WEBDAV_PASSWORD"

# Configuration Apache
cat > /etc/apache2/conf.d/webdav.conf <<EOF
Listen $WEBDAV_PORT

<VirtualHost *:$WEBDAV_PORT>
    ServerAdmin webmaster@localhost
    DocumentRoot /var/www/webdav

    <Directory /var/www/webdav>
        DAV On
        Options Indexes FollowSymLinks
        AuthType Basic
        AuthName "Anemone WebDAV"
        AuthUserFile /etc/apache2/.htpasswd
        Require valid-user

        # Permissions
        AllowOverride None
        Order allow,deny
        Allow from all
    </Directory>

    ErrorLog /logs/webdav-error.log
    CustomLog /logs/webdav-access.log combined
</VirtualHost>
EOF

# Activer les modules nécessaires
sed -i 's/^#LoadModule dav_module/LoadModule dav_module/' /etc/apache2/httpd.conf
sed -i 's/^#LoadModule dav_fs_module/LoadModule dav_fs_module/' /etc/apache2/httpd.conf
sed -i 's/^#LoadModule auth_digest_module/LoadModule auth_digest_module/' /etc/apache2/httpd.conf

# Permissions
chown -R apache:apache /var/www/webdav
chmod -R 755 /var/www/webdav

echo "✅ WebDAV configured on port $WEBDAV_PORT"
