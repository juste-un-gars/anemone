#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

set -e

# Configuration
TRASH_DIR="/mnt/backup/.trash"
MAX_SIZE_GB=${TRASH_MAX_SIZE_GB:-10}
MAX_SIZE_BYTES=$((MAX_SIZE_GB * 1024 * 1024 * 1024))

# Vérifier que le répertoire corbeille existe
if [ ! -d "$TRASH_DIR" ]; then
    # Pas de corbeille = rien à nettoyer
    exit 0
fi

# Calculer la taille actuelle de la corbeille
CURRENT_SIZE=$(du -sb "$TRASH_DIR" 2>/dev/null | awk '{print $1}')

if [ -z "$CURRENT_SIZE" ] || [ "$CURRENT_SIZE" -eq 0 ]; then
    # Corbeille vide
    exit 0
fi

# Vérifier si on dépasse la limite
if [ "$CURRENT_SIZE" -le "$MAX_SIZE_BYTES" ]; then
    echo "🗑️  Trash size: $(numfmt --to=iec-i --suffix=B $CURRENT_SIZE) / ${MAX_SIZE_GB}GB - OK"
    exit 0
fi

echo "⚠️  Trash size exceeded: $(numfmt --to=iec-i --suffix=B $CURRENT_SIZE) / ${MAX_SIZE_GB}GB"
echo "🧹 Cleaning up old files..."

# Supprimer les fichiers les plus anciens jusqu'à atteindre la limite
# Stratégie FIFO : les plus vieux en premier
cd "$TRASH_DIR"

# Lister tous les fichiers par date de modification (plus vieux en premier)
find . -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | while read -r timestamp filepath; do
    # Recalculer la taille actuelle
    CURRENT_SIZE=$(du -sb "$TRASH_DIR" 2>/dev/null | awk '{print $1}')

    if [ "$CURRENT_SIZE" -le "$MAX_SIZE_BYTES" ]; then
        echo "✅ Trash cleaned: $(numfmt --to=iec-i --suffix=B $CURRENT_SIZE) / ${MAX_SIZE_GB}GB"
        break
    fi

    # Supprimer le fichier
    FILE_SIZE=$(stat -c%s "$filepath" 2>/dev/null || echo 0)
    rm -f "$filepath"
    echo "   Deleted: $filepath ($(numfmt --to=iec-i --suffix=B $FILE_SIZE))"
done

# Nettoyer les répertoires vides
find "$TRASH_DIR" -type d -empty -delete 2>/dev/null || true

echo "✅ Trash cleanup completed"
