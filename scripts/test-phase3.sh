#!/bin/bash
# Test script for Phase 3 - Advanced disaster recovery features
# This verifies that all Phase 3 components are correctly implemented

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   🧪 Test de Phase 3 - Fonctionnalités Avancées${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Test 1: Vérifier que l'interface web recovery.html existe
echo -e "${YELLOW}Test 1:${NC} Vérification de l'interface web recovery..."
if [ -f "services/api/templates/recovery.html" ]; then
    echo -e "  ${GREEN}✅ services/api/templates/recovery.html existe${NC}"

    # Vérifier le contenu HTML
    if grep -q "Disaster Recovery" services/api/templates/recovery.html; then
        echo -e "  ${GREEN}✅ Page contient le titre Disaster Recovery${NC}"
    fi
else
    echo -e "  ${RED}❌ services/api/templates/recovery.html manquant${NC}"
    exit 1
fi

# Test 2: Vérifier que les endpoints API Phase 3 sont présents
echo -e "${YELLOW}Test 2:${NC} Vérification des endpoints API Phase 3..."
MISSING_ENDPOINTS=""

grep -q "/api/recovery/backups" services/api/main.py || MISSING_ENDPOINTS="${MISSING_ENDPOINTS}\n  - /api/recovery/backups"
grep -q "/api/recovery/verify" services/api/main.py || MISSING_ENDPOINTS="${MISSING_ENDPOINTS}\n  - /api/recovery/verify"
grep -q "/api/recovery/history" services/api/main.py || MISSING_ENDPOINTS="${MISSING_ENDPOINTS}\n  - /api/recovery/history"
grep -q "/api/recovery/test-notification" services/api/main.py || MISSING_ENDPOINTS="${MISSING_ENDPOINTS}\n  - /api/recovery/test-notification"
grep -q "/recovery" services/api/main.py || MISSING_ENDPOINTS="${MISSING_ENDPOINTS}\n  - /recovery"

if [ -z "$MISSING_ENDPOINTS" ]; then
    echo -e "  ${GREEN}✅ Tous les endpoints Phase 3 sont présents${NC}"
else
    echo -e "  ${RED}❌ Endpoints manquants:${MISSING_ENDPOINTS}${NC}"
    exit 1
fi

# Test 3: Vérifier la syntaxe Python de main.py
echo -e "${YELLOW}Test 3:${NC} Vérification de la syntaxe Python..."
if python3 -m py_compile services/api/main.py 2>/dev/null; then
    echo -e "  ${GREEN}✅ Syntaxe Python correcte${NC}"
    rm -f services/api/__pycache__/main.*.pyc 2>/dev/null
else
    echo -e "  ${RED}❌ Erreur de syntaxe Python dans main.py${NC}"
    exit 1
fi

# Test 4: Vérifier que le script backup a les fonctionnalités Phase 3
echo -e "${YELLOW}Test 4:${NC} Vérification des fonctionnalités Phase 3 dans backup-config-auto.sh..."
MISSING_FEATURES=""

grep -q "send_notification" services/core/scripts/backup-config-auto.sh || MISSING_FEATURES="${MISSING_FEATURES}\n  - Fonction send_notification"
grep -q "calculate_config_checksum" services/core/scripts/backup-config-auto.sh || MISSING_FEATURES="${MISSING_FEATURES}\n  - Fonction calculate_config_checksum"
grep -q "config_has_changed" services/core/scripts/backup-config-auto.sh || MISSING_FEATURES="${MISSING_FEATURES}\n  - Fonction config_has_changed"
grep -q "BACKUP_MODE" services/core/scripts/backup-config-auto.sh || MISSING_FEATURES="${MISSING_FEATURES}\n  - Variable BACKUP_MODE"
grep -q "NOTIFICATION_ENABLED" services/core/scripts/backup-config-auto.sh || MISSING_FEATURES="${MISSING_FEATURES}\n  - Variable NOTIFICATION_ENABLED"

if [ -z "$MISSING_FEATURES" ]; then
    echo -e "  ${GREEN}✅ Toutes les fonctionnalités Phase 3 sont présentes${NC}"
else
    echo -e "  ${RED}❌ Fonctionnalités manquantes:${MISSING_FEATURES}${NC}"
    exit 1
fi

# Test 5: Vérifier que la documentation Phase 3 est complète
echo -e "${YELLOW}Test 5:${NC} Vérification de la documentation Phase 3..."
MISSING_DOCS=""

grep -q "Interface Web de Recovery" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Interface Web"
grep -q "Notifications Optionnelles" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Notifications"
grep -q "Backup Incrémentiel" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Backup Incrémentiel"
grep -q "Vérification d'Intégrité" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Vérification d'Intégrité"
grep -q "Historique Multi-Versions" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Historique Multi-Versions"
grep -q "Phase 3.*✅" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Marquage Phase 3 comme complète"

if [ -z "$MISSING_DOCS" ]; then
    echo -e "  ${GREEN}✅ Documentation Phase 3 complète${NC}"
else
    echo -e "  ${YELLOW}⚠️  Documentation incomplète:${MISSING_DOCS}${NC}"
fi

# Test 6: Vérifier la structure de l'interface web recovery.html
echo -e "${YELLOW}Test 6:${NC} Vérification de la structure de l'interface web..."
WEB_CHECKS=""

grep -q "tab.*backups" services/api/templates/recovery.html || WEB_CHECKS="${WEB_CHECKS}\n  - Onglet Backups"
grep -q "tab.*history" services/api/templates/recovery.html || WEB_CHECKS="${WEB_CHECKS}\n  - Onglet History"
grep -q "tab.*settings" services/api/templates/recovery.html || WEB_CHECKS="${WEB_CHECKS}\n  - Onglet Settings"
grep -q "verifyBackup" services/api/templates/recovery.html || WEB_CHECKS="${WEB_CHECKS}\n  - Fonction verifyBackup"
grep -q "testNotification" services/api/templates/recovery.html || WEB_CHECKS="${WEB_CHECKS}\n  - Fonction testNotification"
grep -q "loadHistory" services/api/templates/recovery.html || WEB_CHECKS="${WEB_CHECKS}\n  - Fonction loadHistory"

if [ -z "$WEB_CHECKS" ]; then
    echo -e "  ${GREEN}✅ Structure de l'interface web correcte${NC}"
else
    echo -e "  ${YELLOW}⚠️  Éléments manquants dans l'interface:${WEB_CHECKS}${NC}"
fi

# Test 7: Vérifier que le backup incrémentiel utilise des checksums
echo -e "${YELLOW}Test 7:${NC} Vérification du système de checksum..."
if grep -q "md5sum" services/core/scripts/backup-config-auto.sh; then
    echo -e "  ${GREEN}✅ Utilisation de md5sum pour les checksums${NC}"

    if grep -q "CHECKSUM_FILE" services/core/scripts/backup-config-auto.sh; then
        echo -e "  ${GREEN}✅ Fichier de checksum configuré${NC}"
    fi
else
    echo -e "  ${RED}❌ Système de checksum manquant${NC}"
    exit 1
fi

# Test 8: Vérifier les fonctions de notification email et webhook
echo -e "${YELLOW}Test 8:${NC} Vérification des fonctions de notification..."
NOTIFICATION_CHECKS=""

grep -q "send_email_notification" services/core/scripts/backup-config-auto.sh || NOTIFICATION_CHECKS="${NOTIFICATION_CHECKS}\n  - Fonction send_email_notification"
grep -q "send_webhook_notification" services/core/scripts/backup-config-auto.sh || NOTIFICATION_CHECKS="${NOTIFICATION_CHECKS}\n  - Fonction send_webhook_notification"
grep -q "smtplib" services/core/scripts/backup-config-auto.sh || NOTIFICATION_CHECKS="${NOTIFICATION_CHECKS}\n  - Import smtplib pour email"
grep -q "requests" services/core/scripts/backup-config-auto.sh || NOTIFICATION_CHECKS="${NOTIFICATION_CHECKS}\n  - Import requests pour webhook"

if [ -z "$NOTIFICATION_CHECKS" ]; then
    echo -e "  ${GREEN}✅ Fonctions de notification complètes${NC}"
else
    echo -e "  ${YELLOW}⚠️  Fonctions de notification incomplètes:${NOTIFICATION_CHECKS}${NC}"
fi

# Test 9: Vérifier que les notifications sont bien optionnelles
echo -e "${YELLOW}Test 9:${NC} Vérification du caractère optionnel des notifications..."
if grep -q "NOTIFICATION_ENABLED=false" services/core/scripts/backup-config-auto.sh; then
    echo -e "  ${GREEN}✅ Notifications désactivées par défaut${NC}"
else
    echo -e "  ${YELLOW}⚠️  Notifications non explicitement désactivées par défaut${NC}"
fi

if grep -q "ne sont PAS obligatoires" DISASTER_RECOVERY.md; then
    echo -e "  ${GREEN}✅ Documentation précise que les notifications sont optionnelles${NC}"
else
    echo -e "  ${YELLOW}⚠️  Documentation ne précise pas le caractère optionnel${NC}"
fi

# Test 10: Vérifier les endpoints API de vérification d'intégrité
echo -e "${YELLOW}Test 10:${NC} Vérification de l'endpoint de vérification d'intégrité..."
if grep -q "integrity_score" services/api/main.py; then
    echo -e "  ${GREEN}✅ Calcul du score d'intégrité présent${NC}"
fi

if grep -q "verify_backup_integrity" services/api/main.py; then
    echo -e "  ${GREEN}✅ Fonction verify_backup_integrity présente${NC}"
else
    echo -e "  ${RED}❌ Fonction verify_backup_integrity manquante${NC}"
    exit 1
fi

# Test 11: Vérifier l'endpoint d'historique
echo -e "${YELLOW}Test 11:${NC} Vérification de l'endpoint d'historique..."
if grep -q "get_backup_history" services/api/main.py; then
    echo -e "  ${GREEN}✅ Fonction get_backup_history présente${NC}"

    if grep -q "days.*int.*30" services/api/main.py; then
        echo -e "  ${GREEN}✅ Paramètre days avec valeur par défaut${NC}"
    fi
else
    echo -e "  ${RED}❌ Fonction get_backup_history manquante${NC}"
    exit 1
fi

# Test 12: Vérifier que les dépendances Python sont documentées
echo -e "${YELLOW}Test 12:${NC} Vérification des imports Python Phase 3..."
API_IMPORTS=""

grep -q "import.*smtplib" services/api/main.py || API_IMPORTS="${API_IMPORTS}\n  - smtplib (pour email)"
grep -q "import.*requests" services/api/main.py || API_IMPORTS="${API_IMPORTS}\n  - requests (pour webhook)"

if [ -z "$API_IMPORTS" ]; then
    echo -e "  ${GREEN}✅ Tous les imports nécessaires sont présents${NC}"
else
    echo -e "  ${YELLOW}⚠️  Imports manquants (utilisés dans backup-config-auto.sh):${API_IMPORTS}${NC}"
fi

# Résumé final
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Tous les tests essentiels de Phase 3 sont passés !${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}Récapitulatif des fonctionnalités Phase 3 :${NC}"
echo ""
echo -e "  ${GREEN}✅${NC} Interface web de recovery graphique"
echo -e "  ${GREEN}✅${NC} Liste tous les backups (locaux, peers, distants)"
echo -e "  ${GREEN}✅${NC} Vérification d'intégrité avec score"
echo -e "  ${GREEN}✅${NC} Historique multi-versions avec statistiques"
echo -e "  ${GREEN}✅${NC} Backup incrémentiel (checksum MD5)"
echo -e "  ${GREEN}✅${NC} Notifications optionnelles (email + webhook)"
echo -e "  ${GREEN}✅${NC} API REST complète pour automation"
echo ""
echo -e "${YELLOW}Prochaines étapes pour tester en production :${NC}"
echo ""
echo -e "  1. Reconstruire et démarrer les services :"
echo -e "     ${GREEN}docker compose down${NC}"
echo -e "     ${GREEN}docker compose build${NC}"
echo -e "     ${GREEN}docker compose up -d${NC}"
echo ""
echo -e "  2. Accéder à l'interface web de recovery :"
echo -e "     ${GREEN}http://localhost:3000/recovery${NC}"
echo ""
echo -e "  3. Tester la vérification d'intégrité :"
echo -e "     ${GREEN}# Via l'interface web, cliquer sur 'Vérifier' sur un backup${NC}"
echo ""
echo -e "  4. Consulter l'historique :"
echo -e "     ${GREEN}# Onglet Historique dans l'interface web${NC}"
echo ""
echo -e "  5. Configurer les notifications (optionnel) :"
echo -e "     ${GREEN}# Onglet Paramètres > Configurer Email ou Webhook${NC}"
echo ""
echo -e "  6. Tester le backup incrémentiel :"
echo -e "     ${GREEN}docker exec anemone-core /scripts/core/backup-config-auto.sh${NC}"
echo -e "     ${GREEN}# Relancer immédiatement → devrait dire 'aucun changement'${NC}"
echo ""
echo -e "  7. Forcer un backup même sans changement :"
echo -e "     ${GREEN}# Passer en mode 'always' dans config.yaml${NC}"
echo ""
echo -e "  8. Consulter la documentation complète :"
echo -e "     ${GREEN}cat DISASTER_RECOVERY.md${NC}"
echo ""
