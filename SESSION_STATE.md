# Anemone - Session State

**Current Version:** v0.9.24-beta
**Last Updated:** 2026-01-21

---

## Current Session

**Session 68** - Persistent Sessions & Documentation
- **Status:** Completed
- **Date:** 2026-01-21

### Summary
Added persistent sessions with "Remember me" feature and rewrote documentation.

### Completed (2026-01-21)
- [x] Persistent sessions stored in SQLite (survive server restarts)
- [x] "Remember me" checkbox on login (30 days vs 2 hours)
- [x] Session tracking (IP address, user agent)
- [x] Auto-cleanup of expired sessions
- [x] Updated tests for new session system
- [x] Simplified README.md (1188 → 87 lines)
- [x] Created full documentation wiki (10 pages in docs/)
- [x] Renamed docs to kebab-case for consistency

### Commits
- `7ed13f8` feat: Add persistent sessions with "Remember me" option
- `316985f` docs: Rewrite README and create full documentation wiki

---

## Previous Session

**Session 67** - Tests VM & Bug Fixes Setup Wizard
- **Status:** Completed
- **Date:** 2026-01-20 to 2026-01-21
- **Details:** [SESSION_067_vm_tests.md](.claude/sessions/SESSION_067_vm_tests.md)

### Summary
Tests du setup wizard sur VM et corrections de bugs chemins personnalisés.

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 68 | Persistent Sessions & Documentation | 2026-01-21 | Completed |
| 67 | Tests VM & Bug Fixes Setup Wizard | 2026-01-21 | Completed |
| 66 | Tests d'intégration Setup Wizard | 2026-01-20 | Completed |
| 65 | Mode Restauration Serveur | 2026-01-20 | Completed |
| 64 | Nouveau install.sh simplifié | 2026-01-20 | Completed |
| 63 | Mode Setup - Frontend | 2026-01-20 | Completed |

---

## Remaining Tests

- [ ] Test complet sur VM Fedora
- [ ] Test ZFS new pool
- [ ] Test ZFS existing pool
- [ ] Test restauration complète

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[README.md](README.md)** - Quick start
- **[docs/](docs/)** - Full documentation
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

---

## Next Steps

**Tests à faire :**
- Tester "Remember me" en production
- Tester les différents modes d'installation (ZFS, restore)

**Sessions planifiées :**
- Session 69 : Tests finaux avant release

Commencer par `"continue"`.
