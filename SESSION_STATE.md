# Anemone - Session State

**Current Version:** v0.9.23-beta
**Last Updated:** 2026-01-20

---

## Current Session

**Session 66** - Tests d'intégration Setup Wizard
- **Status:** Completed
- **Date:** 2026-01-20

### Completed (Session 66)

#### Bug Fix - Mutex Copy
- [x] Corrigé bug `go vet` : SetupState contenait un mutex copié
- [x] Créé `SetupStateView` - struct sans mutex pour usage externe
- [x] Mis à jour `GetState()` pour retourner `SetupStateView`
- [x] Mis à jour `SetupWizardData` dans handlers

#### Vérifications
- [x] `go vet ./...` - Aucune erreur
- [x] `go build` - Compilation OK
- [x] `go test ./...` - Tests passent
- [x] JSON translations valides

### Files Modified (Session 66)
- `internal/setup/setup.go` - Ajout SetupStateView, fix GetState()
- `internal/web/handlers_setup_wizard.go` - Utilise SetupStateView

---

## Previous Session

**Session 65** - Mode Restauration Serveur
- **Status:** Completed
- **Date:** 2026-01-20

### Completed (Session 65)

#### Backend Restauration (restore.go)
- [x] Créé `internal/setup/restore.go`
- [x] `ValidateBackup()` - Décrypte et valide un fichier backup
- [x] `ExecuteRestore()` - Restaure la config dans la DB
- [x] Restauration : system_config, users, shares, peers, sync_config
- [x] Support des chemins configurables (SharesDir, IncomingDir)

#### Handlers Web (handlers_setup_wizard.go)
- [x] `POST /setup/wizard/restore/validate` - Upload + validation backup
- [x] `POST /setup/wizard/restore/execute` - Exécution restauration
- [x] Stockage temporaire du backup pendant le wizard
- [x] Intégration avec le Manager (finalize, cleanup)

#### Frontend Wizard (setup_wizard.html)
- [x] Étape upload : fichier + passphrase
- [x] Étape confirmation : aperçu users/peers
- [x] Étape succès : résumé + lien login
- [x] JavaScript : validation, API calls, navigation
- [x] Drag & drop support pour upload

#### Traductions FR/EN
- [x] 25+ nouvelles clés `setup_wizard.restore.*`
- [x] Mise à jour note mode restore (plus "version future")

### Files Created (Session 65)
- `internal/setup/restore.go` - Logique de restauration

### Files Modified (Session 65)
- `internal/web/handlers_setup_wizard.go` - Endpoints restore
- `web/templates/setup_wizard.html` - UI flux restauration
- `internal/i18n/locales/fr.json` - Traductions restore
- `internal/i18n/locales/en.json` - Traductions restore

---

## Previous Session

**Session 64** - Nouveau install.sh simplifié
- **Status:** Completed
- **Date:** 2026-01-20

### Completed (Session 64)

#### Simplification install.sh
- [x] Supprimé `check_storage_setup()` - délégué au wizard web
- [x] Supprimé `validate_language()` - délégué au wizard web
- [x] Ajouté options CLI : `--data-dir`, `--user`, `--help`
- [x] Ajouté `parse_args()` et `show_help()`
- [x] Simplifié les fonctions (logging, structure)
- [x] Mise à jour message de fin → pointe vers wizard web
- [x] Réduit de 630 à 578 lignes

#### Mise à jour README.md
- [x] Section "Quick Installation" mise à jour
- [x] Section "Prerequisites" simplifiée (installer gère tout)
- [x] Section "One-Line Installation" simplifiée
- [x] Section "Standard Installation" mise à jour
- [x] Ajouté "Advanced Installation Options"
- [x] Section "Initial Setup" → décrit le wizard web

### Files Modified (Session 64)
- `install.sh` - Script simplifié avec options CLI
- `README.md` - Documentation mise à jour pour nouveau flux

---

## Previous Session

**Session 63** - Mode Setup - Frontend
- **Status:** Completed
- **Date:** 2026-01-20

### Completed (Session 63)

#### Setup Wizard Frontend
- [x] Modifié `handlers_setup_wizard.go` pour servir HTML au lieu de JSON
- [x] Créé template `setup_wizard.html` avec wizard multi-étapes complet
- [x] Étape 0 : Choix mode (Nouvelle installation / Restauration / Import pool)
- [x] Étape 1 : Sélection stockage (Défaut / ZFS existant / ZFS nouveau / Custom)
- [x] Étape 2 : Stockage sauvegardes entrantes (Même / Séparé)
- [x] Étape 3 : Création compte administrateur
- [x] Étape 4 : Résumé et finalisation
- [x] Étape 5 : Succès avec clé de chiffrement et mot de passe sync
- [x] Traductions FR/EN complètes (50+ clés)
- [x] JavaScript pour navigation wizard, validation, API calls
- [x] Compilation OK - pas d'erreurs

### Files Created (Session 63)
- `web/templates/setup_wizard.html` - Template wizard complet avec TailwindCSS

### Files Modified (Session 63)
- `internal/web/handlers_setup_wizard.go` - Ajout imports i18n/template, SetupWizardData, rendu HTML
- `internal/web/router.go` - Passage lang à NewSetupWizardServer()
- `internal/i18n/locales/fr.json` - 50+ traductions setup_wizard.*
- `internal/i18n/locales/en.json` - 50+ traductions setup_wizard.*

---

## Previous Session

**Session 62** - Mode Setup - Backend
- **Status:** Completed
- **Date:** 2026-01-20

### Completed (Session 62)

#### Setup Wizard Backend
- [x] Créé `internal/setup/setup.go` - Détection mode setup, gestion d'état
- [x] Créé `internal/setup/storage.go` - Configuration stockage (ZFS, défaut, custom)
- [x] Créé `internal/setup/finalize.go` - Finalisation (DB init, admin user, encryption keys)
- [x] Créé `internal/web/handlers_setup_wizard.go` - API endpoints wizard
- [x] Intégré mode setup dans main.go et router.go
- [x] Tests passent - compilation OK

### Files Created (Session 62)
- `internal/setup/setup.go` - Détection IsSetupNeeded(), Manager, SetupState
- `internal/setup/storage.go` - GetStorageOptions(), GetAvailableDisks(), SetupZFSStorage()
- `internal/setup/finalize.go` - FinalizeSetup(), création admin, sync password
- `internal/web/handlers_setup_wizard.go` - SetupWizardServer, API handlers

### Files Modified (Session 62)
- `cmd/anemone/main.go` - Ajout détection setup mode, runSetupMode()
- `internal/web/router.go` - Ajout NewSetupRouter(), NewRouterWithSetupCheck()

---

## Previous Session

**Session 61** - Refactoring permissions et utilisateur système
- **Status:** Completed
- **Date:** 2026-01-20
- **Commit:** `2174635`

### Completed (Session 61)

#### Permissions Refactoring
- [x] Ajout paramètre `owner` à `QuotaManager.CreateQuotaDir()` - ownership atomique
- [x] Suppression chown redondants dans handlers_auth.go
- [x] Simplification shares.Create() - suppression double chown sur .trash
- [x] Sécurisation sudoers dans install.sh - chemins restreints à DATA_DIR
- [x] Vérification sync P2P peut écrire dans incoming/

### Files Modified (Session 61)
- `internal/quota/enforcement.go` - CreateQuotaDir() avec owner atomique
- `internal/web/handlers_auth.go` - Suppression chown redondants
- `internal/shares/shares.go` - Simplification (un seul chown final)
- `install.sh` - Sudoers sécurisés (chemins restreints)
- `cmd/anemone-migrate/main.go` - Adaptation à la nouvelle signature

---

## Previous Session

**Session 60** - Refactoring chemins configurables
- **Status:** Completed
- **Date:** 2026-01-20
- **Commit:** `f589264`

### Completed (Session 60)
- [x] Fix router.go et handlers_admin_sync.go - cfg.IncomingDir
- [x] Refactor DeleteIncomingBackup() et DeleteUser() - paramètres configurables
- [x] Ajout ValidateDirs() - validation répertoires au démarrage

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

### Session 61 : Refactoring permissions et utilisateur système ✅ COMPLETED
**Objectif :** Sécuriser les permissions et créer un utilisateur dédié

- [x] Refactorer création subvolumes (ownership atomique via paramètre `owner`)
- [x] Sécuriser sudoers (chemins restreints à DATA_DIR, pas de wildcards dangereuses)
- [x] Fix double chown/chmod dans `shares.go` et `handlers_auth.go`
- [x] Vérifier que sync P2P peut toujours écrire dans incoming/ ✅
- [ ] *Option reportée: Créer utilisateur système `anemone` dédié (Session 64)*

**Fichiers modifiés :**
- `install.sh` - Sudoers sécurisés
- `internal/quota/enforcement.go` - CreateQuotaDir() avec owner
- `internal/shares/shares.go` - Simplification chown
- `internal/web/handlers_auth.go` - Suppression chown redondants
- `cmd/anemone-migrate/main.go` - Adaptation signature

---

### Session 62 : Mode Setup - Backend ✅ COMPLETED
**Objectif :** Créer la logique backend du wizard d'installation

- [x] Créer `internal/setup/` package
- [x] Détection "mode setup" au démarrage (pas de DB ou flag `--setup`)
- [x] API endpoints : `/setup/wizard/*` (state, storage, disks, admin, finalize)
- [x] Logique création pool ZFS avec permissions correctes
- [x] Création structure répertoires avec bons droits
- [x] Intégration dans main.go et router.go

**Fichiers créés :**
- `internal/setup/setup.go` - Détection et état du setup
- `internal/setup/storage.go` - Configuration stockage
- `internal/setup/finalize.go` - Finalisation installation
- `internal/web/handlers_setup_wizard.go` - API handlers

---

### Session 63 : Mode Setup - Frontend ✅ COMPLETED
**Objectif :** Créer l'interface utilisateur du wizard

- [x] Template `setup_wizard.html` (wizard multi-étapes)
- [x] **Étape 0 : Choix mode** (Nouvelle installation / Restauration / Import pool)
- [x] Étape 1 : Sélection stockage principal (ZFS / chemin existant / USB)
- [x] Étape 2 : Stockage sauvegardes entrantes (même disque / disque séparé)
- [x] Étape 3 : Création compte admin
- [x] Étape 4 : Résumé et finalisation
- [x] Étape 5 : Succès (clé chiffrement + mot de passe sync)
- [x] Traductions FR/EN complètes (50+ clés)
- [x] JavaScript pour navigation wizard

**Fichiers créés :**
- `web/templates/setup_wizard.html` - Template complet avec TailwindCSS

**Fichiers modifiés :**
- `internal/web/handlers_setup_wizard.go` - Rendu HTML
- `internal/web/router.go` - Passage langue
- `internal/i18n/locales/fr.json` - Traductions
- `internal/i18n/locales/en.json` - Traductions

---

### Session 65 : Mode Restauration Serveur ✅ COMPLETED
**Objectif :** Intégrer la restauration serveur dans le wizard d'installation

- [x] Créé `internal/setup/restore.go` - ValidateBackup() et ExecuteRestore()
- [x] Endpoints: `/setup/wizard/restore/validate` et `/setup/wizard/restore/execute`
- [x] UI wizard: upload, confirmation, succès
- [x] Traductions FR/EN (25+ clés)
- [x] Support drag & drop pour upload fichier

**Fichiers créés/modifiés :**
- `internal/setup/restore.go` - Logique de restauration
- `internal/web/handlers_setup_wizard.go` - Endpoints restauration
- `web/templates/setup_wizard.html` - UI flux restauration
- `internal/i18n/locales/fr.json` - Traductions
- `internal/i18n/locales/en.json` - Traductions

---

### Session 64 : Nouveau install.sh ✅ COMPLETED
**Objectif :** Simplifier le script d'installation

- [x] Réécrire `install.sh` : installe deps + binaire + service uniquement
- [x] Ne configure plus les chemins (délégué au wizard web)
- [x] Ajout options CLI : `--data-dir`, `--user`, `--help`
- [x] Mettre à jour `README.md` avec nouveau flux d'installation
- [ ] Tests d'installation sur VM propre (Fedora + Debian) → Session 67

**Fichiers modifiés :**
- `install.sh` - Script simplifié (630→578 lignes)
- `README.md` - Documentation mise à jour

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
| 66 | Tests d'intégration Setup Wizard | 2026-01-20 | Completed |
| 65 | Mode Restauration Serveur | 2026-01-20 | Completed |
| 64 | Nouveau install.sh simplifié | 2026-01-20 | Completed |
| 63 | Mode Setup - Frontend | 2026-01-20 | Completed |
| 62 | Mode Setup - Backend | 2026-01-20 | Completed |
| 61 | Refactoring permissions | 2026-01-20 | Completed |
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

**Prochaine session : Session 67** - Tests VM

Objectif : Tester l'installation complète sur des VM propres (Fedora et Debian).

**Sessions planifiées :**
- Session 67 : Tests VM (Fedora + Debian)
- Session 68 : Mode Import Pool ZFS (récupération après réinstallation)
- Session 69 : Documentation mise à jour

Commencer par `"continue"` ou `"session 67"`.
