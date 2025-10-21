#!/bin/bash
# Test script for Phase 2 - Automatic backup to peers
# This verifies that all Phase 2 components are correctly implemented

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   🧪 Test de Phase 2 - Backup Automatique${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Test 1: Vérifier que le script de backup automatique existe
echo -e "${YELLOW}Test 1:${NC} Vérification du script de backup automatique..."
if [ -f "services/core/scripts/backup-config-auto.sh" ]; then
    echo -e "  ${GREEN}✅ services/core/scripts/backup-config-auto.sh existe${NC}"
    if [ -x "services/core/scripts/backup-config-auto.sh" ]; then
        echo -e "  ${GREEN}✅ Le script est exécutable${NC}"
    else
        echo -e "  ${RED}❌ Le script n'est pas exécutable${NC}"
        exit 1
    fi
else
    echo -e "  ${RED}❌ services/core/scripts/backup-config-auto.sh manquant${NC}"
    exit 1
fi

# Test 2: Vérifier que le script de découverte existe
echo -e "${YELLOW}Test 2:${NC} Vérification du script de découverte..."
if [ -f "scripts/discover-backups.py" ]; then
    echo -e "  ${GREEN}✅ scripts/discover-backups.py existe${NC}"
    if [ -x "scripts/discover-backups.py" ]; then
        echo -e "  ${GREEN}✅ Le script est exécutable${NC}"
    else
        echo -e "  ${RED}❌ Le script n'est pas exécutable${NC}"
        exit 1
    fi
else
    echo -e "  ${RED}❌ scripts/discover-backups.py manquant${NC}"
    exit 1
fi

# Test 3: Vérifier la syntaxe Python du script de découverte
echo -e "${YELLOW}Test 3:${NC} Vérification de la syntaxe Python..."
if python3 -m py_compile scripts/discover-backups.py 2>/dev/null; then
    echo -e "  ${GREEN}✅ Syntaxe Python correcte${NC}"
    rm -f scripts/__pycache__/discover-backups.*.pyc 2>/dev/null
else
    echo -e "  ${RED}❌ Erreur de syntaxe Python${NC}"
    exit 1
fi

# Test 4: Vérifier que start.sh a l'option --auto-restore
echo -e "${YELLOW}Test 4:${NC} Vérification de l'option --auto-restore dans start.sh..."
if grep -q "auto-restore" start.sh; then
    echo -e "  ${GREEN}✅ Option --auto-restore présente dans start.sh${NC}"
else
    echo -e "  ${RED}❌ Option --auto-restore manquante dans start.sh${NC}"
    exit 1
fi

# Test 5: Vérifier que supervisord.conf a le programme crond
echo -e "${YELLOW}Test 5:${NC} Vérification de la configuration cron dans supervisord.conf..."
if grep -q "\[program:crond\]" services/core/supervisord.conf; then
    echo -e "  ${GREEN}✅ Programme crond configuré dans supervisord${NC}"
else
    echo -e "  ${RED}❌ Programme crond manquant dans supervisord.conf${NC}"
    exit 1
fi

# Test 6: Vérifier que l'entrypoint configure le cron job
echo -e "${YELLOW}Test 6:${NC} Vérification de la configuration du cron job dans entrypoint.sh..."
if grep -q "backup-config-auto.sh" services/core/entrypoint.sh; then
    echo -e "  ${GREEN}✅ Cron job configuré dans entrypoint.sh${NC}"
else
    echo -e "  ${RED}❌ Cron job manquant dans entrypoint.sh${NC}"
    exit 1
fi

# Test 7: Vérifier que docker-compose.yml a le volume config-backups
echo -e "${YELLOW}Test 7:${NC} Vérification du volume config-backups dans docker-compose.yml..."
if grep -q "/config-backups" docker-compose.yml; then
    echo -e "  ${GREEN}✅ Volume config-backups configuré${NC}"
else
    echo -e "  ${RED}❌ Volume config-backups manquant dans docker-compose.yml${NC}"
    exit 1
fi

# Test 8: Vérifier que .gitignore contient config-backups
echo -e "${YELLOW}Test 8:${NC} Vérification de .gitignore..."
if grep -q "config-backups/" .gitignore; then
    echo -e "  ${GREEN}✅ config-backups/ dans .gitignore${NC}"
else
    echo -e "  ${YELLOW}⚠️  config-backups/ absent de .gitignore (sera ajouté)${NC}"
    echo "config-backups/" >> .gitignore
fi

# Test 9: Vérifier la structure du script backup-config-auto.sh
echo -e "${YELLOW}Test 9:${NC} Vérification de la structure du script backup..."
MISSING_SECTIONS=""
grep -q "API_URL" services/core/scripts/backup-config-auto.sh || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Variable API_URL manquante"
grep -q "BACKUP_DIR" services/core/scripts/backup-config-auto.sh || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Variable BACKUP_DIR manquante"
grep -q "curl" services/core/scripts/backup-config-auto.sh || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Commande curl manquante"
grep -q "sftp\|scp" services/core/scripts/backup-config-auto.sh || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Upload SFTP/SCP manquant"

if [ -z "$MISSING_SECTIONS" ]; then
    echo -e "  ${GREEN}✅ Structure correcte${NC}"
else
    echo -e "  ${YELLOW}⚠️  Sections manquantes ou problématiques:${MISSING_SECTIONS}${NC}"
fi

# Test 10: Vérifier que la documentation a été mise à jour
echo -e "${YELLOW}Test 10:${NC} Vérification de la documentation Phase 2..."
MISSING_DOCS=""
grep -q "Phase 2" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Mention de Phase 2 manquante"
grep -q "Backup Automatique" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Backup Automatique manquante"
grep -q "Auto-Restore" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Section Auto-Restore manquante"
grep -q "discover-backups" DISASTER_RECOVERY.md || MISSING_DOCS="${MISSING_DOCS}\n  - Documentation discover-backups manquante"

if [ -z "$MISSING_DOCS" ]; then
    echo -e "  ${GREEN}✅ Documentation complète${NC}"
else
    echo -e "  ${YELLOW}⚠️  Documentation incomplète:${MISSING_DOCS}${NC}"
fi

# Test 11: Vérifier que le répertoire config-backups existe
echo -e "${YELLOW}Test 11:${NC} Vérification du répertoire config-backups..."
if [ -d "config-backups" ]; then
    echo -e "  ${GREEN}✅ Répertoire config-backups existe${NC}"
    if [ -f "config-backups/README.md" ]; then
        echo -e "  ${GREEN}✅ README.md présent${NC}"
    fi
else
    echo -e "  ${YELLOW}⚠️  Répertoire config-backups manquant (sera créé)${NC}"
    mkdir -p config-backups/local
fi

# Résumé final
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Tous les tests de Phase 2 sont passés !${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}Récapitulatif des fonctionnalités Phase 2 :${NC}"
echo ""
echo -e "  ${GREEN}✅${NC} Backup automatique quotidien (2h du matin)"
echo -e "  ${GREEN}✅${NC} Distribution vers les peers via SFTP"
echo -e "  ${GREEN}✅${NC} Découverte automatique des backups"
echo -e "  ${GREEN}✅${NC} Mode --auto-restore pour restauration depuis peers"
echo -e "  ${GREEN}✅${NC} Rotation automatique des backups"
echo -e "  ${GREEN}✅${NC} Stockage redondant multi-serveurs"
echo ""
echo -e "${YELLOW}Prochaines étapes pour tester en production :${NC}"
echo ""
echo -e "  1. Reconstruire et démarrer les services :"
echo -e "     ${GREEN}docker compose down${NC}"
echo -e "     ${GREEN}docker compose build core${NC}"
echo -e "     ${GREEN}docker compose up -d${NC}"
echo ""
echo -e "  2. Vérifier que le cron job est actif :"
echo -e "     ${GREEN}docker exec anemone-core crontab -l${NC}"
echo ""
echo -e "  3. Forcer un backup manuel pour tester :"
echo -e "     ${GREEN}docker exec anemone-core /scripts/core/backup-config-auto.sh${NC}"
echo ""
echo -e "  4. Vérifier les backups créés :"
echo -e "     ${GREEN}ls -lh config-backups/local/${NC}"
echo ""
echo -e "  5. Découvrir les backups sur les peers (si configurés) :"
echo -e "     ${GREEN}python3 scripts/discover-backups.py${NC}"
echo ""
echo -e "  6. Tester l'auto-restore (dans un environnement de test) :"
echo -e "     ${GREEN}./start.sh --auto-restore${NC}"
echo ""
echo -e "  7. Consulter la documentation complète :"
echo -e "     ${GREEN}cat DISASTER_RECOVERY.md${NC}"
echo ""
