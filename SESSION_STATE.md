# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-22

---

## Current Session

**Session 70** - Enhanced SMART Modal
- **Status:** Completed ✅
- **Date:** 2026-01-22

### Summary
Improved the SMART details modal in the storage page with detailed metrics, help tooltips, and visual status indicators.

### Completed (2026-01-22)
- [x] Added NVMe-specific fields to SMARTInfo struct (media errors, unsafe shutdowns, available spare, percentage used, data units read/written)
- [x] Populated NVMe fields in GetSMARTInfo() function
- [x] Redesigned SMART modal with organized sections
- [x] Added help tooltips (?) for each metric with explanations
- [x] Color-coded values (green/yellow/red based on severity)
- [x] Added FR/EN translations for all new labels and help texts
- [x] Collapsible raw attributes table

### Features
The new SMART modal displays:
- Health banner with visual status indicator
- General info (temperature, power-on hours, power cycles)
- For SATA/SSD: Disk errors (reallocated/pending/uncorrectable sectors)
- For NVMe: Media errors, unsafe shutdowns, SSD wear (available spare, percentage used), data volume (TB written/read)
- All raw SMART attributes (collapsible)

### Commits
- `127010b` feat: Enhanced SMART modal with detailed metrics and help tooltips

---

## Previous Session

**Session 69** - Restore Flow Fixes
- **Status:** Completed ✅
- **Date:** 2026-01-21

### Summary
Fixed critical bugs in restore flow: login failure after restore and missing storage configuration.

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 70 | Enhanced SMART Modal | 2026-01-22 | Completed |
| 69 | Restore Flow Fixes | 2026-01-21 | Completed |
| 68 | Persistent Sessions & Documentation | 2026-01-21 | Completed |
| 67 | Tests VM & Bug Fixes Setup Wizard | 2026-01-21 | Completed |
| 66 | Tests d'intégration Setup Wizard | 2026-01-20 | Completed |
| 65 | Mode Restauration Serveur | 2026-01-20 | Completed |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [ ] Test ZFS new pool
- [ ] Test ZFS existing pool
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
