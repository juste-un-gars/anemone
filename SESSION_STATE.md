# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-24

---

## Current Session

**Session 73** - Repair Mode in install.sh
- **Status:** Completed ✅
- **Date:** 2026-01-24

### Summary
Added repair/reinstall mode to `install.sh` for recovering existing Anemone installations after OS reinstall. Fixed critical bug where systemd service ignored wizard-configured data path.

### Bug Fixes
- **Critical:** Removed hardcoded `ANEMONE_DATA_DIR` from systemd service - was overriding the wizard's configured path in `/etc/anemone/anemone.env`
- SMB access denied after repair - fixed path generation bug (was reconstructing path instead of using DB path)
- Added `hide dot files = yes` to Samba config to hide `.anemone` and `.trash`

### Changes
- [x] Added menu in `install.sh`: New installation / Repair
- [x] Repair mode reads existing DB, recreates system users, fixes permissions
- [x] Repair mode regenerates Samba configuration from DB
- [x] Removed "Import existing installation" option from wizard HTML
- [x] Removed `.needs-setup` marker file logic (no longer needed)
- [x] Simplified `isSetupCompleted()` to check DB existence
- [x] Deleted `scripts/cleanup-keep-data.sh` (obsolete)
- [x] Created `scripts/simulate-reinstall.sh` for testing repair mode
- [x] Fixed systemd service to read DATA_DIR from env file (not hardcoded)
- [x] Added `hide dot files = yes` in Samba share config

### New Logic
```
install.sh → Menu:
  1) New installation    → wizard configures path → writes /etc/anemone/anemone.env
  2) Repair/Reinstall    → reads DB, recreates users, direct login

Service reads ANEMONE_DATA_DIR from /etc/anemone/anemone.env (written by wizard)
```

### Files Modified
- `install.sh` - Removed hardcoded ANEMONE_DATA_DIR from systemd service
- `internal/smb/smb.go` - Added `hide dot files = yes`
- `scripts/simulate-reinstall.sh` - New script for testing repair mode

### Deleted Files
- `scripts/cleanup-keep-data.sh`

### Note
After repair, users must reset their password via web interface to restore SMB access.

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
- [x] Test ZFS new pool → Fixed systemd DATA_DIR bug
- [ ] Test repair mode (install.sh option 2) → simulate-reinstall.sh created
- [x] Test restauration complète → Fixed login bug
- [ ] Verify hide dot files works after Samba reload

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

### Local Backup (USB/External Drive)

**Niveau 1 : Config Backup (léger)**
- [ ] Détection automatique des disques USB/externes connectés
- [ ] Interface web pour sélectionner le disque de sauvegarde
- [ ] Export de la configuration : DB, certificats, config Samba
- [ ] Chiffrement avec mot de passe
- [ ] Restauration depuis le Setup Wizard

**Niveau 2 : Data Backup (Local Peer)**
- [ ] Nouveau type de peer : "Local" (disque externe monté)
- [ ] Synchronisation des shares uniquement (pas incoming - c'est déjà des backups)
- [ ] Réutilise la logique de sync incrémentale/manifest existante
- [ ] Planification (quotidien, hebdomadaire...)
- [ ] Interface unifiée avec les peers réseau

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
