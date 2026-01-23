# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-23

---

## Current Session

**Session 72** - Setup Detection Refactor (.needs-setup)
- **Status:** In Progress 🔄
- **Date:** 2026-01-23

### Summary
Refactored setup detection to use a simple `.needs-setup` marker file instead of database flag. Added cleanup script for testing import flow.

### Completed (2026-01-23)
- [x] Created `scripts/cleanup-keep-data.sh` - simulates OS reinstall while preserving data
- [x] Added `.needs-setup` marker file detection in `IsSetupNeeded()`
- [x] `LoadState()` activates setup mode when `.needs-setup` exists
- [x] `Cleanup()` removes both `.setup-state.json` and `.needs-setup`
- [x] Removed all `setup_completed` references from database
- [x] `isSetupCompleted()` now checks absence of `.needs-setup` instead of DB

### New Setup Logic
```
.needs-setup exists     → setup wizard needed
.needs-setup absent     → normal mode (setup completed)
```

### Files Modified
- `internal/setup/setup.go` - `.needs-setup` detection in IsSetupNeeded() and LoadState()
- `internal/web/router.go` - isSetupCompleted() uses file instead of DB
- `internal/web/handlers_setup.go` - removed DB write for setup_completed
- `internal/setup/restore.go` - removed DB write for setup_completed
- `internal/setup/finalize.go` - removed setup_completed from config map
- `scripts/cleanup-keep-data.sh` - new script for testing import flow

### Remaining
- [ ] Test complete import flow on VM

---

## Previous Session

**Session 71** - Import Existing Installation
- **Status:** Completed ✅
- **Date:** 2026-01-23

### Summary
Added a new setup wizard option to import an existing Anemone installation (e.g., after OS reinstall).

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
| 72 | Setup Detection Refactor (.needs-setup) | 2026-01-23 | In Progress |
| 71 | Import Existing Installation | 2026-01-23 | Completed |
| 70 | Enhanced SMART Modal | 2026-01-22 | Completed |
| 69 | Restore Flow Fixes | 2026-01-21 | Completed |
| 68 | Persistent Sessions & Documentation | 2026-01-21 | Completed |
| 67 | Tests VM & Bug Fixes Setup Wizard | 2026-01-21 | Completed |

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
