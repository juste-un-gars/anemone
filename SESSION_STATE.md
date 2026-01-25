# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.11.3-beta
**Last Updated:** 2026-01-25

---

## Current Session

**No active session** - Ready for new work

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
