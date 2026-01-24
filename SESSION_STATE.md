# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-24

---

## Current Session

**Session 73** - Repair Mode in install.sh
- **Status:** Completed ✅
- **Date:** 2026-01-24

### Summary
Added repair/reinstall mode to `install.sh` for recovering existing Anemone installations after OS reinstall. Removed the "Import existing installation" option from the web wizard - repair is now handled entirely by the install script.

### Changes
- [x] Added menu in `install.sh`: New installation / Repair
- [x] Repair mode reads existing DB, recreates system users, fixes permissions
- [x] Repair mode regenerates Samba configuration from DB
- [x] Removed "Import existing installation" option from wizard HTML
- [x] Removed `.needs-setup` marker file logic (no longer needed)
- [x] Simplified `isSetupCompleted()` to check DB existence
- [x] Deleted `scripts/cleanup-keep-data.sh` (obsolete)
- [x] Cleaned up translations (fr.json, en.json)

### New Logic
```
install.sh → Menu:
  1) New installation    → wizard
  2) Repair/Reinstall    → reads DB, recreates users, direct login

Setup needed if:
  - No database exists
  - ANEMONE_SETUP_MODE=true
  - Setup state file active but not finalized
```

### Files Modified
- `install.sh` - Added repair mode with menu, user recreation, Samba regeneration
- `web/templates/setup_wizard.html` - Removed import_existing option
- `internal/i18n/locales/fr.json` - Removed import translations
- `internal/i18n/locales/en.json` - Removed import translations
- `internal/setup/setup.go` - Removed .needs-setup logic
- `internal/web/router.go` - isSetupCompleted() checks DB existence
- `internal/web/handlers_setup.go` - Updated comments
- `internal/setup/restore.go` - Updated comments
- `internal/setup/finalize.go` - Updated comments

### Deleted Files
- `scripts/cleanup-keep-data.sh`

---

## Previous Session

**Session 72** - Setup Detection Refactor (.needs-setup)
- **Status:** Superseded by Session 73
- **Date:** 2026-01-23

### Summary
Attempted to use `.needs-setup` marker file for setup detection. Approach was replaced by repair mode in install.sh.

---

## Previous Session

**Session 71** - Import Existing Installation
- **Status:** Superseded by Session 73
- **Date:** 2026-01-23

### Summary
Added wizard option to import existing installation. Replaced by repair mode in install.sh.

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
| 73 | Repair Mode in install.sh | 2026-01-24 | Completed |
| 72 | Setup Detection Refactor (.needs-setup) | 2026-01-23 | Superseded |
| 71 | Import Existing Installation | 2026-01-23 | Superseded |
| 70 | Enhanced SMART Modal | 2026-01-22 | Completed |
| 69 | Restore Flow Fixes | 2026-01-21 | Completed |
| 68 | Persistent Sessions & Documentation | 2026-01-21 | Completed |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [ ] Test ZFS new pool
- [ ] Test repair mode (install.sh option 2)
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
