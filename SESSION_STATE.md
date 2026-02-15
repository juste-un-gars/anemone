# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.15.3-beta (prochaine: v0.20.0-beta)
**Last Updated:** 2026-02-15

---

## Current Session

**Session 22: !BADKEY Fix + Release v0.20.0-beta** - Planned

**Détails :** `.claude/sessions/SESSION_022_badkey_release.md`

---

## Session 22: !BADKEY Fix + Release v0.20.0-beta

**Date:** (prochaine session)
**Objective:** Corriger 482 appels slog printf-style → key-value, release v0.20.0-beta
**Status:** Planned

### Plan
1. **Commiter session 21** — 6 fichiers non commités (conversion API + roundtrip formats + PDF viewer)
2. **Fix !BADKEY** — Convertir ~482 `logger.Info("msg %s", val)` → `logger.Info("msg", "key", val)` dans ~40 fichiers
3. **Version bump** — `0.15.3-beta` → `0.20.0-beta`
4. **Release** — CHANGELOG, tag, GitHub release

### Contenu release v0.20.0-beta
| Session | Feature |
|---------|---------|
| 18 | Dashboard dernières sauvegardes (toutes sources) |
| 19 | Web File Browser (browse, upload, download, mkdir, rename, delete) |
| 20 | OnlyOffice Integration (édition documents dans le navigateur) |
| 21 | OnlyOffice auto-config, conversion API, formats roundtrip-safe, PDF viewer |
| 22 | Fix !BADKEY slog (~482 occurrences) |

---

## Previous Session

**Session 21: OnlyOffice Auto-Config + Bugfixes** - Complete ✅

**Détails :** `.claude/sessions/SESSION_021_onlyoffice_autoconfig.md`

---

## Session 21: OnlyOffice Auto-Config + Bugfixes

**Date:** 2026-02-14 → 2026-02-15
**Objective:** Rendre OnlyOffice configurable depuis l'interface web + corriger les bugs
**Status:** Complete ✅
**Serveur de test :** FR2 (192.168.83.37)

### Completed (22 items)
| # | Type | Description |
|---|------|-------------|
| 1-9 | Feature/Fix | Config OO en DB, auto-config, CSP, routes, base64url key, TLS, !BADKEY, Docker |
| 10-19 | Fix | Download URL HTTP, HTTP server, proxy headers, auto-patch, HTTPS direct port, CSP dynamique, TLS skip verify, sudo mv |
| 20 | Fix | **Conversion API** — Formats non-natifs reconvertis avant sauvegarde |
| 21 | Fix | **Formats roundtrip-safe** — Retrait md, doc, xls, ppt (non-roundtrip) |
| 22 | Feature | **Bouton "Voir"** — PDF/images ouverts dans navigateur |

### Tests validés (FR2)
- [x] Sauvegarde fichier OO ✅
- [x] Bouton "Modifier" absent pour .md ✅
- [x] Bouton "Voir" pour PDF ✅

### 6 fichiers non commités
- `internal/web/handlers_onlyoffice_api.go` — Conversion API + formats roundtrip
- `internal/web/handlers_files.go` — Content-Disposition inline
- `internal/web/router.go` — IsOOEditable + IsViewable
- `web/templates/v2/v2_files.html` — Bouton "Voir"
- `internal/i18n/locales/fr.json` — Clé files.action.view
- `internal/i18n/locales/en.json` — Clé files.action.view

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
- `!BADKEY` dans ~482 appels logger — **session 22**

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 22 | !BADKEY Fix + Release v0.20.0-beta | (next) | Planned |
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

1. **Session 22** — Fix !BADKEY + Release v0.20.0-beta
2. **V2 UI Redesign — Module E : Pages auth** (optionnel)
3. **API REST JSON pour gestion courante** (optionnel)

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
