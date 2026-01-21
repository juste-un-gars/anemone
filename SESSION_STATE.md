# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-21

---

## Current Session

**Session 69** - Restore Flow Fixes
- **Status:** In Progress (paused)
- **Date:** 2026-01-21

### Summary
Fixed critical bugs in restore flow: login failure after restore and missing storage configuration.

### Completed (2026-01-21)
- [x] Updated docs/storage-setup.md (French → English, ZFS only)
- [x] Fixed restore flow to include storage configuration (was hardcoded to /srv/anemone)
- [x] Added post-restore system setup (SMB users, password hashes, share directories, Samba config)
- [x] Fixed NULL field handling in users.go (root cause of login failure after restore)

### Root Cause Analysis
Login failed after restore because:
1. Restored users have NULL email field in database
2. `GetByUsername()` tried to scan NULL into a Go string → crash
3. Fixed by using `sql.NullString` for nullable fields

### Commits
- `1e18fd1` (earlier) Restore flow with storage config, SMB setup
- `852cad7` fix: Handle NULL fields when reading users from database

### To Test
After deploying (`sudo cp anemone /usr/local/bin/ && sudo systemctl restart anemone`):
1. Use existing backup file to restore
2. Login should now work
3. Samba should work (was already fixed)

---

## Previous Session

**Session 68** - Persistent Sessions & Documentation
- **Status:** Completed
- **Date:** 2026-01-21

### Summary
Added persistent sessions with "Remember me" feature and rewrote documentation.

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 69 | Restore Flow Fixes | 2026-01-21 | In Progress |
| 68 | Persistent Sessions & Documentation | 2026-01-21 | Completed |
| 67 | Tests VM & Bug Fixes Setup Wizard | 2026-01-21 | Completed |
| 66 | Tests d'intégration Setup Wizard | 2026-01-20 | Completed |
| 65 | Mode Restauration Serveur | 2026-01-20 | Completed |
| 64 | Nouveau install.sh simplifié | 2026-01-20 | Completed |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [ ] Test ZFS new pool
- [ ] Test ZFS existing pool
- [x] Test restauration complète → Fixed login bug

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[README.md](README.md)** - Quick start
- **[docs/](docs/)** - Full documentation
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

---

## Next Steps

**À tester :**
- Déployer le fix et tester login après restore
- Tester les différents modes d'installation (ZFS, restore avec stockage personnalisé)

**Fichiers debug à nettoyer (optionnel) :**
- `debug_auth.go`, `fix_hash.go`, `verify_hash.go`, `backup_20260121_154509.enc`
- `/tmp/check_backup.go`

Commencer par `"continue"`.
