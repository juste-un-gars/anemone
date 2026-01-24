# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-24

---

## Current Session

**Session 74** - USB Backup Module
- **Status:** In Progress (awaiting validation)
- **Date:** 2026-01-24
- **Branch:** `feature/backup-modules`

### Summary
Created new `internal/usbbackup/` module for local backup to USB drives and external storage. This is a separate module from P2P sync, following the principle of not modifying existing working code.

### Architecture Decision
- **Approach:** Separate modules (not extending existing peers/sync)
- **Reason:** Minimize risk to working P2P sync, cleaner separation of concerns
- **Shared:** `internal/crypto/` for encryption
- **Duplicated:** Manifest logic (simpler, independent evolution)

### Files Created
- `internal/usbbackup/usbbackup.go` - Structures + CRUD (Create, Get, Update, Delete)
- `internal/usbbackup/detect.go` - USB/external drive detection
- `internal/usbbackup/sync.go` - Encrypted backup with manifest tracking
- `internal/web/handlers_admin_usb.go` - HTTP handlers
- `web/templates/admin_usb_backup.html` - Main USB backup page
- `web/templates/admin_usb_backup_edit.html` - Edit configuration page

### Files Modified
- `internal/database/migrations.go` - Added `usb_backups` table
- `internal/web/router.go` - Added `/admin/usb-backup/*` routes
- `web/templates/dashboard_admin.html` - Added USB Backup card
- `internal/i18n/locales/fr.json` - French translations
- `internal/i18n/locales/en.json` - English translations

### Database Schema
```sql
CREATE TABLE usb_backups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    mount_path TEXT NOT NULL,
    backup_path TEXT DEFAULT 'anemone-backup',
    enabled BOOLEAN DEFAULT 1,
    auto_detect BOOLEAN DEFAULT 1,
    last_sync DATETIME,
    last_status TEXT DEFAULT 'unknown',
    last_error TEXT DEFAULT '',
    files_synced INTEGER DEFAULT 0,
    bytes_synced INTEGER DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
```

### Features Implemented
- [x] USB/external drive detection (`/media/`, `/mnt/`, removable devices)
- [x] Configuration CRUD (add, edit, delete backup configs)
- [x] Manual sync trigger with background execution
- [x] Encrypted file backup (AES-256-GCM via streaming)
- [x] Manifest-based incremental sync (add/update/delete)
- [x] Status tracking (success, error, running, files/bytes synced)
- [x] Web UI integrated in admin dashboard
- [x] FR/EN translations

### Cleanup Done
- Removed debug files: `debug_auth.go`, `fix_hash.go`, `verify_hash.go`, `backup_20260121_154509.enc`

### Pending Validation
- [ ] Test USB detection on real system
- [ ] Test full backup cycle to USB drive
- [ ] Test incremental sync (add/modify/delete files)
- [ ] Verify encrypted files can be restored

### Next Steps (this session)
1. Validate USB Backup module
2. Consider: rclone module or scheduler for USB

---

## Previous Session

**Session 73** - Repair Mode in install.sh
- **Status:** Completed ✅
- **Date:** 2026-01-24

### Summary
Added repair/reinstall mode to `install.sh` for recovering existing Anemone installations after OS reinstall. Fixed critical bug where systemd service ignored wizard-configured data path.

---

## Previous Session

**Session 72** - Setup Detection Refactor (.needs-setup)
- **Status:** Superseded by Session 73
- **Date:** 2026-01-23

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 74 | USB Backup Module | 2026-01-24 | In Progress |
| 73 | Repair Mode in install.sh | 2026-01-24 | Completed |
| 72 | Setup Detection Refactor (.needs-setup) | 2026-01-23 | Superseded |
| 71 | Import Existing Installation | 2026-01-23 | Superseded |
| 70 | Enhanced SMART Modal | 2026-01-22 | Completed |
| 69 | Restore Flow Fixes | 2026-01-21 | Completed |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [x] Test ZFS new pool → Fixed systemd DATA_DIR bug
- [ ] Test repair mode (install.sh option 2) → simulate-reinstall.sh created
- [x] Test restauration complète → Fixed login bug
- [ ] Verify hide dot files works after Samba reload
- [ ] **Test USB Backup module** (new)

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

### USB Drive Management (future)
- [ ] Formatage des disques USB depuis l'interface
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

1. Valider le module USB Backup (Session 74)
2. Module rclone pour backup cloud
3. Scheduler pour USB auto-sync

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
