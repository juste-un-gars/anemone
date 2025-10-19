#!/bin/bash

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════╗"
echo "║   🩺  ANEMONE - Diagnostic VPN        ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"
echo ""

# Fonction pour afficher une section
section() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Fonction pour afficher OK/ERROR
check_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
    fi
}

# ============================================================================
section "1. Vérification des conteneurs"
# ============================================================================

echo -e "${YELLOW}Statut du conteneur WireGuard :${NC}"
if docker ps | grep -q anemone-wireguard; then
    echo -e "${GREEN}✓ Conteneur WireGuard en cours d'exécution${NC}"
    CONTAINER_RUNNING=0
else
    echo -e "${RED}✗ Conteneur WireGuard n'est pas démarré !${NC}"
    CONTAINER_RUNNING=1
fi

# ============================================================================
section "2. Vérification de l'interface WireGuard"
# ============================================================================

if [ $CONTAINER_RUNNING -eq 0 ]; then
    echo -e "${YELLOW}Interface wg0 :${NC}"
    docker exec anemone-wireguard ip addr show wg0 2>/dev/null
    check_result $? "Interface wg0 existe"

    echo ""
    echo -e "${YELLOW}Adresse IP de wg0 :${NC}"
    WG_IP=$(docker exec anemone-wireguard ip -4 addr show wg0 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}')
    if [ -n "$WG_IP" ]; then
        echo -e "${GREEN}✓ IP VPN locale : $WG_IP${NC}"
    else
        echo -e "${RED}✗ Aucune IP sur wg0${NC}"
    fi
fi

# ============================================================================
section "3. Statut WireGuard"
# ============================================================================

if [ $CONTAINER_RUNNING -eq 0 ]; then
    echo -e "${YELLOW}Configuration WireGuard actuelle :${NC}"
    docker exec anemone-wireguard wg show 2>/dev/null || {
        echo -e "${RED}✗ Impossible d'obtenir le statut WireGuard${NC}"
    }
fi

# ============================================================================
section "4. Fichiers de configuration"
# ============================================================================

echo -e "${YELLOW}Clé publique locale :${NC}"
if [ -f config/wireguard/public.key ]; then
    LOCAL_PUBKEY=$(cat config/wireguard/public.key)
    echo -e "${GREEN}$LOCAL_PUBKEY${NC}"

    # Vérifier si c'est un placeholder
    if echo "$LOCAL_PUBKEY" | grep -q "# Clé publique"; then
        echo -e "${RED}⚠ ATTENTION : Clé publique est un placeholder !${NC}"
        echo -e "${YELLOW}Essayez : ./scripts/extract-wireguard-pubkey.sh${NC}"
    fi
else
    echo -e "${RED}✗ Fichier config/wireguard/public.key manquant${NC}"
fi

echo ""
echo -e "${YELLOW}Fichier wg0.conf :${NC}"
if [ -f config/wg_confs/wg0.conf ]; then
    echo -e "${GREEN}✓ config/wg_confs/wg0.conf existe${NC}"
    echo ""
    echo -e "${YELLOW}Contenu de wg0.conf :${NC}"
    echo "─────────────────────────────────────────────────"
    cat config/wg_confs/wg0.conf
    echo "─────────────────────────────────────────────────"

    # Compter les peers
    PEER_COUNT=$(grep -c "^\[Peer\]" config/wg_confs/wg0.conf || echo "0")
    echo ""
    echo -e "${CYAN}Nombre de peers configurés : ${PEER_COUNT}${NC}"

    # Vérifier que la clé privée dans wg0.conf correspond
    if [ -f config/wireguard/private.key ]; then
        WG0_PRIVKEY=$(grep "^PrivateKey" config/wg_confs/wg0.conf | awk '{print $3}')
        FILE_PRIVKEY=$(cat config/wireguard/private.key)

        if [ "$WG0_PRIVKEY" = "$FILE_PRIVKEY" ]; then
            echo -e "${GREEN}✓ Clé privée dans wg0.conf correspond${NC}"
        else
            echo -e "${RED}✗ ERREUR : Clé privée dans wg0.conf ne correspond pas !${NC}"
            echo -e "${YELLOW}  → wg0.conf : ${WG0_PRIVKEY:0:20}...${NC}"
            echo -e "${YELLOW}  → private.key : ${FILE_PRIVKEY:0:20}...${NC}"
        fi
    fi
else
    echo -e "${RED}✗ config/wg_confs/wg0.conf MANQUANT !${NC}"
    echo -e "${YELLOW}Ceci est probablement la cause du problème.${NC}"
    echo -e "${YELLOW}Lancez : ./scripts/init.sh${NC}"
fi

echo ""
echo -e "${YELLOW}Fichier config.yaml (section peers) :${NC}"
if [ -f config/config.yaml ]; then
    echo "─────────────────────────────────────────────────"
    sed -n '/^peers:/,/^backup:/p' config/config.yaml | head -n -1
    echo "─────────────────────────────────────────────────"
else
    echo -e "${RED}✗ config/config.yaml manquant${NC}"
fi

# ============================================================================
section "5. Test de connectivité locale"
# ============================================================================

if [ $CONTAINER_RUNNING -eq 0 ] && [ -n "$WG_IP" ]; then
    echo -e "${YELLOW}Ping de l'IP VPN locale :${NC}"
    docker exec anemone-wireguard ping -c 2 $WG_IP 2>/dev/null
    check_result $? "Ping de $WG_IP"
fi

# ============================================================================
section "6. Logs récents WireGuard"
# ============================================================================

if [ $CONTAINER_RUNNING -eq 0 ]; then
    echo -e "${YELLOW}20 dernières lignes des logs :${NC}"
    echo "─────────────────────────────────────────────────"
    docker logs anemone-wireguard --tail 20 2>&1
    echo "─────────────────────────────────────────────────"
fi

# ============================================================================
section "7. Recommandations"
# ============================================================================

echo ""
echo -e "${CYAN}Problèmes détectés et solutions :${NC}"
echo ""

# Vérifier les problèmes communs
ISSUES_FOUND=0

if [ ! -f config/wg_confs/wg0.conf ]; then
    ISSUES_FOUND=1
    echo -e "${RED}❌ PROBLÈME : wg0.conf manquant${NC}"
    echo -e "   ${YELLOW}Solution : ./scripts/init.sh${NC}"
    echo ""
fi

if [ -f config/wireguard/public.key ] && grep -q "# Clé publique" config/wireguard/public.key; then
    ISSUES_FOUND=1
    echo -e "${RED}❌ PROBLÈME : Clé publique est un placeholder${NC}"
    echo -e "   ${YELLOW}Solution : ./scripts/extract-wireguard-pubkey.sh${NC}"
    echo ""
fi

if [ -f config/wg_confs/wg0.conf ]; then
    PEER_COUNT=$(grep -c "^\[Peer\]" config/wg_confs/wg0.conf || echo "0")
    if [ "$PEER_COUNT" -eq 0 ]; then
        ISSUES_FOUND=1
        echo -e "${RED}❌ PROBLÈME : Aucun peer dans wg0.conf${NC}"
        echo -e "   ${YELLOW}Solution : ./scripts/add-peer.sh${NC}"
        echo -e "   ${YELLOW}Puis : docker-compose restart wireguard${NC}"
        echo ""
    fi
fi

if [ $CONTAINER_RUNNING -ne 0 ]; then
    ISSUES_FOUND=1
    echo -e "${RED}❌ PROBLÈME : Conteneur WireGuard non démarré${NC}"
    echo -e "   ${YELLOW}Solution : docker-compose up -d wireguard${NC}"
    echo ""
fi

if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✅ Aucun problème évident détecté${NC}"
    echo ""
    echo -e "${YELLOW}Si le VPN ne fonctionne toujours pas :${NC}"
    echo "  1. Vérifiez que l'autre serveur a aussi votre clé publique"
    echo "  2. Vérifiez que les endpoints sont accessibles (firewall/port forwarding)"
    echo "  3. Vérifiez que les deux serveurs ont redémarré WireGuard après ajout des peers"
    echo "  4. Essayez : docker-compose restart wireguard"
fi

echo ""
echo -e "${CYAN}Pour tester la connectivité vers un peer :${NC}"
echo -e "  ${YELLOW}docker exec anemone-restic ping 10.8.0.X${NC}"
echo ""
