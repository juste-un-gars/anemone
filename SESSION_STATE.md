# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.20.0-beta
**Last Updated:** 2026-02-15

---

## Current Session

**Session 22: !BADKEY Fix + Bugfixes + Release v0.20.0-beta** - Complete ✅

**Détails :** `.claude/sessions/SESSION_022_badkey_release.md`

---

## Session 22: !BADKEY Fix + Bugfixes + Release v0.20.0-beta

**Date:** 2026-02-15
**Objective:** Corriger !BADKEY slog, fix bugs installation/OnlyOffice, release v0.20.0-beta
**Status:** Complete ✅

### Completed (7 items)
| # | Type | Description |
|---|------|-------------|
| 1 | Fix | !BADKEY slog — ~482 appels printf-style → key-value |
| 2 | Fix | Logs directory manquant dans install.sh |
| 3 | Fix | Docker HTTP security — écoute docker0 bridge uniquement |
| 4 | Chore | Version bump 0.15.3-beta → 0.20.0-beta |
| 5 | Docs | CHANGELOG.md complet |
| 6 | Docs | README.md — téléchargement dernière release |
| 7 | Release | v0.20.0-beta — GitHub release |

### Tests validés (FR2)
- [x] Installation propre
- [x] Service démarre
- [x] OnlyOffice fonctionne
- [x] Compilation, tests, go vet OK

---

## Previous Session

**Session 21: OnlyOffice Auto-Config + Bugfixes** - Complete ✅

**Détails :** `.claude/sessions/SESSION_021_onlyoffice_autoconfig.md`

---

## Previous Session

**Session 20: OnlyOffice Integration** - Complete ✅

**Détails :** `.claude/sessions/SESSION_020_onlyoffice.md`

---

## Previous Session

**Session 19: Web File Browser** - Complete ✅

**Détails :** `.claude/sessions/SESSION_019_file_browser.md`

---

## Previous Session

**Session 18: Dashboard Last Backup Fix + Recent Backups Tab** - Complete ✅

**Détails :** `.claude/sessions/SESSION_018_recent_backups.md`

---

## Bugs connus (non corrigés)
- Permission denied sur manifests marc (shares marc:marc, Anemone tourne en franck)

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 22 | !BADKEY Fix + Bugfixes + Release v0.20.0-beta | 2026-02-15 | Complete ✅ |
| 21 | OnlyOffice Auto-Config + Bugfixes | 2026-02-15 | Complete ✅ |
| 20 | OnlyOffice Integration | 2026-02-14 | Complete ✅ |
| 19 | Web File Browser | 2026-02-14 | Complete ✅ |
| 18 | Dashboard Last Backup Fix + Recent Tab | 2026-02-12 | Complete ✅ |
| 17 | Rclone Crypt Fix + !BADKEY Logs | 2026-02-11 | Complete ✅ |
| 16 | SSH Key Bugfix | 2026-02-11 | Complete ✅ |
| 15 | Rclone & UI Bugfixes | 2026-02-10 | Complete ✅ |
| 14 | v2 UI Bugfixes | 2026-02-10 | Complete ✅ |
| 13 | Cloud Backup Multi-Provider + Chiffrement | 2026-02-10 | Complete ✅ |

---

## Next Steps

1. **V2 UI Redesign — Module E : Pages auth** (optionnel)
2. **API REST JSON pour gestion courante** (optionnel)

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
