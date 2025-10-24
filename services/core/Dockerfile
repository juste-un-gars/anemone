FROM alpine:latest

LABEL maintainer="Anemone Project"
LABEL description="Anemone Core - WireGuard VPN + SFTP Server + Restic Backup"

# Installer les dépendances
RUN apk add --no-cache \
    wireguard-tools \
    iptables \
    ip6tables \
    iproute2 \
    openrc \
    openssh-server \
    openssh-client \
    restic \
    rsync \
    rclone \
    bash \
    curl \
    dcron \
    python3 \
    py3-pip \
    py3-yaml \
    py3-cryptography \
    inotify-tools \
    supervisor \
    tzdata

# Configuration SSH/SFTP
RUN ssh-keygen -A && \
    mkdir -p /run/sshd && \
    mkdir -p /home/restic/.ssh && \
    adduser -D -s /bin/sh -u 1000 restic && \
    echo "restic:*" | chpasswd -e && \
    mkdir -p /home/restic/backups && \
    chown -R restic:restic /home/restic && \
    mkdir -p /root/.ssh && \
    chmod 700 /root/.ssh && \
    echo "Host *" > /root/.ssh/config && \
    echo "    Port 22222" >> /root/.ssh/config && \
    echo "    StrictHostKeyChecking accept-new" >> /root/.ssh/config && \
    echo "    UserKnownHostsFile /root/.ssh/known_hosts" >> /root/.ssh/config && \
    chmod 600 /root/.ssh/config

# Configuration SSHD pour SSH/SFTP (rsync nécessite shell access)
RUN echo "Port 22222" >> /etc/ssh/sshd_config && \
    echo "PermitRootLogin no" >> /etc/ssh/sshd_config && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "AuthorizedKeysFile .ssh/authorized_keys" >> /etc/ssh/sshd_config && \
    echo "ChallengeResponseAuthentication no" >> /etc/ssh/sshd_config && \
    echo "UsePAM no" >> /etc/ssh/sshd_config && \
    echo "Subsystem sftp internal-sftp" >> /etc/ssh/sshd_config && \
    echo "Match User restic" >> /etc/ssh/sshd_config && \
    echo "    X11Forwarding no" >> /etc/ssh/sshd_config && \
    echo "    AllowTcpForwarding no" >> /etc/ssh/sshd_config

# Créer les répertoires nécessaires
RUN mkdir -p /data /config /logs /scripts /root/.ssh /var/run/wireguard

# Copier les scripts Restic existants (depuis services/core/restic-scripts/)
COPY services/core/restic-scripts/ /scripts/
RUN chmod +x /scripts/*.sh /scripts/*.py

# Copier le script de génération WireGuard depuis la racine du projet
COPY scripts/generate-wireguard-config.py /scripts/
RUN chmod +x /scripts/generate-wireguard-config.py

# Copier les scripts spécifiques à core (depuis services/core/scripts/)
COPY services/core/scripts/ /scripts/core/
RUN chmod +x /scripts/core/*.sh /scripts/core/*.py

# Copier la configuration supervisord
COPY services/core/supervisord.conf /etc/supervisord.conf

# Copier l'entrypoint
COPY services/core/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /scripts

# Exposer les ports
# Note: Avec network_mode: host, ces EXPOSE sont informatifs
EXPOSE 51820/udp 22222

ENTRYPOINT ["/entrypoint.sh"]
