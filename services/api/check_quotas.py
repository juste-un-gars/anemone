#!/usr/bin/env python3
"""
Script de vérification et d'application des quotas disque
Peut être exécuté périodiquement via cron
"""

import sys
from quota_manager import QuotaManager


def main():
    """Vérifie et applique les quotas pour tous les pairs"""
    quota_manager = QuotaManager(config_path="/config/config.yaml")

    print("=== Vérification des quotas disque ===\n")

    # Vérifier tous les quotas
    quota_info = quota_manager.check_all_quotas()

    if not quota_info:
        print("Aucun pair configuré")
        return 0

    # Afficher le statut de chaque pair
    for quota in quota_info:
        peer_name = quota['peer_name']
        used = quota['used_formatted']
        limit = quota['quota_formatted']
        percentage = quota['percentage']
        within_quota = quota['within_quota']
        unlimited = quota['unlimited']

        status = "✓" if within_quota else "✗"

        if unlimited:
            print(f"{status} {peer_name}: {used} utilisés (illimité)")
        else:
            print(f"{status} {peer_name}: {used} / {limit} ({percentage}%)")

    print()

    # Appliquer les quotas pour les pairs qui dépassent
    enforced = []
    for quota in quota_info:
        if not quota['within_quota'] and not quota['unlimited']:
            peer_name = quota['peer_name']
            print(f"⚠️  {peer_name} dépasse son quota, application de la restriction...")

            success = quota_manager.enforce_quota(peer_name)
            if success:
                enforced.append(peer_name)

    if enforced:
        print(f"\n✓ {len(enforced)} pair(s) restreint(s): {', '.join(enforced)}")
        return 1  # Code de retour non-zero pour indiquer qu'une action a été prise
    else:
        print("✓ Tous les pairs respectent leur quota")
        return 0


if __name__ == "__main__":
    sys.exit(main())
