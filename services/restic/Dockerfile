FROM alpine:latest

LABEL maintainer="Anemone Project"

RUN apk add --no-cache \
    restic \
    openssh-client \
    bash \
    curl \
    dcron \
    python3 \
    py3-pip \
    py3-yaml \
    inotify-tools \
    tzdata

RUN pip3 install --no-cache-dir watchdog pyyaml cryptography

RUN mkdir -p /data /config /logs /scripts /root/.ssh

COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

COPY decrypt_key.py /scripts/decrypt_key.py
RUN chmod +x /scripts/decrypt_key.py

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /scripts

ENTRYPOINT ["/entrypoint.sh"]
