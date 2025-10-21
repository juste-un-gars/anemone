#!/bin/bash
# Script pour ajouter une clé SSH publique d'un peer
# Usage: ./scripts/add-ssh-key.sh

set -e

echo "════════════════════════════════════════════════════════"
echo "  🔑 Ajout de clé SSH d'un peer"
echo "════════════════════════════════════════════════════════"
echo ""

# Méthode 1 : Copier-coller la clé
echo "Collez la clé SSH publique du peer (commence par 'ssh-rsa'):"
echo "(Vous pouvez l'obtenir sur l'autre serveur avec: cat config/ssh/id_rsa.pub)"
echo ""
read -r SSH_KEY

if [[ ! "$SSH_KEY" =~ ^ssh-(rsa|ed25519) ]]; then
    echo "❌ Erreur: Ceci ne ressemble pas à une clé SSH publique"
    echo "   Elle devrait commencer par 'ssh-rsa' ou 'ssh-ed25519'"
    exit 1
fi

# Ajouter la clé aux authorized_keys
echo ""
echo "🔐 Ajout de la clé aux authorized_keys..."

docker exec anemone-core sh -c "
    echo '$SSH_KEY' >> /home/restic/.ssh/authorized_keys
    chmod 600 /home/restic/.ssh/authorized_keys
    chown restic:restic /home/restic/.ssh/authorized_keys
"

echo "✅ Clé ajoutée avec succès !"
echo ""
echo "📋 Clés actuellement autorisées:"
docker exec anemone-core wc -l /home/restic/.ssh/authorized_keys

echo ""
echo "✅ Terminé ! Le peer devrait maintenant pouvoir se connecter."
echo ""
echo "💡 Pour tester la connexion depuis l'autre serveur:"
echo "   docker exec anemone-core sftp -o StrictHostKeyChecking=no restic@VOTRE_IP <<< 'ls /backups'"
