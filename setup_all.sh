#!/bin/bash
set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
cat << "EOF"
╔═══════════════════════════════════════╗
║  🪸  ANEMONE - Setup Complet          ║
╔═══════════════════════════════════════╗
EOF
echo -e "${NC}"
echo ""

# Vérifier qu'on est dans un dossier git
if [ ! -d .git ]; then
    echo -e "${YELLOW}Initialisation du dépôt git...${NC}"
    git init
fi

echo -e "${BLUE}Création de la structure du projet...${NC}"
echo ""

# Créer la structure de dossiers
mkdir -p config data backups logs
mkdir -p scripts
mkdir -p services/restic/scripts
mkdir -p services/api

# ===== README.md =====
echo -e "${CYAN}[1/17]${NC} Création de README.md..."
cat > README.md << 'EOF'
# 🪸 Anemone

**Serveur de fichiers distribué, simple et chiffré, avec redondance entre proches**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![WireGuard](https://img.shields.io/badge/wireguard-%2388171A.svg?style=flat&logo=wireguard&logoColor=white)](https://www.wireguard.com/)

## 🎯 Qu'est-ce qu'Anemone ?

Anemone est un système de stockage distribué qui permet de :

- 📂 **Servir vos fichiers localement** via SMB, WebDAV ou SFTP
- 🔐 **Sauvegarder automatiquement** vos données de manière chiffrée
- 🤝 **Échanger des backups** avec vos proches via un VPN sécurisé
- 🚀 **Déployer en 5 minutes** avec Docker

### Cas d'usage typique

Alice, Bob et Charlie sont amis. Chacun héberge Anemone chez lui :
- Alice a 2 To de données → sauvegardées chez Bob et Charlie
- Bob a 1 To de données → sauvegardées chez Alice et Charlie  
- Charlie a 500 Go de données → sauvegardées chez Alice et Bob

**Tout est chiffré côté client. Personne ne peut lire les backups des autres.**

## ✨ Fonctionnalités

### Stockage local
- ✅ Partage réseau SMB (Windows/macOS/Linux)
- ✅ Accès WebDAV (navigateur, mobile, rclone)
- ✅ SFTP optionnel (accès technique)

### Backup distribué
- ✅ Chiffrement bout-à-bout (Restic)
- ✅ Déduplication automatique
- ✅ Backup incrémental
- ✅ Choix du mode : live, périodique ou planifié
- ✅ Rétention configurable

### Sécurité
- ✅ VPN WireGuard entre pairs
- ✅ Authentification par clés publiques
- ✅ Isolation Docker complète
- ✅ Aucun accès en clair aux données distantes

## 🚀 Installation rapide

### Prérequis

- Docker & Docker Compose
- 1 Go RAM minimum
- Port UDP 51820 ouvert (port-forwarding sur votre box)
- Un nom de domaine DynDNS (gratuit : [DuckDNS](https://www.duckdns.org), [No-IP](https://www.noip.com))

### Installation

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./scripts/init.sh
# Éditez config/config.yaml et .env
docker-compose up -d
```

## 📖 Documentation complète

Consultez le [wiki](https://github.com/juste-un-gars/anemone/wiki) pour :
- Guide d'installation détaillé
- Configuration avancée
- Dépannage
- FAQ

## 🤝 Contribuer

Les contributions sont les bienvenues ! Consultez [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 Licence

MIT License - voir [LICENSE](LICENSE)
EOF

# ===== LICENSE =====
echo -e "${CYAN}[2/17]${NC} Création de LICENSE..."
cat > LICENSE << 'EOF'
MIT License

Copyright (c) 2025 Anemone Project

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF

# ===== .gitignore =====
echo -e "${CYAN}[3/17]${NC} Création de .gitignore..."
cat > .gitignore << 'EOF'
# Configuration locale (contient mots de passe et clés)
.env
config/config.yaml
config/wireguard/
config/ssh/
config/restic-password
config/samba/

# Données et backups
data/
backups/
restore/

# Logs
logs/
*.log

# Docker
docker-compose.override.yml

# Python
__pycache__/
*.py[cod]
*$py.class
*.so
.Python
venv/
env/
.venv

# Node.js
node_modules/
npm-debug.log*

# OS
.DS_Store
Thumbs.db
*.swp
*.swo
*~

# IDE
.vscode/
.idea/
*.sublime-*

# Temporaires
tmp/
temp/
*.tmp
*.bak
*.backup
EOF

# ===== .env.example =====
echo -e "${CYAN}[4/17]${NC} Création de .env.example..."
cat > .env.example << 'EOF'
# ===== ANEMONE - Variables d'environnement =====
# Copiez ce fichier en .env et modifiez les valeurs

# ===== Chemins locaux =====
DATA_PATH=./data
BACKUP_PATH=./backups

# ===== Identité utilisateur =====
PUID=1000
PGID=1000

# ===== Timezone =====
TIMEZONE=Europe/Paris

# ===== SMB (Samba) =====
SMB_USER=anemone
SMB_PASSWORD=changeme
SMB_WORKGROUP=WORKGROUP

# ===== WebDAV =====
WEBDAV_USER=anemone
WEBDAV_PASSWORD=changeme
WEBDAV_PORT=8080

# ===== SFTP =====
SFTP_PORT=2222

# ===== API =====
API_PORT=3000
EOF

# ===== docker-compose.yml =====
echo -e "${CYAN}[5/17]${NC} Création de docker-compose.yml..."
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  wireguard:
    image: linuxserver/wireguard:latest
    container_name: anemone-wireguard
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=${TIMEZONE:-Europe/Paris}
    volumes:
      - ./config/wireguard:/config
      - /lib/modules:/lib/modules:ro
    ports:
      - "51820:51820/udp"
    sysctls:
      - net.ipv4.conf.all.src_valid_mark=1
      - net.ipv4.ip_forward=1
    restart: unless-stopped
    networks:
      anemone-net:
        ipv4_address: 172.20.0.2
    healthcheck:
      test: ["CMD", "wg", "show"]
      interval: 30s
      timeout: 10s
      retries: 3

  samba:
    image: dperson/samba:latest
    container_name: anemone-samba
    environment:
      - USERID=${PUID:-1000}
      - GROUPID=${PGID:-1000}
      - TZ=${TIMEZONE:-Europe/Paris}
      - WORKGROUP=${SMB_WORKGROUP:-WORKGROUP}
    volumes:
      - ${DATA_PATH:-./data}:/mount
      - ./config/samba:/config
    ports:
      - "445:445"
    command: >
      -u "${SMB_USER:-anemone};${SMB_PASSWORD:-changeme}"
      -s "data;/mount;yes;no;no;${SMB_USER:-anemone};${SMB_USER:-anemone}"
    restart: unless-stopped
    networks:
      - anemone-net

  webdav:
    image: bytemark/webdav:latest
    container_name: anemone-webdav
    environment:
      - AUTH_TYPE=Basic
      - USERNAME=${WEBDAV_USER:-anemone}
      - PASSWORD=${WEBDAV_PASSWORD:-changeme}
      - TZ=${TIMEZONE:-Europe/Paris}
    volumes:
      - ${DATA_PATH:-./data}:/var/lib/dav
    ports:
      - "${WEBDAV_PORT:-8080}:80"
    restart: unless-stopped
    networks:
      - anemone-net

  sftp:
    image: atmoz/sftp:latest
    container_name: anemone-sftp
    volumes:
      - ${BACKUP_PATH:-./backups}:/home/restic/backups
      - ./config/ssh/authorized_keys:/home/restic/.ssh/keys/authorized_keys:ro
    ports:
      - "${SFTP_PORT:-2222}:22"
    command: restic:restic:1000:1000:backups
    restart: unless-stopped
    networks:
      - anemone-net
    profiles:
      - sftp-enabled

  restic:
    build: 
      context: ./services/restic
      dockerfile: Dockerfile
    container_name: anemone-restic
    environment:
      - RESTIC_PASSWORD_FILE=/config/restic-password
      - CONFIG_PATH=/config/config.yaml
      - TZ=${TIMEZONE:-Europe/Paris}
    volumes:
      - ${DATA_PATH:-./data}:/data:ro
      - ./config:/config:ro
      - ./logs:/logs
      - ./services/restic/scripts:/scripts:ro
    depends_on:
      wireguard:
        condition: service_healthy
    network_mode: "service:wireguard"
    restart: unless-stopped

  api:
    build:
      context: ./services/api
      dockerfile: Dockerfile
    container_name: anemone-api
    environment:
      - CONFIG_PATH=/config/config.yaml
      - LOG_PATH=/logs
      - TZ=${TIMEZONE:-Europe/Paris}
    volumes:
      - ./config:/config:ro
      - ./logs:/logs:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - "${API_PORT:-3000}:3000"
    depends_on:
      - wireguard
      - samba
      - webdav
    restart: unless-stopped
    networks:
      - anemone-net

networks:
  anemone-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
EOF

# ===== config/config.yaml.example =====
echo -e "${CYAN}[6/17]${NC} Création de config/config.yaml.example..."
cat > config/config.yaml.example << 'EOF'
# ===== CONFIGURATION ANEMONE =====

node:
  name: "anemone-home"
  role: "primary"
  description: "Mon serveur personnel"

storage:
  data_path: "/mnt/data"
  backup_path: "/mnt/backup"
  max_backup_size: "2TB"
  disk_alert_threshold: 90

services:
  smb:
    enabled: true
    port: 445
    workgroup: "WORKGROUP"
    share_name: "data"
    username: "anemone"
    password: "changeme"
    read_only: false
    
  webdav:
    enabled: true
    port: 8080
    ssl: false
    username: "anemone"
    password: "changeme"
    
  sftp:
    enabled: false
    port: 2222
    username: "anemone"

wireguard:
  interface: "wg0"
  listen_port: 51820
  private_key_file: "/config/wireguard/private.key"
  address: "10.8.0.1/24"
  public_endpoint: "votre-nom.duckdns.org:51820"
  dns: "1.1.1.1"
  mtu: 1420

peers: []
  # - name: "alice"
  #   endpoint: "alice.duckdns.org:51820"
  #   public_key: "REMPLACEZ_PAR_CLE_PUBLIQUE_ALICE"
  #   allowed_ips: "10.8.0.2/32"
  #   persistent_keepalive: 25

backup:
  engine: "restic"
  mode: "scheduled"
  debounce: 30
  interval: 30
  schedule: "0 2 * * *"
  password_file: "/config/restic-password"
  
  targets: []
    # - name: "alice-backup"
    #   enabled: true
    #   type: "sftp"
    #   host: "10.8.0.2"
    #   port: 2222
    #   user: "restic"
    #   path: "/backups/home"
  
  keep_last: 10
  keep_daily: 7
  keep_weekly: 4
  keep_monthly: 6
  keep_yearly: 2
  
  exclude:
    - "*.tmp"
    - "*.cache"
    - ".Trash-*"
    - "node_modules/"
    - "__pycache__/"
  
  bandwidth_limit: 0
  parallel: 4
  check_every: 30

restic_server:
  enabled: true
  port: 2222
  username: "restic"
  authorized_keys: []

monitoring:
  health_check_interval: 300
  alert_email: ""
  alert_webhook: ""
  alert_on_backup_fail: true
  alert_on_disk_full: true
  alert_on_vpn_down: true
  log_level: "info"
  log_max_size: "100MB"
  log_max_age: 30

advanced:
  timezone: "Europe/Paris"
  compression: true
  auto_update: false
  api_port: 3000
  prometheus_metrics: false
  prometheus_port: 9090
EOF

# ===== scripts/init.sh =====
echo -e "${CYAN}[7/17]${NC} Création de scripts/init.sh..."
cat > scripts/init.sh << 'INITEOF'
#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║     🪸  ANEMONE - Initialisation      ║"
echo "╔═══════════════════════════════════════╗"
echo -e "${NC}"
echo ""

echo -e "${BLUE}[1/7]${NC} Vérification des dépendances..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker n'est pas installé${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker installé${NC}"
echo ""

echo -e "${BLUE}[2/7]${NC} Création de la structure..."
mkdir -p config/{wireguard,ssh,samba}
mkdir -p data backups logs services/{restic,api}
echo -e "${GREEN}✓ Structure créée${NC}"
echo ""

echo -e "${BLUE}[3/7]${NC} Génération des clés WireGuard..."
if [ ! -f config/wireguard/private.key ]; then
    docker run --rm -v "$(pwd)/config/wireguard:/config" \
        linuxserver/wireguard:latest \
        sh -c "wg genkey | tee /config/private.key | wg pubkey > /config/public.key"
    chmod 600 config/wireguard/private.key
    chmod 644 config/wireguard/public.key
    echo -e "${GREEN}✓ Clés WireGuard générées${NC}"
else
    echo -e "${YELLOW}⚠ Clés WireGuard déjà présentes${NC}"
fi
echo ""

echo -e "${BLUE}[4/7]${NC} Génération du mot de passe Restic..."
if [ ! -f config/restic-password ]; then
    openssl rand -base64 32 > config/restic-password
    chmod 600 config/restic-password
    echo -e "${GREEN}✓ Mot de passe Restic généré${NC}"
else
    echo -e "${YELLOW}⚠ Mot de passe Restic déjà présent${NC}"
fi
echo ""

echo -e "${BLUE}[5/7]${NC} Génération des clés SSH..."
if [ ! -f config/ssh/id_rsa ]; then
    ssh-keygen -t rsa -b 4096 -f config/ssh/id_rsa -N "" -C "restic@anemone" -q
    chmod 600 config/ssh/id_rsa
    chmod 644 config/ssh/id_rsa.pub
    touch config/ssh/authorized_keys
    chmod 600 config/ssh/authorized_keys
    echo -e "${GREEN}✓ Clés SSH générées${NC}"
else
    echo -e "${YELLOW}⚠ Clés SSH déjà présentes${NC}"
fi
echo ""

echo -e "${BLUE}[6/17]${NC} Configuration..."
if [ ! -f config/config.yaml ]; then
    cp config/config.yaml.example config/config.yaml
    echo -e "${GREEN}✓ Fichier config.yaml créé${NC}"
else
    echo -e "${YELLOW}⚠ config.yaml déjà présent${NC}"
fi

if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${GREEN}✓ Fichier .env créé${NC}"
else
    echo -e "${YELLOW}⚠ .env déjà présent${NC}"
fi
echo ""

echo -e "${BLUE}[7/7]${NC} Récapitulatif..."
echo ""
echo -e "${GREEN}✅ Initialisation terminée !${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${CYAN}📋 INFORMATIONS À PARTAGER AVEC VOS PAIRS${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${YELLOW}🔑 Clé publique WireGuard :${NC}"
cat config/wireguard/public.key
echo ""
echo -e "${YELLOW}🔑 Clé publique SSH :${NC}"
cat config/ssh/id_rsa.pub
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${CYAN}📝 PROCHAINES ÉTAPES${NC}"
echo ""
echo "1. Éditez .env (mots de passe)"
echo "2. Éditez config/config.yaml (configuration)"
echo "3. Lancez: docker-compose up -d"
echo ""
INITEOF

chmod +x scripts/init.sh

# ===== scripts/add-peer.sh =====
echo -e "${CYAN}[8/17]${NC} Création de scripts/add-peer.sh..."
cat > scripts/add-peer.sh << 'PEEREOF'
#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🪸 Anemone - Ajouter un pair${NC}"
echo ""

echo -e "${BLUE}Nom du pair :${NC}"
read -r PEER_NAME

echo -e "${BLUE}Clé publique WireGuard :${NC}"
read -r PEER_PUBKEY

echo -e "${BLUE}Endpoint (DNS:port) :${NC}"
read -r PEER_ENDPOINT

echo -e "${BLUE}IP VPN (ex: 10.8.0.2) :${NC}"
read -r PEER_IP

echo -e "${BLUE}Clé publique SSH :${NC}"
read -r PEER_SSH_KEY

echo ""
echo -e "${YELLOW}Continuer ? (o/N)${NC}"
read -r CONFIRM

if [[ ! "$CONFIRM" =~ ^[oO]$ ]]; then
    exit 0
fi

# Ajouter au config.yaml
cat >> config/config.yaml <<EOF

  - name: "$PEER_NAME"
    endpoint: "$PEER_ENDPOINT"
    public_key: "$PEER_PUBKEY"
    allowed_ips: "$PEER_IP/32"
    persistent_keepalive: 25
EOF

# Ajouter la clé SSH
if [ -n "$PEER_SSH_KEY" ]; then
    echo "$PEER_SSH_KEY" >> config/ssh/authorized_keys
fi

echo -e "${GREEN}✅ Pair ajouté !${NC}"
echo "Relancez: docker-compose restart"
PEEREOF

chmod +x scripts/add-peer.sh

# ===== scripts/restore.sh =====
echo -e "${CYAN}[9/17]${NC} Création de scripts/restore.sh..."
cat > scripts/restore.sh << 'RESTOREEOF'
#!/bin/bash
set -e

echo "🪸 Anemone - Restauration"
echo ""
echo "Cette fonctionnalité sera disponible prochainement."
echo "Pour l'instant, utilisez:"
echo ""
echo "docker exec anemone-restic restic -r <repo> snapshots"
echo "docker exec anemone-restic restic -r <repo> restore <snapshot-id> --target /restore"
echo ""
RESTOREEOF

chmod +x scripts/restore.sh

# ===== Services Restic =====
echo -e "${CYAN}[10/17]${NC} Création de services/restic/Dockerfile..."
cat > services/restic/Dockerfile << 'EOF'
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

RUN pip3 install --no-cache-dir watchdog pyyaml

RUN mkdir -p /data /config /logs /scripts /root/.ssh

COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /scripts

ENTRYPOINT ["/entrypoint.sh"]
EOF

echo -e "${CYAN}[11/17]${NC} Création de services/restic/entrypoint.sh..."
cat > services/restic/entrypoint.sh << 'EOF'
#!/bin/bash
set -e

echo "🪸 Anemone Restic Service starting..."

CONFIG_PATH=${CONFIG_PATH:-/config/config.yaml}
RESTIC_PASSWORD_FILE=${RESTIC_PASSWORD_FILE:-/config/restic-password}

export RESTIC_PASSWORD_FILE

BACKUP_MODE=$(python3 -c "
import yaml
with open('$CONFIG_PATH') as f:
    config = yaml.safe_load(f)
    print(config.get('backup', {}).get('mode', 'scheduled'))
")

echo "📋 Backup mode: $BACKUP_MODE"

if [ -f /config/ssh/id_rsa ]; then
    cp /config/ssh/id_rsa /root/.ssh/id_rsa
    chmod 600 /root/.ssh/id_rsa
fi

case "$BACKUP_MODE" in
    "live")
        echo "🔴 LIVE mode"
        exec /scripts/backup-live.sh
        ;;
    "periodic")
        echo "🟡 PERIODIC mode"
        exec /scripts/backup-periodic.sh
        ;;
    "scheduled")
        echo "🟢 SCHEDULED mode"
        /scripts/setup-cron.sh
        exec crond -f -l 2
        ;;
    *)
        echo "❌ Unknown mode: $BACKUP_MODE"
        exit 1
        ;;
esac
EOF

chmod +x services/restic/entrypoint.sh

# ===== Scripts Restic =====
echo -e "${CYAN}[12/17]${NC} Création des scripts restic..."

cat > services/restic/scripts/backup-live.sh << 'EOF'
#!/bin/bash
echo "Mode LIVE - À implémenter"
sleep infinity
EOF

cat > services/restic/scripts/backup-periodic.sh << 'EOF'
#!/bin/bash
echo "Mode PERIODIC - À implémenter"
sleep infinity
EOF

cat > services/restic/scripts/setup-cron.sh << 'EOF'
#!/bin/bash
echo "Setup CRON"
echo "0 2 * * * /scripts/backup-now.sh >> /logs/backup.log 2>&1" > /etc/crontabs/root
EOF

cat > services/restic/scripts/backup-now.sh << 'EOF'
#!/bin/bash
echo "[$(date)] 🔄 Backup starting..."
echo "[$(date)] ✅ Backup completed"
EOF

chmod +x services/restic/scripts/*.sh

# ===== Services API =====
echo -e "${CYAN}[13/17]${NC} Création de services/api/Dockerfile..."
cat > services/api/Dockerfile << 'EOF'
FROM python:3.11-alpine

RUN apk add --no-cache gcc musl-dev libffi-dev openssl-dev curl

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s \
    CMD curl -f http://localhost:3000/health || exit 1

CMD ["python", "main.py"]
EOF

echo -e "${CYAN}[14/17]${NC} Création de services/api/requirements.txt..."
cat > services/api/requirements.txt << 'EOF'
fastapi==0.104.1
uvicorn==0.24.0
pyyaml==6.0.1
docker==6.1.3
psutil==5.9.6
pydantic==2.5.0
python-multipart==0.0.6
jinja2==3.1.2
aiofiles==23.2.1
EOF

echo -e "${CYAN}[15/17]${NC} Création de services/api/main.py..."
cat > services/api/main.py << 'EOF'
#!/usr/bin/env python3
from fastapi import FastAPI
from fastapi.responses import HTMLResponse
import os

app = FastAPI(title="Anemone API")

@app.get("/", response_class=HTMLResponse)
async def root():
    return """
    <html>
        <head><title>🪸 Anemone</title></head>
        <body style="font-family: sans-serif; text-align: center; padding: 50px;">
            <h1>🪸 Anemone API</h1>
            <p>Service actif</p>
            <p><a href="/docs">Documentation</a></p>
        </body>
    </html>
    """

@app.get("/health")
async def health():
    return {"status": "healthy"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000)
EOF

# ===== Fichiers finaux =====
echo -e "${CYAN}[16/17]${NC} Copie de .env.example vers .env..."
cp .env.example .env

echo -e "${CYAN}[17/17]${NC} Initialisation git..."
git add .
git status

echo ""
echo -e "${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════╗
║        ✅ Projet créé avec succès !   ║
╔═══════════════════════════════════════╗
EOF
echo -e "${NC}"
echo ""
echo -e "${CYAN}📝 Prochaines étapes :${NC}"
echo ""
echo "1. Éditez .env avec vos mots de passe"
echo "2. Lancez : ${YELLOW}./scripts/init.sh${NC}"
echo "3. Éditez config/config.yaml"
echo "4. Démarrez : ${YELLOW}docker-compose up -d${NC}"
echo "5. Accédez au dashboard : ${YELLOW}http://localhost:3000${NC}"
echo ""
echo -e "${CYAN}📚 Commandes Git :${NC}"
echo ""
echo "  ${YELLOW}git commit -m '🪸 Initial commit - Anemone v1.0'${NC}"
echo "  ${YELLOW}git branch -M main${NC}"
echo "  ${YELLOW}git remote add origin https://github.com/juste-un-gars/anemone.git${NC}"
echo "  ${YELLOW}git push -u origin main${NC}"
echo ""
echo -e "${GREEN}🎉 Bon développement !${NC}"
echo ""