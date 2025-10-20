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
    restic \
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
    chown -R restic:restic /home/restic

# Configuration SSHD pour SFTP
RUN echo "Port 22" >> /etc/ssh/sshd_config && \
    echo "PermitRootLogin no" >> /etc/ssh/sshd_config && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "AuthorizedKeysFile /config/ssh/authorized_keys" >> /etc/ssh/sshd_config && \
    echo "ChallengeResponseAuthentication no" >> /etc/ssh/sshd_config && \
    echo "UsePAM no" >> /etc/ssh/sshd_config && \
    echo "Subsystem sftp internal-sftp" >> /etc/ssh/sshd_config && \
    echo "Match User restic" >> /etc/ssh/sshd_config && \
    echo "    ChrootDirectory /home/restic" >> /etc/ssh/sshd_config && \
    echo "    ForceCommand internal-sftp" >> /etc/ssh/sshd_config && \
    echo "    X11Forwarding no" >> /etc/ssh/sshd_config && \
    echo "    AllowTcpForwarding no" >> /etc/ssh/sshd_config

# Créer les répertoires nécessaires
RUN mkdir -p /data /config /logs /scripts /root/.ssh /var/run/wireguard

# Copier les scripts Restic existants
COPY restic-scripts/ /scripts/
RUN chmod +x /scripts/*.sh /scripts/*.py

# Copier le script de génération WireGuard depuis la racine
COPY ../../scripts/generate-wireguard-config.py /scripts/
RUN chmod +x /scripts/generate-wireguard-config.py

# Copier les scripts spécifiques à core
COPY scripts/ /scripts/core/
RUN chmod +x /scripts/core/*.sh

# Copier la configuration supervisord
COPY supervisord.conf /etc/supervisord.conf

# Copier l'entrypoint
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /scripts

# Exposer les ports
# Note: Avec network_mode: host, ces EXPOSE sont informatifs
EXPOSE 51820/udp 22

ENTRYPOINT ["/entrypoint.sh"]
