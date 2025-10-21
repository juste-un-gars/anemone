#!/bin/bash
# Script de backup automatique de la configuration
# Exporte la configuration chiffrée et la distribue vers tous les peers
# Appelé quotidiennement via cron

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="/config/config.yaml"
BACKUP_DIR="/config-backups/local"
API_URL="http://api:3000/api/config/export"

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

# Vérifier que la configuration existe
if [ ! -f "$CONFIG_FILE" ]; then
    log_error "Configuration file not found: $CONFIG_FILE"
    exit 1
fi

# Créer le répertoire de backup local s'il n'existe pas
mkdir -p "$BACKUP_DIR"

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
exit 0
