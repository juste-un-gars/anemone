FROM alpine:latest

LABEL maintainer="Anemone Project"
LABEL description="Anemone Shares - Samba SMB + WebDAV File Sharing"

# Installer Samba et dépendances WebDAV
RUN apk add --no-cache \
    samba \
    samba-common-tools \
    apache2 \
    apache2-webdav \
    apache2-utils \
    bash \
    supervisor \
    tzdata

# Créer les répertoires
RUN mkdir -p /var/log/samba \
    /var/lib/samba \
    /var/cache/samba \
    /run/samba \
    /etc/samba \
    /logs \
    /mnt/data \
    /mnt/backup

# Configuration Apache pour WebDAV
RUN mkdir -p /var/www/webdav && \
    mkdir -p /run/apache2 && \
    chown -R apache:apache /var/www/webdav

# Copier les scripts de configuration
COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

# Copier la configuration supervisord
COPY supervisord.conf /etc/supervisord.conf

# Copier l'entrypoint
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Exposer les ports
EXPOSE 445 139 80 443

WORKDIR /scripts

ENTRYPOINT ["/entrypoint.sh"]
