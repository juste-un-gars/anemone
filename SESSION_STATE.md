# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.15.3-beta
**Last Updated:** 2026-02-14

---

## Current Session

**Session 19: Web File Browser** - Complete ✅

**Détails :** `.claude/sessions/SESSION_019_file_browser.md`

---

## Session 19: Web File Browser

**Date:** 2026-02-14
**Objective:** Add a web file browser for users to browse, upload, download, create folders, rename and delete files
**Status:** Complete ✅

### Changes
| # | Type | Description |
|---|------|-------------|
| 1 | Feature | New file browser at `/files` — browse shares, navigate folders with breadcrumb |
| 2 | Feature | Upload files (multipart, progress bar, 2GB limit) |
| 3 | Feature | Download files via `/api/files/download` |
| 4 | Feature | Create folders via `/api/files/mkdir` |
| 5 | Feature | Rename files/folders via `/api/files/rename` |
| 6 | Feature | Delete files → moved to `.trash/{username}/` (Samba recycle pattern) |
| 7 | UI | Sidebar link "Fichiers" between Dashboard and Trash |
| 8 | i18n | 30 translation keys FR + EN |

### Files Created
- `internal/web/handlers_files.go` (640 lines) — All handlers + helpers
- `web/templates/v2/v2_files.html` (328 lines) — File browser template

### Files Modified
- `internal/web/router.go` — 6 routes
- `web/templates/v2/v2_base_user.html` — Sidebar link
- `internal/i18n/locales/en.json` — 30 keys
- `internal/i18n/locales/fr.json` — 30 keys

### Security
- Path traversal protection via `resolveSharePath()` + `isPathTraversal()` + `filepath.EvalSymlinks()`
- Share ownership validation (user can only access own shares)
- Filename validation (`isValidFileName()` rejects `..`, `/`, `\`, null, dotfiles)
- Upload size limit via `http.MaxBytesReader`
- `sudo` for filesystem ops (same pattern as trash.go/shares.go)

---

## Previous Session

**Session 18: Dashboard Last Backup Fix + Recent Backups Tab** - Complete ✅

**Détails :** `.claude/sessions/SESSION_018_recent_backups.md`

---

## Previous Session

**Session 17: Rclone Crypt Fix + !BADKEY Logs** - Complete ✅

**Détails :** `.claude/sessions/SESSION_017_rclone_crypt_badkey.md`

---

## Previous Session

**Session 16: SSH Key Bugfix** - Complete ✅

**Détails :** `.claude/sessions/SESSION_016_ssh_key_bugfix.md`

---

## Session 16: SSH Key Bugfix

**Date:** 2026-02-11
**Objective:** Fix SSH key generation error + cleanup !BADKEY log
**Status:** Complete ✅

### Bugs corrigés (3)
| # | Bug | Fix |
|---|-----|-----|
| 1 | Génération clé SSH affiche "Error" malgré succès | JS vérifiait `data.success` mais l'API renvoie `data.exists` |
| 2 | `!BADKEY` dans les logs génération SSH key | printf-style → slog key-value |
| 3 | Dashboard utilisateur crashe (QuotaInfo int vs float64) | `eq .QuotaBackupGB 0.0` → `eq .QuotaBackupGB 0` + fix log !BADKEY |

### Release
**v0.15.3-beta** — SSH key + user dashboard bugfixes

---

## Previous Session

**Session 15: Rclone & UI Bugfixes** - Complete ✅

**Détails :** `.claude/sessions/SESSION_015_rclone_bugfixes.md`

---

## Previous Session

**Session 14: v2 UI Bugfixes** - Complete ✅

**Détails :** `.claude/sessions/SESSION_014_v2_bugfixes.md`

---

## Previous Session

**Session 13: Cloud Backup Multi-Provider + Chiffrement** - Complete ✅

**Détails :** `.claude/sessions/SESSION_013_cloud_multi_provider.md`

---

## Previous Session

**Session 12: Module F Cleanup** - Complete ✅

**Détails :** `.claude/sessions/SESSION_011_v2_ui_redesign.md`

---

## Previous Session

**Session 11: V2 UI Redesign** - Complete ✅ (Modules A-D)

---

## Session 11: V2 UI Redesign

**Date:** 2026-02-08
**Objective:** Refonte complète UI : dark theme, sidebar gauche, sauvegardes consolidées (inspiré newui.jpg)
**Status:** Complete ✅ — Modules A-D + F terminés

**Détails :** `.claude/sessions/SESSION_011_v2_ui_redesign.md`

---

## Previous Session

**Session 10: USB Backup Mount Fix** - Complete ✅

---

## Session 10: USB Backup Mount Fix

**Date:** 2026-02-08
**Objective:** Fix false success status when USB drive is physically removed without unmounting
**Status:** Complete ✅

### Bug Description

When a USB drive is physically removed without being properly unmounted, the backup still shows "Success" status because `IsMounted()` only checked if the directory `/mnt/xxx` existed (which it does - it's an empty folder on the root filesystem). This also caused backup data to be written to the system disk instead of the USB drive.

### Fix Applied

| File | Change |
|------|--------|
| `internal/usbbackup/usbbackup.go` | `IsMounted()` now reads `/proc/mounts` to verify actual mount point |
| `internal/updater/updater.go` | Version bump to 0.13.6-beta |
| `CHANGELOG.md` | Added v0.13.6-beta entry |

### Release

**v0.13.6-beta** released: https://github.com/juste-un-gars/anemone/releases/tag/v0.13.6-beta

---

## Previous Session

**Session 9: Documentation Update & Cleanup** - Complete ✅

---

## Session 9: Documentation Update & Cleanup

**Date:** 2026-01-31
**Objective:** Update all documentation, add architecture diagram, cleanup releases
**Status:** Complete ✅

### Documentation Updates

| File | Changes |
|------|---------|
| **CHANGELOG.md** | Added v0.13.4-beta (Rclone) + v0.13.5-beta (SSH Key) + comparison links |
| **README.md** | Added WireGuard, Cloud backup, Logging in Features + architecture diagram |
| **docs/README.md** | Added link to rclone-backup.md |
| **docs/user-guide.md** | Added sections: Cloud Backup, WireGuard VPN, System Logs |
| **.claude/REFERENCE.md** | Updated version, DB tables, packages, env vars, date |
| **docs/architecture.png** | New architecture diagram |
| **TROUBLESHOOTING.md** | Removed personal info from examples |

### Cleanup

- Removed personal info ("franck") from documentation examples
- Deleted all old GitHub releases (kept only v0.13.5-beta)
- Tags git conserved for traceability

### Summary

- All documentation now reflects v0.13.5-beta features
- Architecture diagram added to README.md
- User guide covers all major features (USB Backup, Cloud Backup, WireGuard, Logs)
- GitHub releases page cleaned up (single release)

---

## Previous Session

**Session 8: Rclone SSH Key Generation** - Complete ✅

---

## Session 8: Rclone SSH Key Generation

**Date:** 2026-01-31
**Objectif:** Ajouter génération de clé SSH depuis l'interface web + documentation serveur distant
**Status:** Complete ✅

### Contexte

Améliorer l'UX du module rclone pour éviter les lignes de commande :
- Générer la clé SSH depuis l'interface web
- Afficher la clé publique à copier
- Utiliser des chemins relatifs pour la portabilité
- Documenter la configuration du serveur distant

### Modules implémentés

| # | Module | Objectif | Status |
|---|--------|----------|--------|
| 1 | **SSH Key Generation** | Fonctions GenerateSSHKey, GetSSHKeyInfo, ResolveKeyPath | ✅ Done |
| 2 | **Handlers + Routes** | Endpoints /admin/rclone/key-info et /generate-key | ✅ Done |
| 3 | **UI Update** | Section clé SSH dans admin_rclone.html | ✅ Done |
| 4 | **Traductions** | Clés i18n FR + EN pour section clé SSH | ✅ Done |
| 5 | **Documentation** | docs/rclone-backup.md | ✅ Done |

### Architecture

```
internal/rclone/
├── rclone.go       # Struct RcloneBackup, CRUD DB
├── sync.go         # Fonctions sync (modifié: ResolveKeyPath)
├── sshkey.go       # NEW: GenerateSSHKey, GetSSHKeyInfo, ResolveKeyPath
└── scheduler.go    # Scheduler pour sync automatique
```

### Files Created
- `internal/rclone/sshkey.go` - Génération et gestion clé SSH (~100 lignes)
- `docs/rclone-backup.md` - Documentation complète (~200 lignes)

### Files Modified
- `internal/rclone/sync.go` - buildRemoteString prend dataDir, utilise ResolveKeyPath
- `internal/web/handlers_admin_rclone.go` - handleAdminRcloneKeyInfo, handleAdminRcloneGenerateKey
- `internal/web/router.go` - Routes /admin/rclone/key-info et /generate-key
- `web/templates/admin_rclone.html` - Section clé SSH + JavaScript
- `internal/i18n/locales/fr.json` - Traductions FR (~15 clés)
- `internal/i18n/locales/en.json` - Traductions EN (~15 clés)
- `internal/updater/updater.go` - Version bump 0.13.5-beta

### Fonctionnalités

- **Génération clé SSH depuis UI** : Bouton "Générer une clé SSH"
- **Affichage clé publique** : Zone de texte copiable
- **Régénération avec confirmation** : Avertit que l'ancienne clé sera invalidée
- **Chemins relatifs** : `certs/rclone_key` stocké en DB, résolu au runtime
- **Pré-remplissage** : Le champ "Key Path" utilise le chemin relatif par défaut
- **Documentation serveur distant** : Guide complet pour configurer le serveur SFTP

### Release

**v0.13.5-beta** : SSH key generation + relative paths + documentation

### Tests en cours

- [ ] Générer clé SSH depuis l'interface web
- [ ] Copier clé publique sur serveur distant (FR2)
- [ ] Configurer destination SFTP avec chemin relatif
- [ ] Tester connexion
- [ ] Lancer synchronisation manuelle
- [ ] Vérifier fichiers sur serveur distant

---

## Previous Session

**Session 7: Rclone Cloud Backup** - Complete ✅

---

## Session 7: Rclone Cloud Backup

**Date:** 2026-01-31
**Objectif:** Ajouter module rclone pour backup SFTP des répertoires backup/ utilisateurs
**Status:** Complete ✅

### Contexte

Module pour sauvegarder les répertoires `backup/` de tous les utilisateurs vers un serveur SFTP externe via rclone. Mode push uniquement (Anemone → SFTP).

### Modules implémentés

| # | Module | Objectif | Status |
|---|--------|----------|--------|
| 1 | **Infrastructure DB + Struct** | Table `rclone_backups`, migration, struct Go, CRUD | ✅ Done |
| 2 | **Sync avec rclone** | Fonctions sync via rclone CLI | ✅ Done |
| 3 | **Scheduler** | Planification automatique des backups | ✅ Done |
| 4 | **UI + Handlers** | Interface admin, routes, templates | ✅ Done |
| 5 | **Traductions** | i18n FR + EN | ✅ Done |

### Architecture

```
internal/rclone/
├── rclone.go       # Struct RcloneBackup, CRUD DB
├── sync.go         # Fonctions de synchronisation (appel rclone)
└── scheduler.go    # Scheduler pour sync automatique

web/templates/
├── admin_rclone.html       # Interface admin (liste + formulaire)
└── admin_rclone_edit.html  # Formulaire d'édition

internal/web/
└── handlers_admin_rclone.go  # Handlers HTTP
```

### Schéma DB

```sql
CREATE TABLE rclone_backups (
    id INTEGER PRIMARY KEY,
    name TEXT,
    sftp_host TEXT,
    sftp_port INTEGER DEFAULT 22,
    sftp_user TEXT,
    sftp_key_path TEXT,
    sftp_password TEXT,
    remote_path TEXT,
    enabled INTEGER DEFAULT 1,
    sync_enabled INTEGER DEFAULT 0,
    sync_frequency TEXT DEFAULT 'daily',
    sync_time TEXT DEFAULT '02:00',
    sync_day_of_week INTEGER,
    sync_day_of_month INTEGER,
    sync_interval_minutes INTEGER DEFAULT 60,
    last_sync DATETIME,
    last_status TEXT DEFAULT 'unknown',
    last_error TEXT,
    files_synced INTEGER DEFAULT 0,
    bytes_synced INTEGER DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
```

### Files Created
- `internal/rclone/rclone.go` - Struct + CRUD (~400 lignes)
- `internal/rclone/sync.go` - Fonctions sync (~250 lignes)
- `internal/rclone/scheduler.go` - Scheduler (~60 lignes)
- `internal/web/handlers_admin_rclone.go` - Handlers HTTP (~350 lignes)
- `web/templates/admin_rclone.html` - Template liste + ajout
- `web/templates/admin_rclone_edit.html` - Template édition

### Files Modified
- `internal/database/migrations.go` - Ajout `migrateRcloneTable()`
- `internal/web/router.go` - Routes `/admin/rclone/*`
- `web/templates/dashboard_admin.html` - Tuile "Rclone Cloud Backup"
- `internal/i18n/locales/fr.json` - Traductions FR (~40 clés)
- `internal/i18n/locales/en.json` - Traductions EN (~40 clés)
- `cmd/anemone/main.go` - Démarrage scheduler rclone
- `internal/updater/updater.go` - Version bump 0.13.4-beta

### Fonctionnalités

- **Destinations SFTP multiples** : Configurer plusieurs serveurs SFTP
- **Auth SSH key ou password** : Support clé SSH ou mot de passe
- **Test de connexion** : Vérifier la connexion avant sync
- **Sync manuelle** : Bouton "Sync maintenant"
- **Planification** : Interval/Daily/Weekly/Monthly (comme USB Backup)
- **Statistiques** : Fichiers/octets synchronisés, dernière sync

### Commandes rclone utilisées

```bash
# Test de connexion
rclone lsd :sftp,host=server,user=user,key_file=/path/to/key: /path

# Sync d'un répertoire backup utilisateur
rclone sync /srv/anemone/shares/alice/backup/ \
    :sftp,host=server,user=user,key_file=/path/to/key:/backups/anemone/alice/ \
    --progress --stats-one-line
```

### Release

**v0.13.4-beta** released: https://github.com/juste-un-gars/anemone/releases/tag/v0.13.4-beta

### À faire (non implémenté)

- **install.sh** : Ajouter installation optionnelle de rclone via `curl https://rclone.org/install.sh | sudo bash`
- **Vérification version rclone** : Notifier l'admin si une nouvelle version est disponible (comme pour Anemone)

---

## Release v0.13.5-beta (2026-01-31) ✅

### New Features - SSH Key Generation for Rclone
- **Generate SSH key from UI**: One-click SSH key generation in Cloud Backup page
- **Display public key**: Copyable public key to add to remote servers
- **Relative paths**: Key paths stored as relative (e.g., `certs/rclone_key`), resolved at runtime
- **Portable configuration**: Moving data directory doesn't break key paths
- **Pre-filled key path**: Form auto-fills with generated key path

### Documentation
- **New: docs/rclone-backup.md**: Complete guide for configuring rclone SFTP backup
  - Anemone configuration (via UI)
  - Remote server setup (SSH, user creation, authorized_keys)
  - Troubleshooting section

### Technical Changes
- New `internal/rclone/sshkey.go` with GenerateSSHKey, GetSSHKeyInfo, ResolveKeyPath
- Modified `buildRemoteString()` to resolve relative key paths
- New routes: `/admin/rclone/key-info`, `/admin/rclone/generate-key`
- Updated admin_rclone.html with SSH key section
- i18n translations (FR + EN) for SSH key UI elements

---

## Release v0.13.4-beta (2026-01-31) ✅

### New Features - Rclone Cloud Backup
- **SFTP backup destinations**: Configure multiple SFTP servers for cloud backup
- **SSH key or password auth**: Support both authentication methods
- **Connection testing**: Verify SFTP connection before syncing
- **Manual sync**: "Sync now" button for immediate backup
- **Automatic scheduling**: Interval/Daily/Weekly/Monthly (like USB Backup)
- **Sync statistics**: Files/bytes synced, last sync time and status

### Technical Changes
- New `internal/rclone/` package (rclone.go, sync.go, scheduler.go)
- New `rclone_backups` table with SFTP config, scheduling, and status fields
- New admin UI at `/admin/rclone` with add/edit/delete/sync/test actions
- Scheduler runs every minute, checks enabled backups
- i18n translations (FR + EN) for all rclone UI elements

---

## Previous Session

**Session 6: WireGuard Integration** - Complete ✅

---

## Session 6: WireGuard Integration

**Date:** 2026-01-30 → 2026-01-31
**Objectif:** Ajouter le support WireGuard pour VPN entre peers Anemone
**Status:** Complete ✅

### Contexte WireGuard

WireGuard est un VPN moderne, simple et performant:
- Utilise des paires de clés publiques/privées (comme SSH)
- Configuration minimaliste (vs OpenVPN)
- Intégré au kernel Linux depuis 5.6
- Idéal pour connecter des peers Anemone à travers Internet

### Modules planifiés

| # | Module | Objectif | Status |
|---|--------|----------|--------|
| 1 | **Infrastructure DB** | Table `wireguard_config`, migration, struct Go, CRUD | ✅ Done |
| 2 | **install.sh** | Installation optionnelle wireguard-tools | ✅ Done |
| 3 | **UI Dashboard + Routes** | Tuile admin, handlers, template | ✅ Done |
| 4 | **Import .conf** | Parser fichier et stocker en DB | ✅ Done |
| 5 | **Édition manuelle** | Formulaire pour modifier les champs | ✅ Done (2026-01-31) |
| 6 | **Activation** | `wg-quick up/down`, toggle ON/OFF | ✅ Done |
| 7 | **Auto-start** | Lancer au démarrage d'Anemone si configuré | ✅ Done |
| 8 | **Statut** | Afficher état connexion | ✅ Done |
| 9 | **Backup/Restore** | Intégration avec sauvegarde/restauration | ✅ Done |
| 10 | **PresharedKey** | Support clé pré-partagée dans parser/conf | ✅ Done (2026-01-31) |
| 11 | **Client PublicKey** | Dérivation et affichage clé publique client | ✅ Done (2026-01-31) |

### Architecture prévue

```
internal/wireguard/
├── wireguard.go        # Struct WireGuardConfig, CRUD DB
└── config.go           # Génération fichier .conf pour wg-quick

web/templates/
└── admin_wireguard.html  # Interface admin

internal/web/
└── handlers_admin_wireguard.go  # Handlers HTTP
```

### Schéma DB (client only)

```sql
CREATE TABLE wireguard_config (
    id INTEGER PRIMARY KEY,
    name TEXT DEFAULT 'wg0',
    -- Interface
    private_key TEXT,
    address TEXT,
    dns TEXT,
    -- Peer (serveur)
    peer_public_key TEXT,
    peer_preshared_key TEXT,  -- Added 2026-01-31
    peer_endpoint TEXT,
    allowed_ips TEXT,
    persistent_keepalive INTEGER DEFAULT 25,
    -- Options
    enabled INTEGER DEFAULT 0,
    auto_start INTEGER DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
```

**Note:** La clé publique du client (`PublicKey`) est dérivée de `PrivateKey` via `wg pubkey`, pas stockée en DB.

### Release

**v0.13.3-beta** released: https://github.com/juste-un-gars/anemone/releases/tag/v0.13.3-beta

### Current Module

**Working on:** Session Complete
**Progress:** ✅ All modules done

### Bugfixes (2026-01-31)

| Issue | Fix |
|-------|-----|
| Template panic on startup | Replaced `\"` with backticks in Go template |
| No edit button | Added Edit modal + handler `/admin/wireguard/edit` |
| PresharedKey missing | Added parsing + generation in .conf files |
| VPN not connecting | PresharedKey was not included in generated conf |
| Client PublicKey not shown | Added `DerivePublicKey()` using `wg pubkey` |

### Files Modified
- `internal/database/migrations.go` - Ajout `migrateWireGuardTable()` + colonne `peer_preshared_key`
- `internal/wireguard/wireguard.go` - Struct Config avec PublicKey/PeerPresharedKey, `DerivePublicKey()`
- `internal/wireguard/parser.go` - Parser pour fichiers .conf + PresharedKey
- `internal/wireguard/conffile.go` - Génération fichier .conf avec PresharedKey
- `cmd/anemone/main.go` - Appel AutoConnect au démarrage
- `internal/wireguard/status.go` - Récupération statut détaillé (handshake, transfer)
- `internal/backup/backup.go` - Ajout WireGuardBackup struct + export
- `internal/setup/restore.go` - Ajout restoreWireGuard()
- `install.sh` - Ajout `install_wireguard()` + règles sudoers wg-quick
- `web/templates/dashboard_admin.html` - Ajout tuile WireGuard
- `internal/i18n/locales/en.json` - Traductions WireGuard (EN) + edit/public_key
- `internal/i18n/locales/fr.json` - Traductions WireGuard (FR) + edit/public_key
- `internal/web/handlers_admin_wireguard.go` - Handlers WireGuard + `handleAdminWireGuardEdit`
- `internal/web/router.go` - Routes `/admin/wireguard/*`
- `web/templates/admin_wireguard.html` - Template avec Edit modal + PublicKey display

### Décisions techniques

| Décision | Choix | Raison |
|----------|-------|--------|
| Génération clés | `wg genkey` / `wg pubkey` | Standard WireGuard, sécurisé |
| Port par défaut | 51820 | Standard WireGuard |
| Réseau VPN | 10.0.0.0/24 | Plage privée non-routable |
| Stockage clés | DB chiffrée (comme sync keys) | Cohérent avec le reste |

**Pour démarrer Module 1:** Attente validation du plan

---

## Completed: Session 5 - Audit CLAUDE.md ✅

**Résultat:**
- Logging system validé (v0.13.1-beta)
- Refactoring des 4 fichiers > 800 lignes terminé (v0.13.2-beta)

---

## Release v0.13.2-beta (2026-01-30) ✅

### File Refactoring
- https://github.com/juste-un-gars/anemone/releases/tag/v0.13.2-beta
- 4 fichiers > 800 lignes refactorisés pour conformité CLAUDE.md

4 fichiers refactorisés :

| Fichier original | Avant | Après | Nouveaux fichiers |
|-----------------|-------|-------|-------------------|
| sync.go | 1273 | 431 | sync_incremental.go (606), sync_archive.go (279) |
| handlers_admin_storage.go | 1152 | 246 | handlers_admin_storage_zfs.go (682), handlers_admin_storage_disk.go (259) |
| handlers_restore.go | 1002 | 794 | handlers_restore_warning.go (221) |
| handlers_sync_api.go | 915 | 574 | handlers_sync_api_read.go (361) |

Tous les fichiers sont maintenant conformes aux guidelines CLAUDE.md (< 800 lignes)

---

## Completed Today (2026-01-30): Logging System ✅

### Session 4 - Migration logs ✅

- Migrated ~40 files from `log` → `logger` package
- All web handlers now use `logger.Info/Warn/Error`
- All internal packages migrated
- Only 2 `log.Fatalf` remain in main.go (before logger init - intentional)
- Build verified, all tests pass

## Completed: Session 3 - UI Admin Logs (2026-01-30) ✅

- Created `internal/web/handlers_admin_logs.go` - Handlers for log management page
- Created `web/templates/admin_logs.html` - Template with level selector + file list
- Added routes `/admin/logs`, `/admin/logs/level`, `/admin/logs/download`
- Added dashboard card for System Logs
- Added i18n translations (FR + EN)
- Build verified, tests pass

## Completed: Session 2 - Config + DB (2026-01-30) ✅

- Added `LogLevel`, `LogDir` to `internal/config/config.go`
- Added `ANEMONE_LOG_LEVEL`, `ANEMONE_LOG_DIR` env vars
- Added `GetLogLevel()`/`SetLogLevel()` to `internal/sysconfig/sysconfig.go`
- Initialized logger in `cmd/anemone/main.go` (early init + DB level update)
- Build verified

## Completed: Session 1 - Logger Infrastructure (2026-01-30) ✅

- Created `internal/logger/logger.go` - Core logger with slog
- Created `internal/logger/rotation.go` - Daily rotation with retention
- Created `internal/logger/logger_test.go` - Unit tests
- All tests pass (5/5), build verified

---

## Planned: Logging System + Audit

### Context (2026-01-30)

- Mis à jour CLAUDE.md v2.0.0 → v3.0.0 (ajout Quick Reference, File Size Guidelines, Security Audit amélioré)
- Exploration du code : 622 occurrences de `log.` à migrer, aucun système de niveaux
- Décisions techniques prises (voir ci-dessous)

### Décisions techniques

| Paramètre | Valeur |
|-----------|--------|
| Package | `log/slog` (Go 1.21+ standard) |
| Niveau défaut | WARN |
| Rétention | 1 mois **ou** 200 Mo (premier atteint) |
| Persistence niveau | **DB** (table settings, persiste après redémarrage) |
| Override env | `ANEMONE_LOG_LEVEL` (priorité sur DB) |
| Format | Texte lisible avec timestamp |
| Rotation | Quotidienne (1 fichier/jour) |
| Destination | stdout + fichier (`/srv/anemone/logs/`) |

### Sessions planifiées

| # | Session | Objectif | Status |
|---|---------|----------|--------|
| 1 | **Logger Infrastructure** | Créer `internal/logger/` avec slog, niveaux, rotation | ✅ Done |
| 2 | **Config + DB** | Variables LOG_*, init dans main.go | ✅ Done |
| 3 | **UI Admin Logs** | Page `/admin/logs` : changer niveau, télécharger fichiers | ✅ Done |
| 4 | **Migration logs** | Migrer ~40 fichiers `log.` → `logger.` | ✅ Done |
| 5 | **Audit CLAUDE.md** | Vérifier conformité code vs nouvelles règles + refactoring | ✅ Done |

### Format logs prévu

```
2026-01-30 14:32:15 [INFO]  Starting Anemone NAS...
2026-01-30 14:32:15 [INFO]  Loaded 12 users, 3 peers
2026-01-30 14:32:16 [WARN]  Peer "backup-server" unreachable
2026-01-30 14:33:01 [ERROR] Sync failed: connection timeout
```

### UI Admin prévue

```
/admin/logs
├── Niveau actuel : [DEBUG] [INFO] [WARN ✓] [ERROR]  ← sélection
├── Fichiers disponibles :
│   ├── anemone-2026-01-30.log  (2.3 MB) [Télécharger]
│   └── ...
└── [Purger anciens logs]
```

**Pour démarrer** : `"continue session logging"` ou `"session 1: logger infrastructure"`

---

## Completed: Documentation Update (2026-01-26)

### Tâches complétées

#### 1. CHANGELOG.md ✅
- [x] v0.11.5-beta - Mount Disk + Persistent fstab
- [x] v0.11.7-beta - Shared access option, UID/GID fix, Trash fix
- [x] v0.11.8-beta - Format disk dialog fix
- [x] v0.11.9-beta - USB drives on NVMe systems
- [x] v0.12.0-beta - USB Backup refactoring (backup type, share selection)
- [x] v0.13.0-beta - USB Backup automatic scheduling
- [x] Liens de comparaison mis à jour

#### 2. docs/usb-backup.md ✅ (nouveau fichier)
- [x] Introduction et cas d'usage
- [x] Détection des disques USB
- [x] Configuration d'un backup (nom, mount path, backup path)
- [x] Types de backup (Config only vs Config + Data)
- [x] Sélection des shares à sauvegarder
- [x] Planification automatique (interval/daily/weekly/monthly)
- [x] Synchronisation manuelle
- [x] Éjection sécurisée
- [x] Dépannage

#### 3. docs/user-guide.md ✅
- [x] USB Backup (résumé avec lien vers docs/usb-backup.md)
- [x] Storage management (formatage, mount/unmount)

#### 4. README.md ✅
- [x] "USB Backup" et "Storage management" dans Features
- [x] Lien vers docs/usb-backup.md dans la table Documentation

#### 5. docs/README.md ✅
- [x] Lien vers usb-backup.md dans la section Guides

#### 6. docs/p2p-sync.md ✅
- [x] Section Scheduler mise à jour avec les nouvelles options (daily/weekly/monthly)

---

## Release v0.13.0-beta (2026-01-26) ✅

### New Features - USB Backup Automatic Scheduling
- **Automatic sync scheduling**: Schedule USB backups to run automatically
  - **Interval mode**: Every 15min, 30min, 1h, 2h, 4h, 8h, 12h, or 24h
  - **Daily mode**: Every day at a specific time (HH:MM)
  - **Weekly mode**: Every week on a specific day and time
  - **Monthly mode**: Every month on a specific day (1-28) and time
- **Schedule configuration UI**: New section in USB backup edit form
  - Enable/disable toggle for automatic sync
  - Frequency selector with conditional fields
  - Time picker for daily/weekly/monthly modes
  - Day selector for weekly/monthly modes

### Technical Changes
- New DB columns: `sync_enabled`, `sync_frequency`, `sync_time`, `sync_day_of_week`, `sync_day_of_month`, `sync_interval_minutes`
- New `USBBackup.ShouldSync()` function for schedule evaluation
- New `StartScheduler(db, dataDir)` in `internal/usbbackup/scheduler.go`
- Scheduler runs every minute, checks all enabled backups
- Updated `Create`, `Update`, `GetByID`, `GetAll`, `GetEnabled` functions
- New template functions: `deref`, `iterate`
- New i18n keys for schedule UI (FR + EN)

---

## Release v0.12.0-beta (2026-01-26) ✅

### New Features - USB Backup Refactoring
- **Backup type selection**: Choose between "Config only" or "Config + Data"
  - **Config only**: Backs up DB, certificates, smb.conf (~10 MB) - fits on any USB drive
  - **Config + Data**: Config + selected user shares
- **Share selection**: Choose which shares to backup (instead of all shares)
  - Shows share size to help estimate required space
  - No more risk of running out of space on small USB drives
- **Estimated size display**: See share sizes before starting backup

### Technical Changes
- New DB columns: `backup_type`, `selected_shares` in `usb_backups` table
- New `SyncConfig()` function for config-only backups
- `SyncAllShares()` now respects selected shares
- New helper functions: `GetSelectedShareIDs()`, `IsShareSelected()`, `CalculateDirSize()`
- Updated UI with backup type radio buttons and share checkboxes

---

## Release v0.11.9-beta (2026-01-26) ✅

### Bug Fixes
- Fixed: USB drives not detected in "USB Backup" section on NVMe systems
  - The code was hardcoded to exclude `/dev/sda*` assuming it's always the system disk
  - On NVMe systems, the OS is on `/dev/nvme0n1` and USB drives appear as `/dev/sda`
  - Now dynamically detects the system disk via `findmnt /`
  - Any disk mounted in `/mnt/`, `/media/`, or `/run/media/` is now correctly detected

---

## Release v0.11.8-beta (2026-01-26) ✅

### Bug Fixes
- Fixed: "Format disk" dialog was missing "Persistent mount" option
  - Now includes all three options: Mount after format, Shared access, Persistent mount
  - All checked by default for convenience
- Note: Requires sudoers rule for `tee -a /etc/fstab` (added in install.sh, manual add needed for older installs)

---

## Release v0.11.7-beta (2026-01-26) ✅

### New Features
- **Shared access option for disk mount** - New checkbox to allow all users read/write access
  - Available in both "Mount disk" and "Format disk" dialogs
  - Uses `umask=000` for FAT/exFAT, `chmod 777` for ext4/XFS
  - Checked by default for convenience

### Bug Fixes
- Fixed: Persistent mount (fstab) used hardcoded uid=1000,gid=1000 instead of actual user UID/GID
  - FAT32/exFAT disks mounted via fstab now use the correct user permissions
- Fixed: Trash listing now shows actual deleted files with full relative path
  - Previously showed parent directories instead of files (due to Samba keeptree=yes)
  - Restore now works correctly to original location

---

## Hotfix v0.11.5-beta (2026-01-25)

- Fixed: Version number in code was still `0.11.4-beta` instead of `0.11.5-beta`
- Fixed: GitHub release was in Draft mode, not published
- Fixed: Git tag pointed to wrong commit

---

## Release v0.11.5-beta (2026-01-25) ✅

### Mount Disk Feature
- New "Mount" button for formatted but unmounted disks
- Mount path selection dialog with validation
- **Persistent mount option** - adds entry to /etc/fstab using UUID
- Supports FAT/exFAT filesystems with proper uid/gid options

### UI Improvements
- Combined Filesystem and Status columns to reduce table width
- Mounted disks show: mount point + (filesystem type)
- Unmounted formatted disks show: filesystem badge + "(not mounted)"
- Fixed horizontal scrollbar issue on Physical Disks table

### Bug Fixes
- Mount point directory now removed after unmount
- Added missing sudoers rules for mount with options, rmdir, fstab

---

## Release v0.11.4-beta (2026-01-25) ✅

### Disk Mount Status & Permissions
- New "Status" column showing mount point with green icon
- Unmount/Eject buttons for mounted disks in Storage page
- FAT32/exFAT now mounted with correct user permissions (uid/gid)
- ext4/XFS ownership set via chown after mounting

### Security & Validation
- Frontend validation for mount path (/mnt/ or /media/ only)
- Added auth check on unmount endpoint
- Updated sudoers for mount -o and chown

---

## Release v0.11.3-beta (2026-01-25) ✅

### Mount After Formatting
- Option to automatically mount disk after formatting
- Customizable mount path (default: /mnt/{diskname})
- Checkbox enabled by default in format dialog

### Eject Disk Button
- New eject button in USB Backup page
- Safely unmounts and ejects USB drives

### Installer Improvements
- `install.sh` now installs `dosfstools` (FAT32) and `exfatprogs` (exFAT)
- Added sudoers permissions for: mount, umount, eject, mkfs.vfat, mkfs.exfat

---

## Release v0.11.2-beta (2026-01-25) ✅

- Consolidated disk formatting in Storage section (ext4, XFS, exFAT, FAT32)
- USB Backup section now links to Storage for formatting

---

## Release v0.11.1-beta (2026-01-25) ✅

- Fixed SQLite database locking (added WAL mode + busy_timeout)
- Fixed auto-update script failing on existing git tags

---

## Release v0.11.0-beta (2026-01-25) ✅

Session 76 merged, released and pushed to GitHub. Features:

### USB Disk Formatting (Session 76)
- Format USB drives directly from web UI
- FAT32 and exFAT support (Windows-compatible)
- Automatic detection of unmounted disks
- Safe formatting with device validation

### Bug Fixes
- **NVMe SMART data**: Fixed detection for NVMe drives (different protocol than ATA/SATA)

---

## Release v0.10.0-beta (2026-01-24)

Sessions 71-74 merged and released. Major features:

### USB Backup Module (Session 74)
- New `internal/usbbackup/` package
- Auto-detection of USB/external drives
- Encrypted backups (AES-256-GCM)
- Manifest-based incremental sync
- Web UI in admin dashboard

### Setup Wizard - Import Existing Installation (Session 71)
- New wizard option to recover after OS reinstall
- Validates existing DB, recreates system users
- Regenerates Samba config automatically

### Installation Script - Repair Mode (Session 73)
- Option 2 in `install.sh` for recovery
- Auto-detects data directory from `anemone.env`

### Other Changes
- Hide dot files in Samba shares
- Fixed systemd DATA_DIR hardcoded bug
- `.needs-setup` marker as single source of truth

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 19 | Web File Browser | 2026-02-14 | Complete ✅ |
| 18 | Dashboard Last Backup Fix + Recent Tab | 2026-02-12 | Complete ✅ |
| 17 | Rclone Crypt Fix + !BADKEY Logs | 2026-02-11 | Complete ✅ |
| 16 | SSH Key Bugfix | 2026-02-11 | Complete ✅ |
| 15 | Rclone & UI Bugfixes | 2026-02-10 | Complete ✅ |
| 14 | v2 UI Bugfixes | 2026-02-10 | Complete ✅ |
| 13 | Cloud Backup Multi-Provider + Chiffrement | 2026-02-10 | Complete ✅ |
| 12 | Module F Cleanup | 2026-02-08 | Complete ✅ |
| 11 | V2 UI Redesign | 2026-02-08 | Complete ✅ (A-D + F) |
| 10 | USB Backup Mount Fix | 2026-02-08 | Complete ✅ |
| 9 | Documentation Update & Cleanup | 2026-01-31 | Complete ✅ |
| 8 | Rclone SSH Key Generation | 2026-01-31 | Complete ✅ |
| 7 | Rclone Cloud Backup | 2026-01-31 | Complete ✅ |
| 6 | WireGuard Integration | 2026-01-30 | Complete ✅ |
| 5 | Audit CLAUDE.md + Refactoring | 2026-01-30 | Completed ✅ |
| 77 | Mount Disk + Persistent fstab | 2026-01-25 | Completed ✅ |
| 76 | USB Format + NVMe SMART Fix | 2026-01-25 | Completed ✅ |
| 75 | Release v0.10.0-beta | 2026-01-24 | Completed ✅ |
| 74 | USB Backup Module | 2026-01-24 | Completed ✅ |
| 73 | Repair Mode in install.sh | 2026-01-24 | Completed ✅ |
| 72 | Setup Detection Refactor (.needs-setup) | 2026-01-23 | Completed ✅ |
| 71 | Import Existing Installation | 2026-01-23 | Completed ✅ |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [x] Test ZFS new pool → Fixed systemd DATA_DIR bug
- [ ] Test repair mode (install.sh option 2) → simulate-reinstall.sh created
- [x] Test restauration complète → Fixed login bug
- [ ] Verify hide dot files works after Samba reload
- [x] **Test USB Backup module** ✅ 2026-01-30
- [x] **Test USB Format feature** ✅ 2026-01-30
- [x] **Test NVMe SMART display** ✅ 2026-01-30
- [x] **Test WireGuard import** ✅ 2026-01-31
- [x] **Test WireGuard connect/disconnect** ✅ 2026-01-31
- [x] **Test WireGuard PresharedKey** ✅ 2026-01-31
- [x] **Test WireGuard edit config** ✅ 2026-01-31
- [ ] **Test Rclone SSH key generation** ← En cours
- [ ] **Test Rclone sync FR1 → FR2**

---

## Future Features

### WireGuard Integration ✅ Complete (v0.13.3-beta)
- [x] Proposer installation WireGuard au début de `install.sh`
- [x] Nouvelle tuile dans dashboard admin
- [x] Interface web pour gérer la configuration (import, edit, delete)
- [x] Génération de fichiers de configuration `.conf`
- [x] Statut de connexion VPN dans le dashboard
- [x] Support PresharedKey
- [x] Affichage clé publique client (pour config serveur)
- [x] Auto-start au démarrage

### Simple Sync Peers (rclone) ✅ Complete (v0.13.4-beta)
- [x] Nouveau module `internal/rclone/` (séparé des peers)
- [x] Synchronisation unidirectionnelle Anemone → destination externe
- [x] Support SFTP (premier backend implémenté)
- [x] Configuration simplifiée pour utilisateurs ne souhaitant pas le P2P complet
- [x] Planification des sauvegardes (interval/daily/weekly/monthly)
- [ ] Support backends additionnels (S3, Google Drive, etc.) - à faire
- [ ] Installation rclone dans install.sh (curl script officiel)
- [ ] Vérification version rclone + notification admin si mise à jour disponible

### Local Backup (USB/External Drive)

**Niveau 1 : Data Backup** ✅ Session 74
- [x] Nouveau module `internal/usbbackup/` (séparé des peers)
- [x] Détection automatique des disques USB/externes connectés
- [x] Interface web pour configurer les sauvegardes
- [x] Synchronisation chiffrée avec manifest
- [ ] Planification automatique (à faire)
- [ ] Auto-sync quand disque branché (à faire)

**Niveau 2 : Config Backup (léger)** - À faire
- [ ] Export de la configuration : DB, certificats, config Samba
- [ ] Chiffrement avec mot de passe
- [ ] Restauration depuis le Setup Wizard

### USB Drive Management ✅ Session 76
- [x] Formatage des disques USB depuis l'interface (FAT32/exFAT)
- [ ] État des disques dans le dashboard
- [ ] Gestion de la mise en veille

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[README.md](README.md)** - Quick start
- **[docs/](docs/)** - Full documentation
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

---

## Next Steps

1. **V2 UI Redesign — Module E : Pages auth** (optionnel)
   - Migrer login, setup, activate, reset_password vers v2
   - Style séparé (pas de sidebar, page centrée)
2. **API REST JSON pour gestion courante** (optionnel)
   - Users, Peers, Shares, Settings n'ont pas d'API JSON (form HTML uniquement)
   - Storage/ZFS et P2P Sync ont déjà des API JSON complètes (`docs/API.md`)

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
