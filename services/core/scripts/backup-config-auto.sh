#!/bin/bash
# Script de backup automatique de la configuration
# Exporte la configuration chiffrée et la distribue vers tous les peers
# Appelé quotidiennement via cron
# Phase 3: Backup incrémentiel + notifications optionnelles

set -e

# Trap pour capturer les erreurs et envoyer des notifications
trap 'handle_error $? $LINENO' ERR

handle_error() {
    local exit_code=$1
    local line_number=$2
    send_notification "error" "Backup failed at line $line_number with exit code $exit_code"
    exit $exit_code
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="/config/config.yaml"
BACKUP_DIR="/config-backups/local"
API_URL="http://api:3000/api/config/export"
CHECKSUM_FILE="/config-backups/.last-checksum"
NOTIFICATION_ENABLED=false
NOTIFICATION_TYPE=""
BACKUP_MODE="incremental" # incremental ou always

# Couleurs pour les logs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ❌ $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  $1${NC}"
}

# Charger la configuration des notifications (optionnel)
load_notification_config() {
    if [ -f "$CONFIG_FILE" ]; then
        NOTIFICATION_ENABLED=$(python3 -c "
import yaml
try:
    with open('$CONFIG_FILE', 'r') as f:
        config = yaml.safe_load(f)
        notifications = config.get('backup', {}).get('notifications', {})
        print('true' if notifications.get('enabled', False) else 'false')
except:
    print('false')
" 2>/dev/null || echo "false")

        if [ "$NOTIFICATION_ENABLED" = "true" ]; then
            NOTIFICATION_TYPE=$(python3 -c "
import yaml
with open('$CONFIG_FILE', 'r') as f:
    config = yaml.safe_load(f)
    notifications = config.get('backup', {}).get('notifications', {})
    print(notifications.get('type', ''))
" 2>/dev/null || echo "")
        fi

        # Charger le mode de backup (incremental par défaut)
        BACKUP_MODE=$(python3 -c "
import yaml
try:
    with open('$CONFIG_FILE', 'r') as f:
        config = yaml.safe_load(f)
        print(config.get('backup', {}).get('mode', 'incremental'))
except:
    print('incremental')
" 2>/dev/null || echo "incremental")
    fi
}

# Envoyer une notification (si configuré)
send_notification() {
    local status="$1"  # success ou error
    local message="$2"

    if [ "$NOTIFICATION_ENABLED" != "true" ]; then
        return 0
    fi

    if [ "$NOTIFICATION_TYPE" = "email" ]; then
        send_email_notification "$status" "$message"
    elif [ "$NOTIFICATION_TYPE" = "webhook" ]; then
        send_webhook_notification "$status" "$message"
    fi
}

send_email_notification() {
    local status="$1"
    local message="$2"

    # Cette fonction sera appelée via Python pour plus de flexibilité
    python3 -c "
import smtplib
import yaml
from email.mime.text import MIMEText
from datetime import datetime

try:
    with open('$CONFIG_FILE', 'r') as f:
        config = yaml.safe_load(f)
        notifications = config.get('backup', {}).get('notifications', {})
        email_config = notifications.get('email', {})

    subject = 'Anemone Backup - $status'
    body = '''
Statut du backup: $status
Date: ''' + datetime.now().strftime('%Y-%m-%d %H:%M:%S') + '''
Serveur: ''' + config.get('server', {}).get('name', 'unknown') + '''

Message:
$message
'''

    msg = MIMEText(body)
    msg['Subject'] = subject
    msg['From'] = email_config.get('smtp_user', '')
    msg['To'] = email_config.get('to_email', '')

    with smtplib.SMTP(email_config.get('smtp_server', ''), email_config.get('smtp_port', 587)) as server:
        server.starttls()
        server.login(email_config.get('smtp_user', ''), email_config.get('smtp_password', ''))
        server.send_message(msg)

    print('Email sent successfully')
except Exception as e:
    print(f'Failed to send email: {e}')
" 2>&1
}

send_webhook_notification() {
    local status="$1"
    local message="$2"

    python3 -c "
import requests
import yaml
import json
from datetime import datetime

try:
    with open('$CONFIG_FILE', 'r') as f:
        config = yaml.safe_load(f)
        notifications = config.get('backup', {}).get('notifications', {})
        webhook_url = notifications.get('webhook', {}).get('url', '')

    if not webhook_url:
        exit(0)

    payload = {
        'event': 'backup_$status',
        'status': '$status',
        'message': '$message',
        'timestamp': datetime.now().isoformat(),
        'server': config.get('server', {}).get('name', 'unknown')
    }

    response = requests.post(webhook_url, json=payload, timeout=10)
    response.raise_for_status()
    print('Webhook sent successfully')
except Exception as e:
    print(f'Failed to send webhook: {e}')
" 2>&1
}

# Calculer le checksum de la configuration
calculate_config_checksum() {
    # Checksum de tous les fichiers de configuration importants
    local checksum=""

    if [ -f "/config/config.yaml" ]; then
        checksum+=$(md5sum /config/config.yaml | cut -d' ' -f1)
    fi

    if [ -f "/config/wireguard/private.key" ]; then
        checksum+=$(md5sum /config/wireguard/private.key | cut -d' ' -f1)
    fi

    if [ -f "/config/wireguard/public.key" ]; then
        checksum+=$(md5sum /config/wireguard/public.key | cut -d' ' -f1)
    fi

    if [ -f "/config/ssh/id_rsa" ]; then
        checksum+=$(md5sum /config/ssh/id_rsa | cut -d' ' -f1)
    fi

    if [ -f "/config/ssh/id_rsa.pub" ]; then
        checksum+=$(md5sum /config/ssh/id_rsa.pub | cut -d' ' -f1)
    fi

    echo -n "$checksum" | md5sum | cut -d' ' -f1
}

# Vérifier si la configuration a changé
config_has_changed() {
    local current_checksum=$(calculate_config_checksum)

    if [ ! -f "$CHECKSUM_FILE" ]; then
        # Première exécution, pas de checksum précédent
        echo "$current_checksum" > "$CHECKSUM_FILE"
        return 0  # Considérer comme changé
    fi

    local last_checksum=$(cat "$CHECKSUM_FILE")

    if [ "$current_checksum" != "$last_checksum" ]; then
        echo "$current_checksum" > "$CHECKSUM_FILE"
        return 0  # Changé
    fi

    return 1  # Pas de changement
}

# Charger la configuration des notifications
load_notification_config

# Vérifier que la configuration existe
if [ ! -f "$CONFIG_FILE" ]; then
    log_error "Configuration file not found: $CONFIG_FILE"
    send_notification "error" "Configuration file not found"
    exit 1
fi

# Créer le répertoire de backup local s'il n'existe pas
mkdir -p "$BACKUP_DIR"

# Phase 3: Backup incrémentiel - vérifier si la configuration a changé
if [ "$BACKUP_MODE" = "incremental" ]; then
    if ! config_has_changed; then
        log "⏭️  Backup incrémentiel : aucun changement détecté"
        log "   Configuration inchangée depuis le dernier backup"
        exit 0
    fi
    log "🔄 Backup incrémentiel : changements détectés"
fi

# Extraire le nom du serveur depuis la configuration
HOSTNAME=$(python3 -c "
import yaml
with open('$CONFIG_FILE', 'r') as f:
    config = yaml.safe_load(f)
    print(config.get('server', {}).get('name', 'unknown'))
" 2>/dev/null || echo "unknown")

# Générer le nom du fichier de backup
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/anemone-backup-${HOSTNAME}-${TIMESTAMP}.enc"

log "📦 Démarrage du backup automatique de configuration..."
log "   Serveur: ${HOSTNAME}"
log "   Fichier: ${BACKUP_FILE}"

# Exporter la configuration via l'API
log "🔐 Export de la configuration chiffrée..."
if curl -f -s -o "$BACKUP_FILE" "$API_URL" 2>/dev/null; then
    log_success "Configuration exportée: $(du -h "$BACKUP_FILE" | cut -f1)"
else
    log_error "Échec de l'export de configuration"
    exit 1
fi

# Vérifier que le fichier a été créé
if [ ! -f "$BACKUP_FILE" ]; then
    log_error "Le fichier de backup n'a pas été créé"
    exit 1
fi

# Vérifier la taille (doit être > 0)
FILE_SIZE=$(stat -c%s "$BACKUP_FILE" 2>/dev/null || echo "0")
if [ "$FILE_SIZE" -eq 0 ]; then
    log_error "Le fichier de backup est vide"
    rm -f "$BACKUP_FILE"
    exit 1
fi

log_success "Backup local créé avec succès"

# Distribuer vers les peers
log "🌐 Distribution vers les peers..."

# Lire la liste des peers depuis config.yaml
PEERS=$(python3 -c "
import yaml
with open('$CONFIG_FILE', 'r') as f:
    config = yaml.safe_load(f)
    peers = config.get('peers', [])
    for peer in peers:
        if peer.get('enabled', True):
            vpn_ip = peer.get('vpn_ip', '')
            name = peer.get('name', 'unknown')
            print(f'{vpn_ip}:{name}')
" 2>/dev/null)

if [ -z "$PEERS" ]; then
    log_warning "Aucun peer configuré, backup local uniquement"
    exit 0
fi

# Uploader vers chaque peer
SUCCESS_COUNT=0
FAIL_COUNT=0

while IFS=':' read -r VPN_IP PEER_NAME; do
    if [ -z "$VPN_IP" ]; then
        continue
    fi

    log "   → Envoi vers ${PEER_NAME} (${VPN_IP})..."

    # Créer le répertoire distant si nécessaire
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i /root/.ssh/id_rsa "restic@${VPN_IP}" \
        "mkdir -p /config-backups/${HOSTNAME}" 2>/dev/null || true

    # Uploader le fichier via SFTP
    if echo "put ${BACKUP_FILE} /config-backups/${HOSTNAME}/$(basename ${BACKUP_FILE})" | \
       sftp -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i /root/.ssh/id_rsa \
       "restic@${VPN_IP}" >/dev/null 2>&1; then
        log_success "   ✓ ${PEER_NAME}"
        ((SUCCESS_COUNT++))
    else
        log_error "   ✗ ${PEER_NAME} (échec de connexion)"
        ((FAIL_COUNT++))
    fi
done <<< "$PEERS"

# Résumé
log ""
log "📊 Résumé du backup:"
log "   Local: ${BACKUP_FILE}"
log "   Distribué vers: ${SUCCESS_COUNT} peer(s)"
if [ $FAIL_COUNT -gt 0 ]; then
    log_warning "   Échecs: ${FAIL_COUNT} peer(s)"
fi

# Nettoyer les anciens backups locaux (garder les 7 derniers)
log "🧹 Nettoyage des anciens backups..."
cd "$BACKUP_DIR"
ls -t anemone-backup-${HOSTNAME}-*.enc 2>/dev/null | tail -n +8 | xargs -r rm -f
REMAINING=$(ls -1 anemone-backup-${HOSTNAME}-*.enc 2>/dev/null | wc -l)
log "   Backups locaux conservés: ${REMAINING}"

log_success "Backup automatique terminé"

# Envoyer notification de succès (Phase 3)
if [ $FAIL_COUNT -gt 0 ]; then
    send_notification "warning" "Backup completed with warnings: ${SUCCESS_COUNT} peers succeeded, ${FAIL_COUNT} peers failed"
else
    send_notification "success" "Backup completed successfully: distributed to ${SUCCESS_COUNT} peer(s)"
fi

exit 0
