# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.13.1-beta
**Last Updated:** 2026-01-30

---

## Current Session

**Session 5: Audit CLAUDE.md** - En attente de validation logging

**Prochaines étapes:**
1. Tester logging system (v0.13.1-beta)
2. Si OK → Refactoring des fichiers > 800 lignes

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
| 5 | **Audit CLAUDE.md** | Vérifier conformité code vs nouvelles règles | ⏳ Next |

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
- [ ] **Test USB Backup module**
- [ ] **Test USB Format feature** (new)
- [ ] **Test NVMe SMART display** (new)

---

## Future Features

### WireGuard Integration
- [ ] Installation automatique du client WireGuard lors de l'installation d'Anemone
- [ ] Interface web pour gérer la configuration WireGuard (clés, endpoints, peers)
- [ ] Génération de fichiers de configuration `.conf`
- [ ] Statut de connexion VPN dans le dashboard

### Simple Sync Peers (rclone)
- [ ] Nouveau module `internal/rclone/` (séparé des peers)
- [ ] Synchronisation unidirectionnelle Anemone → destination externe
- [ ] Support rclone pour multiples backends (S3, SFTP, Google Drive, etc.)
- [ ] Configuration simplifiée pour utilisateurs ne souhaitant pas le P2P complet
- [ ] Planification des sauvegardes simples

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

1. Tester le module USB Backup sur un vrai disque
2. Tester le formatage USB (FAT32/exFAT)
3. Tester l'affichage SMART NVMe
4. Module rclone pour backup cloud
5. WireGuard integration

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
