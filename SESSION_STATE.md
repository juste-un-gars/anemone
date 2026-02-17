# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.22.0-beta
**Last Updated:** 2026-02-17

---

## Current Session

**Session 26: Retest Securite FR2** - Complete

---

## Session 26: Retest Securite FR2

**Date:** 2026-02-15
**Objective:** Retest complet securite sur FR2 apres reinstallation
**Status:** Complete (32/32 tests PASS)

### Resultat retest : 32/32 PASS

#### 1. Reconnaissance (4/4)
- [x] TLS TLSv1.3 + ECDSA, X25519
- [x] Headers securite : HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy
- [x] CSP : `unsafe-eval` absent, `object-src 'none'`, `base-uri 'self'`
- [x] HTTP :8080 = connection refused

#### 2. Tests non authentifies (5/5)
- [x] /dashboard, /admin/*, /files, /settings → 303 /login
- [x] Sync API /api/sync/manifest sans auth = 401
- [x] /.env, /.git/config, /debug/pprof = non exposes (303)
- [x] Path traversal /static/../../../etc/passwd = 303
- [x] Path traversal encode %2f = 404

#### 3. Rate limiting + brute-force (3/3)
- [x] 5 echecs /login → 6e = HTTP 429
- [x] Rate limit par username (marc) → 6e = HTTP 429
- [x] PUT/DELETE/PATCH/OPTIONS /login = 405

#### 4. CSRF (2/2)
- [x] POST /login sans CSRF token = 403
- [x] Cookie CSRF : Secure + SameSite=Strict

#### 5. Cookies + sessions (3/3)
- [x] anemone_session : HttpOnly + Secure
- [x] Session sans remember_me = ~2h
- [x] Session avec remember_me = 14 jours

#### 6. Auth timing (1/1)
- [x] User existant ~3.6ms vs inexistant ~4.2ms (ecart <2ms, bruit reseau)

#### 7. Admin (2/2)
- [x] /admin/security accessible admin = 200, contenu OK
- [x] Marc → /admin/* = 403

#### 8. OnlyOffice (3/3)
- [x] Config DB : oo_enabled=true, oo_url=localhost:9980
- [x] Host `<script>` → CSP default (pas d'injection)
- [x] Host `evil.com` → CSP default (session non reconnue)

#### 9. IDOR (5/5)
- [x] Marc → /admin/users, /security, /settings, /logs, /peers = 403
- [x] Marc → /files?path=../../admin = erreurs (pas de fuite)
- [x] Marc → /files?path=/ = erreurs (pas de fuite)
- [x] Marc → /files (propres fichiers) = 200
- [x] Marc → /api/files/list?user=admin = 303

**Rapport complet :** `SECURITY_FIXES_2026-02-15.md`

---

## Session 25: Corrections Securite Prioritaires - Complete

**Date:** 2026-02-15
**Objective:** Corriger les vulns identifiees en session 24
**Status:** Complete (vagues 1-5 done, audit complet, retest FR2 OK)

**Details :** `.claude/sessions/SESSION_025_security_fixes.md`
**Rapport audit :** `SECURITY_AUDIT_2026-02-15.md` (racine, NE PAS committer)

---

## Previous Sessions

**Session 24: Audit de Securite** - Complete
**Details :** `.claude/sessions/SESSION_024_security_audit.md`

**Session 23: ZFS Wizard Fix + Documentation Cleanup** - Complete
**Details :** `.claude/sessions/SESSION_023_docs_cleanup.md`

---

## Bugs connus (non corriges)
- Permission denied sur manifests marc (shares marc:marc, Anemone tourne en franck)
- ZFS wizard : retour arriere ne re-affiche pas pool name/mountpoint (workaround : redemarrer Anemone)

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 26 | Retest Securite FR2 | 2026-02-15 | Complete |
| 25 | Corrections Securite Prioritaires | 2026-02-15 | Complete |
| 24 | Audit de Securite | 2026-02-15 | Complete |
| 23 | ZFS Wizard Fix + Documentation Cleanup | 2026-02-15 | Complete |
| 22 | !BADKEY Fix + Bugfixes + Release v0.20.0-beta | 2026-02-15 | Complete |
| 21 | OnlyOffice Auto-Config + Bugfixes | 2026-02-15 | Complete |
| 20 | OnlyOffice Integration | 2026-02-14 | Complete |
| 19 | Web File Browser | 2026-02-14 | Complete |
| 18 | Dashboard Last Backup Fix + Recent Tab | 2026-02-12 | Complete |
| 17 | Rclone Crypt Fix + !BADKEY Logs | 2026-02-11 | Complete |
| 16 | SSH Key Bugfix | 2026-02-11 | Complete |

---

## Ameliorations futures (non urgentes)

| # | Description | Effort | Impact |
|---|-------------|--------|--------|
| A10 full | Retirer `unsafe-inline` du CSP : externaliser JS inline (32 fichiers, 140 onclick) dans fichiers .js + data-attributes | ~2-3 jours | Defense en profondeur (aucun XSS trouve, risque theorique) |

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
