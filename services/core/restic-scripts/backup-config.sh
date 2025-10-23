#!/bin/bash
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.

# Script de backup manuel de la configuration
# Appelle le script automatique en forçant l'exécution (pas de vérification incrémentielle)

set -e

export BACKUP_MODE="always"  # Forcer le backup même sans changements

# Appeler le script de backup automatique
exec /scripts/core/backup-config-auto.sh
