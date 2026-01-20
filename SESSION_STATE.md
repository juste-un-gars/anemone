# Anemone - Session State

**Current Version:** v0.9.23-beta
**Last Updated:** 2026-01-20

---

## Current Session

**Session 60** - Refactoring chemins configurables
- **Status:** Completed
- **Date:** 2026-01-20
- **Commits:** `ae62041` (S59), `7f5607a` (S59 docs), `f589264` (S60)

### Completed (Session 60)

#### Configurable Paths Refactoring
- [x] Fix router.go - utilise `cfg.IncomingDir` au lieu de chemin hardcodé
- [x] Fix handlers_admin_sync.go - utilise `cfg.IncomingDir`
- [x] Refactor `DeleteIncomingBackup()` - accepte `incomingDir` pour sécurité configurable
- [x] Refactor `DeleteUser()` - accepte `sharesDir` comme paramètre
- [x] Ajout `ValidateDirs()` - validation répertoires au démarrage
- [x] Tests unitaires passent

### Files Modified (Session 60)
- `cmd/anemone/main.go` - Appel ValidateDirs() au démarrage
- `internal/config/config.go` - Ajout ValidateDirs() function
- `internal/incoming/incoming.go` - DeleteIncomingBackup() accepte incomingDir
- `internal/users/users.go` - DeleteUser() accepte sharesDir
- `internal/web/handlers_admin_sync.go` - Utilise cfg.IncomingDir
- `internal/web/handlers_admin_users.go` - Passe sharesDir à DeleteUser()
- `internal/web/router.go` - Utilise cfg.IncomingDir

---

## Previous Session

**Session 59** - Corrections urgentes (pré-refactoring)
- **Status:** Completed
- **Date:** 2026-01-20
- **Commit:** `ae62041`

### Completed (Session 59)
- [x] Fix 7 chemins hardcodés dans `handlers_sync_api.go`
- [x] Fix permissions ZFS après création pool/dataset
- [x] Unifier valeur par défaut DataDir (`/srv/anemone`)
- [x] Ajout `IncomingDir` et `ANEMONE_INCOMING_DIR`
- [x] Ajout `ANEMONE_SHARES_DIR`

---

## Previous Session

**Session 58.5** - Architecture Audit & Planning (Setup Wizard)
- **Status:** Completed (Planning)
- **Date:** 2026-01-20

### Completed (Session 58.5)

#### Architecture Audit for Setup Wizard Feature
- [x] Audit complet des chemins hardcodés dans le code
- [x] Audit du système de configuration et dépendances
- [x] Audit des permissions, sudo, et utilisateur système
- [x] Planification des sessions 59-64

### Audit Findings Summary

#### Problèmes critiques identifiés

| Priorité | Problème | Fichier(s) | Status |
|----------|----------|------------|--------|
| ~~CRITIQUE~~ | ~~7 chemins hardcodés `/srv/anemone/backups/incoming`~~ | ~~`handlers_sync_api.go`~~ | ✅ Fixed (S59) |
| CRITIQUE | Subvolumes Btrfs créés par root, ownership fixé après | `enforcement.go`, `handlers_auth.go` | Pending (S61) |
| HIGH | Sudoers avec wildcards dangereuses (`chown -R *`, `rm *`) | `install.sh:401-407` | Pending (S61) |
| ~~HIGH~~ | ~~ZFS : pas de fix permissions après création pool/dataset~~ | ~~`zfs_pool.go`, `zfs_dataset.go`~~ | ✅ Fixed (S59) |
| MEDIUM | Double chown/chmod sur les répertoires | `shares.go`, `handlers_auth.go` | Pending (S61) |
| MEDIUM | Pas d'utilisateur système dédié "anemone" | `install.sh` | Pending (S61) |
| ~~MEDIUM~~ | ~~Valeurs par défaut incohérentes (`/app/data` vs `/srv/anemone`)~~ | ~~`config.go`~~ | ✅ Fixed (S59) |

#### Points positifs
- Architecture propre, pas de dépendances circulaires
- Tout dérive de `ANEMONE_DATA_DIR` ✅ (bugs hardcodés corrigés en S59)
- Initialisation séquentielle bien ordonnée
- SharesDir et IncomingDir maintenant configurables séparément

---

## Planned Sessions (59-64) - Setup Wizard Refactoring

### IMPORTANT - Points d'attention

1. **Sync P2P et permissions** : Le système de sauvegarde/restauration entre pairs a ses propres droits. Les fichiers dans `incoming/` sont écrits par le processus de sync distant. Ne pas casser cette logique.

2. **Rollback possible** : Si problèmes majeurs, pouvoir revenir en arrière. Faire des commits atomiques et testables.

3. **Compatibilité** : Pas de migration nécessaire (beta), mais documenter les breaking changes.

### UX du Setup Wizard - Question principale

**"Où souhaitez-vous installer Anemone ?"**

| Option | Description | Action |
|--------|-------------|--------|
| **Répertoire par défaut** | `/srv/anemone` | Crée le répertoire si nécessaire |
| **Autre disque à monter** | Spécifier le disque et le point de montage | Monte le disque, crée la structure |
| **Pool ZFS existant** | Sélectionner parmi les pools détectés | Utilise le pool comme stockage |
| **Nouveau pool ZFS** | Sélectionner disques + point de montage | Crée le pool puis la structure |

Ensuite, question optionnelle pour le stockage séparé des backups entrants (si l'utilisateur a choisi ZFS pour les données principales).

### Scénarios de stockage supportés

| Scénario | Shares (données utilisateurs) | Incoming (backups pairs) | Cas d'usage |
|----------|-------------------------------|--------------------------|-------------|
| **Simple** | Répertoire unique | Même répertoire | Dev, test, petit déploiement |
| **ZFS unifié** | Pool ZFS | Même pool ZFS | Redondance complète |
| **Hybride** | Pool ZFS (mirror/raidz) | Disque séparé simple | Économie d'espace ZFS |
| **Avancé** | Chemin personnalisé | Chemin personnalisé | Configurations spéciales |

---

### Session 59 : Corrections urgentes (pré-refactoring) ✅ COMPLETED
**Objectif :** Corriger les bugs critiques avant le refactoring majeur

- [x] Fix 7 chemins hardcodés dans `handlers_sync_api.go`
- [x] Fix permissions ZFS après création pool/dataset (chown mountpoint)
- [x] Unifier valeur par défaut DataDir (`/srv/anemone` partout)
- [x] Tests passent - non-régression vérifiée

**Fichiers modifiés :**
- `internal/web/handlers_sync_api.go`
- `internal/storage/zfs_pool.go`
- `internal/storage/zfs_dataset.go`
- `internal/config/config.go`
- `internal/incoming/incoming.go`

---

### Session 60 : Refactoring chemins configurables ✅ COMPLETED
**Objectif :** Propager l'utilisation de SharesDir et IncomingDir dans tous les packages

- [x] Ajouter `SharesDir`, `IncomingDir` dans `config.Config` ✅ (S59)
- [x] Variables d'environnement : `ANEMONE_SHARES_DIR`, `ANEMONE_INCOMING_DIR` ✅ (S59)
- [x] Valeurs par défaut : `{DataDir}/shares`, `{DataDir}/backups/incoming` ✅ (S59)
- [x] Validation mountpoint : existence, permissions d'écriture au démarrage
- [x] Mettre à jour `incoming.go` - sécurité configurable avec incomingDir
- [x] Mettre à jour `users.go` - utilise sharesDir paramètre
- [x] Vérifier `handlers_admin_sync.go` utilise bien les chemins config
- [x] Vérifier impact sur sync P2P (syncauth/sync/ propres - pas de chemins hardcodés)

**Fichiers modifiés :**
- `cmd/anemone/main.go` - ValidateDirs() au démarrage
- `internal/config/config.go` - ValidateDirs() function
- `internal/incoming/incoming.go` - DeleteIncomingBackup() avec incomingDir
- `internal/users/users.go` - DeleteUser() avec sharesDir
- `internal/web/handlers_admin_sync.go` - cfg.IncomingDir
- `internal/web/handlers_admin_users.go` - sharesDir à DeleteUser()
- `internal/web/router.go` - cfg.IncomingDir

---

### Session 61 : Refactoring permissions et utilisateur système
**Objectif :** Sécuriser les permissions et créer un utilisateur dédié

- [ ] Option : Créer utilisateur système `anemone` dédié dans `install.sh`
- [ ] Refactorer création subvolumes (ownership atomique, pas de double chown)
- [ ] Sécuriser sudoers (arguments explicites, pas de wildcards)
- [ ] Fix double chown/chmod dans `shares.go`
- [ ] Vérifier que sync P2P peut toujours écrire dans incoming/

**Fichiers concernés :**
- `install.sh`
- `internal/quota/enforcement.go`
- `internal/shares/shares.go`
- `internal/web/handlers_auth.go`

---

### Session 62 : Mode Setup - Backend
**Objectif :** Créer la logique backend du wizard d'installation

- [ ] Créer `internal/setup/` package
- [ ] Détection "mode setup" au démarrage (pas de DB ou flag `--setup`)
- [ ] API endpoints : `/setup/storage`, `/setup/admin`, `/setup/finalize`
- [ ] Logique création pool ZFS avec permissions correctes
- [ ] Logique montage disque USB
- [ ] Création structure répertoires avec bons droits

**Fichiers à créer :**
- `internal/setup/setup.go` - Détection et état du setup
- `internal/setup/storage.go` - Configuration stockage
- `internal/setup/finalize.go` - Finalisation installation
- `internal/web/handlers_setup.go` - API handlers

---

### Session 63 : Mode Setup - Frontend
**Objectif :** Créer l'interface utilisateur du wizard

- [ ] Template `setup_wizard.html` (wizard multi-étapes)
- [ ] Étape 1 : Sélection stockage principal (ZFS / chemin existant / USB)
- [ ] Étape 2 : Stockage sauvegardes entrantes (même disque / disque séparé)
- [ ] Étape 3 : Configuration avancée (chemins personnalisés, optionnel)
- [ ] Étape 4 : Création compte admin
- [ ] Étape 5 : Résumé et finalisation
- [ ] Traductions FR/EN complètes
- [ ] JavaScript pour navigation wizard

**Fichiers à créer :**
- `web/templates/setup_wizard.html`
- `web/static/js/setup.js` (optionnel)

---

### Session 64 : Nouveau install.sh
**Objectif :** Simplifier le script d'installation

- [ ] Réécrire `install.sh` : installe deps + binaire + service uniquement
- [ ] Ne configure plus les chemins (délégué au wizard web)
- [ ] Créer utilisateur système `anemone` si option choisie
- [ ] Mettre à jour `README.md` avec nouveau flux d'installation
- [ ] Tests d'installation sur VM propre (Fedora + Debian)

**Fichiers concernés :**
- `install.sh`
- `README.md`
- `docs/INSTALL.md` (nouveau, optionnel)

---

## Previous Session

**Session 58** - Storage Management Bug Fixes & Mountpoint
- **Status:** Completed
- **Date:** 2026-01-20
- **Commits:** `5ac947f`, `7fb3afc`, `5456134`, `8964835`

### Completed (Session 58)

#### Bug Fix: ZFS Pool Creation Modal Issues
- [x] Fixed password modal appearing behind form modal (same z-index)
- [x] Close form modals before showing password verification modal
- [x] Fixed pendingAction being nullified before callback execution
- [x] Fixed JavaScript syntax error in pool name template generation

#### Feature: Mountpoint Option for Pool Creation
- [x] Added mountpoint field in pool creation form
- [x] Added mountpoint validation to prevent system path usage (/etc, /var, etc.)
- [x] Prevent deletion of root datasets (must use "Destroy Pool" instead)
- [x] Added FR/EN translations for mountpoint

### Files Modified (Session 58)
- `web/templates/admin_storage.html` - Fixed modals, added mountpoint field
- `internal/storage/zfs_pool.go` - Added ValidateMountpoint function
- `internal/storage/zfs_dataset.go` - Prevent deletion of root datasets
- `internal/i18n/locales/fr.json` - Added mountpoint translations
- `internal/i18n/locales/en.json` - Added mountpoint translations

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 60 | Refactoring chemins configurables | 2026-01-20 | Completed |
| 59 | Corrections urgentes (pré-refactoring) | 2026-01-20 | Completed |
| 58.5 | Architecture Audit & Planning | 2026-01-20 | Completed |
| 58 | Storage Bug Fixes & Mountpoint | 2026-01-20 | Completed |
| 57 | Storage Management (Phase 2-3) | 2026-01-20 | Completed |
| 56 | Storage Management (Phase 1) | 2026-01-20 | Completed |
| 55 | Bug Fixes (Speed, Empty Dirs, Datetime) | 2026-01-19 | Completed |
| 54 | Bug Fixes & Release Management | 2026-01-18 | Completed |
| 53 | Performance & Real-time Manifests | 2025-01-18 | Completed |
| 52 | Security Audit Phases 1-5 | 2025-01-18 | Completed |
| 51 | User Share Manifests | 2025-01-18 | Completed |
| 37-39 | Security Audit & Fixes | 2024-12 | Completed |
| 31-34 | Update System | 2024-11 | Completed |
| 27-30 | Restore Interface | 2024-11 | Completed |
| 26 | Internationalization FR/EN | 2024-11-20 | Completed |
| 20-24 | P2P Sync & Scheduler | 2024-11 | Completed |
| 17-19 | Trash & Quotas | 2024-11 | Completed |
| 12-16 | SMB Automation | 2024-11 | Completed |
| 8-11 | P2P Foundation | 2024-11 | Completed |
| 1-7 | Initial Setup & Auth | 2024-10 | Completed |

---

## Session Archives

All detailed session files are in `.claude/sessions/`:

- `SESSION_052_security_audit.md` - Current audit session
- `SESSION_051_user_manifests.md` - User manifests
- `SESSION_STATE_ARCHIVE.md` - Sessions 1-7
- `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md` - P2P Foundation
- `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md` - SMB Automation
- `SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md` - Trash & Quotas
- `SESSION_STATE_ARCHIVE_SESSIONS_20_24.md` - P2P Sync & Scheduler
- `SESSIONS_ARCHIVE.md` - Session 26 (i18n)
- `SESSION_STATE_ARCHIVE_SESSIONS_27_30.md` - Restore Interface
- `SESSION_STATE_ARCHIVE_31_32_33_34.md` - Update System
- `SESSION_STATE_ARCHIVE_SESSIONS_37_39.md` - Security Audit

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[.claude/REFERENCE.md](.claude/REFERENCE.md)** - Quick reference
- **[README.md](README.md)** - Installation guide
- **[CHANGELOG.md](CHANGELOG.md)** - Version history
- **[docs/SECURITY.md](docs/SECURITY.md)** - Security documentation
- **[docs/API.md](docs/API.md)** - API documentation

---

## Next Steps

**Prochaine session : Session 61** - Refactoring permissions et utilisateur système

Objectif : Sécuriser les permissions et créer un utilisateur système dédié `anemone`.

Commencer par `"continue"` ou `"session 61"`.
