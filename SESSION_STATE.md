# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.20.0-beta
**Last Updated:** 2026-02-15

---

## Current Session

**Session 25: Corrections Securite Prioritaires** - En cours (vagues 1-5 terminees, audit complet)

**Détails :** `.claude/sessions/SESSION_025_security_fixes.md`
**Rapport audit :** `SECURITY_AUDIT_2026-02-15.md` (racine, NE PAS committer)

---

## Session 25: Corrections Securite Prioritaires

**Date:** 2026-02-15
**Objective:** Corriger les vulns identifiees en session 24
**Status:** En cours (vagues 1+2+3+4+5 done, audit complet)

### Corrections appliquees (vague 1)
| # | Correction | Statut |
|---|-----------|--------|
| 1 | Go 1.22.2 → 1.26.0 + deps a jour (24 vulns stdlib corrigees) | DONE |
| 2 | Rate limiting /login (5 tentatives/15min, lockout 15min) | DONE + teste FR2 |
| 3 | Cookie Secure=true (activation_key, setup_key) | DONE |
| 4 | Restriction methodes HTTP /login (GET/POST only, 405 sinon) | DONE + teste FR2 |
| 5 | Protection CSRF formulaires publics (double-submit cookie) | DONE + teste FR2 |
| 6 | Deploy + test complet sur FR2 | DONE |

### Corrections appliquees (vague 2)
| # | Correction | Statut |
|---|-----------|--------|
| 7 | A3 : Timing side-channel — DummyCheckPassword bcrypt constant-time | DONE + teste FR2 |
| 8 | G3 : HTTP Timeouts SlowLoris — http.Server avec timeouts | DONE + teste FR2 |
| 9 | A5 : Race condition tokens — MarkAsUsed atomique + avant activation/reset | DONE |
| 10 | Bug CSRF middleware — token passe via context au premier GET | DONE + teste FR2 |

### Corrections appliquees (vague 3)
| # | Correction | Statut |
|---|-----------|--------|
| 11 | A4 : Rate limiting par username en plus de l'IP (anti brute-force distribue) | DONE + teste FR2 |
| 12 | A7 : Validation IP en session — mismatch = session invalidee | DONE + teste FR2 |
| 13 | Bouton admin "Debloquer" compte dans /admin/users (badge Verrouille) | DONE + teste FR2 |

### Corrections appliquees (vague 4)
| # | Correction | Statut |
|---|-----------|--------|
| 14 | Page /admin/security : IPs bloquees + comptes verrouilles + deblocage | DONE + teste FR2 |
| A10 | CSP : retrait unsafe-eval, ajout object-src 'none' + base-uri 'self' | DONE + deploy FR2 |
| A11 | RememberMe 30j → 14j | DONE + deploy FR2 |
| I1 | SSRF : ValidatePeerAddress() bloque loopback/link-local/metadata | DONE + deploy FR2 |
| I2 | Injection rclone : quoteValue() sur SFTP host/user/key_file | DONE + deploy FR2 |
| G5 | Bombe tar : limite 10Go/fichier + 50Go total + io.LimitReader | DONE + deploy FR2 |

### Corrections appliquees (vague 5)
| # | Correction | Statut |
|---|-----------|--------|
| A6 | X-Forwarded-For spoofable : suppression getClientIP(), utilise clientIP() (RemoteAddr only) | DONE |
| I4 | CSP Host injection : isValidHostname() valide browserHost avant injection dans CSP | DONE |
| Phase 8 | Audit IDOR : 10 endpoints testes, 5 sync API acceptes (design P2P + chiffrement E2E) | DONE |
| Phase 9 | Audit XSS : 7 vecteurs testes, aucune vuln critique (html/template auto-escaping) | DONE |
| Phase 10 | Rapport final complete dans SECURITY_AUDIT_2026-02-15.md | DONE |

### Findings acceptes (pas de correction)
| # | Finding | Severite | Raison |
|---|---------|----------|--------|
| G2 | InsecureSkipVerify: true | HIGH | Necessaire pour certs auto-signes P2P |
| A9 | Complexite mdp (min 8 chars) | MEDIUM | Choix utilisateur |
| A12 | Token admin TTL 5 min | MEDIUM | Acceptable |
| A14 | Timeout inactivite session | LOW | Mitigue par IP binding + 14j max |
| ID1-5 | Acces cross-user sync API | MEDIUM | Design P2P, chiffrement E2E |
| G4 | MD5 integrite fichiers | MEDIUM | Content-addressable, pas securite |

### Resume technique
- **Go** : 1.26.0, go-sqlite3 1.14.34, x/crypto 0.48.0, x/sys 0.41.0
- **CSRF** : double-submit cookie avec context (fix premier GET)
- **Rate limiting** : 5 tentatives / 15 min par IP ET par username, lockout 15 min
- **Session IP binding** : mismatch IP = session invalidee + log
- **Admin unlock** : bouton debloquer compte dans /admin/users + badge Verrouille
- **Cookies** : tous Secure=true + SameSite=Strict
- **HTTP timeouts** : ReadHeader=10s, Read=30s, Write=60s, Idle=120s
- **Tokens** : MarkAsUsed atomique `(bool, error)`, consomme avant action
- **CSP** : unsafe-eval retire, object-src 'none', base-uri 'self' (unsafe-inline garde : 180+ handlers inline)
- **RememberMe** : 14 jours (etait 30j)
- **SSRF** : ValidatePeerAddress() dans peers.go, appelee au create/update peer
- **Rclone** : quoteValue() sur SFTP host/user/key_file dans buildSFTPRemote
- **Tar bomb** : maxFileSize=10Go, maxTotalSize=50Go, io.LimitReader dans ExtractTarGz
- **Page /admin/security** : handler + template + routes + i18n FR/EN + lien sidebar
- **IP detection** : clientIP() utilise seulement RemoteAddr (pas de proxy headers spoofables)
- **CSP Host** : isValidHostname() valide le Host header avant usage dans CSP OnlyOffice
- **IDOR** : Endpoints protege-session OK, sync API accepte (design P2P + chiffrement E2E)
- **XSS** : Aucune vuln trouvee (html/template auto-escaping)
- **Build OK, tests OK, audit securite COMPLET**

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
| 25 | Corrections Securite Prioritaires | 2026-02-15 | En cours |
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

## Next Steps

1. **Audit securite COMPLET** — toutes les corrections deployees sur FR2

## Ameliorations futures (non urgentes)

| # | Description | Effort | Impact |
|---|-------------|--------|--------|
| A10 full | Retirer `unsafe-inline` du CSP : externaliser JS inline (32 fichiers, 140 onclick) dans fichiers .js + data-attributes | ~2-3 jours | Defense en profondeur (aucun XSS trouve, risque theorique) |

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
