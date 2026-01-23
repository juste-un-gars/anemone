# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-23

---

## Current Session

**Session 71** - Import Existing Installation
- **Status:** Completed ✅
- **Date:** 2026-01-23

### Summary
Added a new setup wizard option to import an existing Anemone installation (e.g., after OS reinstall). This replaces the redundant "existing ZFS pool" option with a more useful feature.

### Completed (2026-01-23)
- [x] Removed redundant "zfs_existing" storage option (use "custom path" instead)
- [x] Added new "import_existing" storage option in setup wizard
- [x] Created `FinalizeImport()` function for importing existing installations
- [x] Validates that `db/anemone.db` exists at specified path
- [x] Writes `/etc/anemone/anemone.env` with correct data directory
- [x] Updates sudoers for custom paths
- [x] Regenerates `smb.conf` from existing database
- [x] Recreates system users and Samba accounts (decrypts passwords using master_key from DB)
- [x] Fixes ownership (chown) on share directories
- [x] Skips admin creation step (uses existing admin from DB)
- [x] Added FR/EN translations for new UI elements

### Features
The "Import existing installation" option:
- Detects and validates existing Anemone database
- Automatically recreates all system users with their original passwords
- Restores Samba access without manual intervention
- Works like a restoration but without needing a backup file

### Files Modified
- `internal/setup/storage.go` - Added import_existing option, removed zfs_existing
- `internal/setup/finalize.go` - Added FinalizeImport() function
- `internal/web/handlers_setup_wizard.go` - Handle import_existing flow
- `web/templates/setup_wizard.html` - UI for import_existing option
- `internal/i18n/locales/{en,fr}.json` - Translations

---

## Previous Session

**Session 70** - Enhanced SMART Modal
- **Status:** Completed ✅
- **Date:** 2026-01-22

### Summary
Improved the SMART details modal in the storage page with detailed metrics, help tooltips, and visual status indicators.

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 71 | Import Existing Installation | 2026-01-23 | Completed |
| 70 | Enhanced SMART Modal | 2026-01-22 | Completed |
| 69 | Restore Flow Fixes | 2026-01-21 | Completed |
| 68 | Persistent Sessions & Documentation | 2026-01-21 | Completed |
| 67 | Tests VM & Bug Fixes Setup Wizard | 2026-01-21 | Completed |
| 66 | Tests d'intégration Setup Wizard | 2026-01-20 | Completed |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [ ] Test ZFS new pool
- [ ] Test import existing installation
- [x] Test restauration complète → Fixed login bug

---

## Future Features

### WireGuard Integration
- [ ] Installation automatique du client WireGuard lors de l'installation d'Anemone
- [ ] Interface web pour gérer la configuration WireGuard (clés, endpoints, peers)
- [ ] Génération de fichiers de configuration `.conf`
- [ ] Statut de connexion VPN dans le dashboard

### Simple Sync Peers (rclone)
- [ ] Nouveau type de pair : "Simple Sync" (en plus du P2P existant)
- [ ] Synchronisation unidirectionnelle Anemone → destination externe
- [ ] Support rclone pour multiples backends (S3, SFTP, Google Drive, etc.)
- [ ] Configuration simplifiée pour utilisateurs ne souhaitant pas le P2P complet
- [ ] Planification des sauvegardes simples

### USB Configuration Backup
- [ ] Détection automatique des clés USB connectées au serveur
- [ ] Interface web pour sélectionner la clé USB de sauvegarde
- [ ] Export de la configuration complète (DB, certificats, config Samba)
- [ ] Chiffrement de la sauvegarde avec mot de passe (défaut configurable)
- [ ] Restauration depuis clé USB dans le Setup Wizard
- [ ] Sauvegarde automatique programmable (quotidienne/hebdomadaire)

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[README.md](README.md)** - Quick start
- **[docs/](docs/)** - Full documentation
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

---

## Next Steps

**Fichiers debug à nettoyer (optionnel) :**
- `debug_auth.go`, `fix_hash.go`, `verify_hash.go`, `backup_20260121_154509.enc`

Commencer par `"continue"`.
