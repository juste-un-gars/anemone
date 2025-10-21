# Audit de Documentation - Anemone

**Date:** 2025-10-21
**Objectif:** Identifier et nettoyer la documentation redondante, obsolète, et réorganiser pour plus de clarté

## 📋 Fichiers de Documentation Actuels

### Documentation Principale (À Conserver)

| Fichier | Statut | Raison |
|---------|--------|--------|
| `README.md` | ⚠️ À METTRE À JOUR | Fichier principal - doit refléter Phase 3 |
| `CLAUDE.md` | ✅ OK | Instructions pour Claude Code - à jour |
| `DISASTER_RECOVERY.md` | ✅ OK | Documentation DR complète Phase 1-3 |
| `TROUBLESHOOTING.md` | ✅ OK | Guide de dépannage général |
| `ARCHITECTURE.md` | ⚠️ À VÉRIFIER | Architecture générale - vérifier si à jour |

### Guides Spécifiques (À Conserver)

| Fichier | Statut | Raison |
|---------|--------|--------|
| `INTERCONNEXION_GUIDE.md` | ✅ OK | Guide pour connecter les peers |
| `EXTERNAL_SHARES.md` | ✅ OK | Configuration des partages |
| `CONTRIBUTING.md` | ⚠️ À METTRE À JOUR | Mentionne restore.sh obsolète |

### Documentation Technique Détaillée (À Regrouper)

| Fichier | Statut | Action Recommandée |
|---------|--------|-------------------|
| `WIREGUARD_ARCHITECTURE.md` | ⚠️ REDONDANT | Fusionner dans ARCHITECTURE.md |
| `WIREGUARD_SETUP.md` | ⚠️ REDONDANT | Fusionner dans ARCHITECTURE.md |
| `VPN_TROUBLESHOOTING.md` | ⚠️ REDONDANT | Fusionner dans TROUBLESHOOTING.md |
| `NETWORK_AUTO_ALLOCATION.md` | ⚠️ SPÉCIFIQUE | Peut être fusionné dans ARCHITECTURE.md |

### Historique et Corrections (À Archiver)

| Fichier | Statut | Action Recommandée |
|---------|--------|-------------------|
| `CHANGELOG_WIREGUARD_FIX.md` | ⚠️ HISTORIQUE | Déplacer dans docs/archive/ |
| `CORRECTIONS_APPLIQUEES.md` | ⚠️ HISTORIQUE | Déplacer dans docs/archive/ |
| `WIREGUARD_KEY_FIX.md` | ⚠️ HISTORIQUE | Déplacer dans docs/archive/ |
| `ORDRE_INITIALISATION.md` | ⚠️ HISTORIQUE | Déplacer dans docs/archive/ |

### Guides de Migration (À Archiver ou Supprimer)

| Fichier | Statut | Action Recommandée |
|---------|--------|-------------------|
| `MIGRATION_GUIDE.md` | ❌ OBSOLÈTE | Migration vers système actuel - probablement obsolète |
| `PEERS_GUIDE.md` | ⚠️ REDONDANT | Probablement remplacé par INTERCONNEXION_GUIDE.md |

### Résumés d'Implémentation Phase (À Conserver Temporairement)

| Fichier | Statut | Action Recommandée |
|---------|--------|-------------------|
| `PHASE1_IMPLEMENTATION_SUMMARY.md` | ⚠️ TEMPORAIRE | Utile pour développement, archiver après v1.0 |
| `PHASE2_IMPLEMENTATION_SUMMARY.md` | ⚠️ TEMPORAIRE | Utile pour développement, archiver après v1.0 |
| `PHASE3_IMPLEMENTATION_SUMMARY.md` | ⚠️ TEMPORAIRE | Utile pour développement, archiver après v1.0 |

### Index

| Fichier | Statut | Action Recommandée |
|---------|--------|-------------------|
| `DOCUMENTATION_INDEX.md` | ⚠️ À METTRE À JOUR | Recréer avec nouvelle structure |

---

## 📁 Scripts

### Scripts Actifs (À Conserver)

| Script | Utilisé Par | Statut |
|--------|-------------|--------|
| `init.sh` | start.sh | ✅ ACTIF |
| `add-peer.sh` | Manuel | ✅ ACTIF |
| `discover-backups.py` | Phase 2/3 | ✅ ACTIF |
| `restore-config.py` | Phase 1-3 | ✅ ACTIF |
| `generate-wireguard-config.py` | init.sh | ✅ ACTIF |
| `diagnose-vpn.sh` | Manuel | ✅ ACTIF |
| `show-keys.sh` | Manuel | ✅ ACTIF |
| `extract-wireguard-pubkey.sh` | Manuel | ✅ ACTIF |
| `regenerate-wg-config.sh` | Manuel | ✅ ACTIF |
| `restart-vpn.sh` | Manuel | ✅ ACTIF |

### Scripts de Test (À Conserver)

| Script | Objectif | Statut |
|--------|----------|--------|
| `test-disaster-recovery.sh` | Test Phase 1 | ✅ ACTIF |
| `test-phase2.sh` | Test Phase 2 | ✅ ACTIF |
| `test-phase3.sh` | Test Phase 3 | ✅ ACTIF |

### Scripts Obsolètes (À Supprimer)

| Script | Raison | Action |
|--------|--------|--------|
| `init_script.sh` | ❌ Remplacé par init.sh | SUPPRIMER |
| `restore.sh` | ❌ Remplacé par restore-config.py + --auto-restore | SUPPRIMER |

---

## 📝 Plan de Nettoyage Recommandé

### Phase 1 : Archivage

Créer `docs/archive/` et y déplacer :
```
docs/archive/
├── CHANGELOG_WIREGUARD_FIX.md
├── CORRECTIONS_APPLIQUEES.md
├── WIREGUARD_KEY_FIX.md
├── ORDRE_INITIALISATION.md
├── PHASE1_IMPLEMENTATION_SUMMARY.md (après v1.0)
├── PHASE2_IMPLEMENTATION_SUMMARY.md (après v1.0)
└── PHASE3_IMPLEMENTATION_SUMMARY.md (après v1.0)
```

### Phase 2 : Consolidation

Créer `docs/technical/` et fusionner :
```
docs/technical/
├── ARCHITECTURE.md (fusionné avec WIREGUARD_*)
├── VPN_CONFIGURATION.md (contenu de WIREGUARD_SETUP + NETWORK_AUTO_ALLOCATION)
└── TROUBLESHOOTING.md (fusionné avec VPN_TROUBLESHOOTING)
```

### Phase 3 : Suppression

Supprimer les fichiers vraiment obsolètes :
```bash
rm scripts/init_script.sh
rm scripts/restore.sh
rm MIGRATION_GUIDE.md  # Vérifie contenu d'abord
rm PEERS_GUIDE.md  # Si redondant avec INTERCONNEXION_GUIDE
```

### Phase 4 : Mise à Jour

Mettre à jour les fichiers existants :
- `README.md` : Ajouter Phase 3, mise à jour des commandes
- `CONTRIBUTING.md` : Supprimer références à restore.sh
- `DOCUMENTATION_INDEX.md` : Recréer avec nouvelle structure

### Phase 5 : Nouvelle Structure

```
anemone/
├── README.md (Guide de démarrage)
├── CLAUDE.md (Instructions pour Claude Code)
├── ARCHITECTURE.md (Architecture complète consolidée)
├── DISASTER_RECOVERY.md (DR Phase 1-3 complet)
├── INTERCONNEXION_GUIDE.md (Connexion entre serveurs)
├── TROUBLESHOOTING.md (Dépannage consolidé)
├── EXTERNAL_SHARES.md (Configuration partages)
├── CONTRIBUTING.md (Guide de contribution)
│
├── docs/
│   ├── archive/ (Historique et corrections)
│   └── technical/ (Docs techniques détaillées si besoin)
│
└── scripts/
    ├── init.sh
    ├── add-peer.sh
    ├── restore-config.py
    ├── discover-backups.py
    ├── test-*.sh
    └── ... (autres scripts actifs)
```

---

## ✅ Actions Immédiates Recommandées

### Priorité 1 : Critique
1. ❌ Supprimer `scripts/init_script.sh` (clairement obsolète)
2. ❌ Supprimer `scripts/restore.sh` (remplacé par système Phase 1-3)
3. ⚠️ Mettre à jour `README.md` avec Phase 3
4. ⚠️ Mettre à jour `CONTRIBUTING.md` (supprimer ref à restore.sh)

### Priorité 2 : Important
5. 📁 Créer `docs/archive/` et y déplacer les corrections historiques
6. ⚠️ Vérifier et mettre à jour `ARCHITECTURE.md`
7. 🔄 Recréer `DOCUMENTATION_INDEX.md` propre

### Priorité 3 : Nettoyage
8. 🔄 Fusionner WIREGUARD_* dans ARCHITECTURE.md
9. 🔄 Fusionner VPN_TROUBLESHOOTING dans TROUBLESHOOTING.md
10. ❓ Vérifier si MIGRATION_GUIDE.md et PEERS_GUIDE.md sont encore pertinents

---

## 🎯 Résultat Attendu

**Avant nettoyage:**
- 22 fichiers .md
- 17 scripts
- Documentation dispersée et redondante

**Après nettoyage:**
- ~10 fichiers .md principaux
- 15 scripts actifs
- Documentation claire et organisée
- Historique archivé mais accessible
- README à jour avec toutes les fonctionnalités

---

## ⚠️ Précautions

Avant de supprimer quoi que ce soit :
1. ✅ Vérifier qu'aucun script ne référence le fichier
2. ✅ Lire le contenu pour s'assurer qu'il est bien obsolète
3. ✅ Archiver plutôt que supprimer si doute
4. ✅ Commit séparé pour chaque type de nettoyage
5. ✅ Tester que tout fonctionne après chaque étape
