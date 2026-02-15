# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.20.0-beta
**Last Updated:** 2026-02-15

---

## Current Session

**Session 25: Corrections Securite Prioritaires** - Paused

**Détails :** `.claude/sessions/SESSION_025_security_fixes.md`
**Rapport audit :** `SECURITY_AUDIT_2026-02-15.md` (racine, NE PAS committer)

---

## Session 25: Corrections Securite Prioritaires

**Date:** 2026-02-15
**Objective:** Corriger les vulns identifiees en session 24
**Status:** Paused (corrections prioritaires faites, audit a finir plus tard)

### Corrections appliquees
| # | Correction | Statut |
|---|-----------|--------|
| 1 | Go 1.22.2 → 1.26.0 + deps a jour (24 vulns stdlib corrigees) | DONE |
| 2 | Rate limiting /login (5 tentatives/15min, lockout 15min) | DONE + teste FR2 |
| 3 | Cookie Secure=true (activation_key, setup_key) | DONE |
| 4 | Restriction methodes HTTP /login (GET/POST only, 405 sinon) | DONE + teste FR2 |
| 5 | Protection CSRF formulaires publics (double-submit cookie) | DONE + teste FR2 |
| 6 | Deploy + test complet sur FR2 | DONE |

### Bug resolu
- Rate limiting + CSRF + restriction methodes ne fonctionnaient pas sur FR2
- **Cause** : service unit `ExecStart=/home/franck/anemone/anemone`, deploiements precedents copiaient vers `/usr/local/bin/anemone` (mauvais chemin)
- **Fix** : deployer vers `/home/franck/anemone/anemone` sur FR2

### Reste a faire (audit securite)
| # | Finding | Severite | Statut |
|---|---------|----------|--------|
| A3 | Enumeration utilisateurs (timing side-channel) | CRITICAL | NON CORRIGE |
| A4 | Verrouillage par compte (pas seulement par IP) | HIGH | PARTIEL |
| A5 | Race condition tokens activation | HIGH | NON CORRIGE |
| A7 | Validation IP en session | HIGH | NON CORRIGE |
| G2 | InsecureSkipVerify: true (14 instances) | HIGH | NON CORRIGE (attendu: certs auto-signes P2P) |
| G3 | Timeouts HTTP (SlowLoris) | HIGH | NON CORRIGE |
| A9 | Complexite mdp (min 8 chars seulement) | MEDIUM | NON CORRIGE |
| A10 | CSP trop permissif (unsafe-inline/eval) | MEDIUM | NON CORRIGE |
| A11 | RememberMe 30j trop long | MEDIUM | NON CORRIGE |
| I1 | SSRF via adresse peer | MEDIUM | NON CORRIGE |
| I2 | Injection credentials rclone | MEDIUM | NON CORRIGE |
| G5 | Bombe decompression tar | MEDIUM | NON CORRIGE |
| Phase 8 | Tests authentifies (IDOR, privilege escalation) | - | A FAIRE |
| Phase 9 | Tests XSS | - | A FAIRE |
| Phase 10 | Rapport final + recommandations | - | A FAIRE |

### Resume technique
- **Go** : 1.26.0, go-sqlite3 1.14.34, x/crypto 0.48.0, x/sys 0.41.0
- **CSRF** : double-submit cookie sur login, setup, activate, reset-password
- **Rate limiting** : 5 tentatives / 15 min par IP, lockout 15 min
- **Cookies** : tous Secure=true + SameSite=Strict
- **Build OK, tests OK, deploy FR2 OK**

---

## Previous Sessions

**Session 24: Audit de Securite** - Complete

**Détails :** `.claude/sessions/SESSION_024_security_audit.md`

**Session 23: ZFS Wizard Fix + Documentation Cleanup** - Complete

**Détails :** `.claude/sessions/SESSION_023_docs_cleanup.md`

---

## Bugs connus (non corrigés)
- Permission denied sur manifests marc (shares marc:marc, Anemone tourne en franck)
- ZFS wizard : retour arrière ne ré-affiche pas pool name/mountpoint (workaround : redémarrer Anemone)

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 25 | Corrections Securite Prioritaires | 2026-02-15 | Paused |
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

## Next Steps (prochaine session)

1. **Finir audit securite** — phases 8 (IDOR/privesc), 9 (XSS), 10 (rapport final)
2. **Corriger findings HIGH restants** — G3 (timeouts HTTP), A3 (timing), A5 (race condition tokens)
3. **Evaluer findings MEDIUM** — complexite mdp, CSP, RememberMe, SSRF

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
