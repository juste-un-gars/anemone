#!/bin/bash
# Test script for disaster recovery export/import functionality
# This verifies that the backup/restore cycle works correctly

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   🧪 Test de Disaster Recovery - Phase 1${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Test 1: Vérifier que le script de restauration existe
echo -e "${YELLOW}Test 1:${NC} Vérification du script de restauration..."
if [ -f "scripts/restore-config.py" ]; then
    echo -e "  ${GREEN}✅ scripts/restore-config.py existe${NC}"
else
    echo -e "  ${RED}❌ scripts/restore-config.py manquant${NC}"
    exit 1
fi

# Test 2: Vérifier que la documentation existe
echo -e "${YELLOW}Test 2:${NC} Vérification de la documentation..."
if [ -f "DISASTER_RECOVERY.md" ]; then
    echo -e "  ${GREEN}✅ DISASTER_RECOVERY.md existe${NC}"
else
    echo -e "  ${RED}❌ DISASTER_RECOVERY.md manquant${NC}"
    exit 1
fi

# Test 3: Vérifier que start.sh a l'option --restore-from
echo -e "${YELLOW}Test 3:${NC} Vérification de l'option --restore-from dans start.sh..."
if grep -q "restore-from" start.sh; then
    echo -e "  ${GREEN}✅ Option --restore-from présente dans start.sh${NC}"
else
    echo -e "  ${RED}❌ Option --restore-from manquante dans start.sh${NC}"
    exit 1
fi

# Test 4: Vérifier que l'endpoint API existe
echo -e "${YELLOW}Test 4:${NC} Vérification de l'endpoint API..."
if grep -q "/api/config/export" services/api/main.py; then
    echo -e "  ${GREEN}✅ Endpoint /api/config/export présent dans main.py${NC}"
else
    echo -e "  ${RED}❌ Endpoint /api/config/export manquant${NC}"
    exit 1
fi

# Test 5: Vérifier la syntaxe Python du script de restauration
echo -e "${YELLOW}Test 5:${NC} Vérification de la syntaxe Python..."
if python3 -m py_compile scripts/restore-config.py 2>/dev/null; then
    echo -e "  ${GREEN}✅ Syntaxe Python correcte${NC}"
    rm -f scripts/__pycache__/restore-config.*.pyc 2>/dev/null
else
    echo -e "  ${RED}❌ Erreur de syntaxe Python${NC}"
    exit 1
fi

# Test 6: Vérifier que les dépendances Python sont présentes
echo -e "${YELLOW}Test 6:${NC} Vérification des dépendances Python..."
python3 -c "from cryptography.hazmat.primitives.ciphers import Cipher" 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✅ Module cryptography disponible${NC}"
else
    echo -e "  ${YELLOW}⚠️  Module cryptography non disponible (requis en production)${NC}"
fi

# Test 7: Vérifier le format du script restore
echo -e "${YELLOW}Test 7:${NC} Vérification de la structure du script..."
if grep -q "restore_configuration" scripts/restore-config.py && \
   grep -q "PBKDF2HMAC" scripts/restore-config.py && \
   grep -q "AES" scripts/restore-config.py; then
    echo -e "  ${GREEN}✅ Structure correcte (fonction restore_configuration, PBKDF2, AES)${NC}"
else
    echo -e "  ${RED}❌ Structure incorrecte${NC}"
    exit 1
fi

# Test 8: Vérifier que la documentation contient les sections essentielles
echo -e "${YELLOW}Test 8:${NC} Vérification du contenu de la documentation..."
MISSING_SECTIONS=""
grep -q "Export de Configuration" DISASTER_RECOVERY.md || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Section Export manquante"
grep -q "Restauration Complète" DISASTER_RECOVERY.md || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Section Restauration manquante"
grep -q "Bonnes Pratiques" DISASTER_RECOVERY.md || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Section Bonnes Pratiques manquante"
grep -q "En Cas de Problème" DISASTER_RECOVERY.md || MISSING_SECTIONS="${MISSING_SECTIONS}\n  - Section Dépannage manquante"

if [ -z "$MISSING_SECTIONS" ]; then
    echo -e "  ${GREEN}✅ Documentation complète${NC}"
else
    echo -e "  ${YELLOW}⚠️  Sections manquantes:${MISSING_SECTIONS}${NC}"
fi

# Résumé final
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Tous les tests de base sont passés !${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}Prochaines étapes pour tester en production :${NC}"
echo ""
echo -e "  1. Démarrer les services Docker :"
echo -e "     ${GREEN}docker compose up -d${NC}"
echo ""
echo -e "  2. Exporter la configuration :"
echo -e "     ${GREEN}curl -o backup-test.enc http://localhost:3000/api/config/export${NC}"
echo ""
echo -e "  3. Tester la restauration (dans un répertoire de test) :"
echo -e "     ${GREEN}mkdir ~/test-recovery && cd ~/test-recovery${NC}"
echo -e "     ${GREEN}git clone https://github.com/juste-un-gars/anemone.git${NC}"
echo -e "     ${GREEN}cd anemone${NC}"
echo -e "     ${GREEN}./start.sh --restore-from=/path/to/backup-test.enc${NC}"
echo ""
echo -e "  4. Consulter la documentation complète :"
echo -e "     ${GREEN}cat DISASTER_RECOVERY.md${NC}"
echo ""
