# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur

**Current Version:** v0.23.0-beta
**Last Updated:** 2026-02-18

---

## Current Session

**Session 30: Test Installation FR1 (depuis GitHub)** - En cours
**Details :** `.claude/sessions/SESSION_030_install_test_fr1.md`

### Progression

| # | Etape | Statut |
|---|-------|--------|
| 1 | Installation depuis GitHub | DONE |
| 2 | Service running | DONE |
| 3 | Setup wizard (storage default) | DONE |
| 4 | Setup wizard (admin creation) | DONE |
| 5 | Finalize + restart | DONE |
| 6 | Login admin | DONE |
| 7 | Creation utilisateurs (alice, bob) | DONE |
| 8 | Activation utilisateurs | DONE |
| 9 | Upload fichiers test | DONE |
| 10 | Verification Samba + permissions | DONE |
| 11 | Test isolation inter-utilisateurs | DONE |
| 12 | Test Samba SMB + isolation | DONE |
| 13 | Verification logs (0 ERROR) | DONE |
| 14 | Installation serveur pair FR2 | DONE |
| 15 | Setup FR2 + Config pair FR1↔FR2 | DONE |
| 16 | Test sync P2P bidirectionnel | DONE |

### Phase 1 - Resultats
- Cree `web/static/js/theme-init.js`, `tailwind-config.js`, `common.js`
- Modifie `v2_base.html` et `v2_base_user.html` : inline scripts → src=, onclick → data-action
- Ajoute `{{block "pageScripts" .}}{{end}}` dans les 2 bases
- Build OK

### Phase 2 - Resultats
- Cree `web/static/js/auth.js` (changeLanguage, passwordValidation, copyKey, downloadKey)
- Modifie `activate.html`, `activate_success.html`, `setup.html`, `setup_success.html`
- Remplace onchange/onclick → data-action, inline scripts → page-data + auth.js
- Build OK

### Phase 3 - Resultats
- Cree `web/static/js/users.js` (deleteUser, unlockUser)
- Cree `web/static/js/peers.js` (testPeer, deletePeer)
- Ajoute data-confirm + data-submit-disable handlers dans common.js
- Ajoute initSyncFrequencyToggle() dans common.js
- Modifie `v2_users.html`, `v2_peers.html`, `v2_peers_add.html`, `v2_peers_edit.html`, `v2_onlyoffice.html`
- Build OK

### Phase 4 - Resultats
- Cree `files.js`, `trash.js`, `backups.js`, `restore.js`, `wireguard.js`
- Cree `rclone_add.js`, `rclone_edit.js`, `usb_backup_add.js`, `usb_backup_edit.js`, `restore_warning.js`
- Modifie tous les templates v2 correspondants
- Build OK

### Phase 5 - Resultats
- Cree `editor.js` (v2_editor.html - OnlyOffice, onerror remplace par detection typeof)
- Cree `admin_backup.js` (download modal, delete, form validation)
- Cree `admin_sync.js` (toggleFixedHour)
- Cree `admin_export.js` (validateForm passphrase)
- Cree `admin_restore.js` (filterByPeer, restoreUser, restoreAll + backups en page-data JSON)
- Cree `admin_rclone.js` (copyPublicKey, generateKey, confirmRegenerateKey)
- Cree `admin_usb.js` (toggleShareSelection, ejectDisk)
- Modifie 7 templates : v2_editor, admin_backup, admin_sync, admin_incoming, admin_backup_export, admin_restore_users, admin_rclone, admin_usb_backup
- Ajoute common.js aux pages legacy pour data-confirm
- Build OK

### Phase 6 - Resultats
- Cree `storage.js` (633 lignes - tabs, SMART modal, password verification, pool/dataset/snapshot CRUD, disk format/mount/unmount/eject, 90+ traductions dans page-data)
- Cree `setup_wizard.js` (1011 lignes - wizard multi-etapes, mode/storage/incoming selection, ZFS config, admin form, finalize, restore flow complet, drag-and-drop)
- Modifie `v2_storage.html` (1103→629 lignes) : supprime 584 lignes JS inline, 30+ onclick → data-action
- Modifie `setup_wizard.html` (1607→742 lignes) : supprime 895 lignes JS inline, 25+ onclick/onchange → data-action
- `{{if .State.Finalized}}` deplace dans page-data JSON comme boolean
- `{{range .Pools}}{{.Name}},{{end}}` deplace dans page-data JSON comme poolNames
- Build OK

### Phase 2 (serveur pair) - Resultats

**Etape 14 - Installation FR2 (192.168.83.37)**
- install.sh mode 1 (new), Ubuntu 24.04
- Go 1.26, GCC, Samba, ZFS, rclone, Docker, WireGuard
- Groupe `anemone` cree, service actif

**Etape 15 - Setup FR2 + Pairs**
- Setup wizard : storage default, admin/Admin1234, server_name=FR2
- Users alice (Alice1234) et bob (Bob12345) crees et actives
- Pair FR2 ajoute sur FR1, pair FR1 ajoute sur FR2
- Test connectivite bidirectionnel : "Connexion reussie" des 2 cotes
- Cles de chiffrement differentes entre FR1 et FR2 (isolation crypto confirmee)

**Etape 16 - Sync P2P**
- FR1->FR2 : 6 fichiers chiffres (alice: Classeur.xlsx, qa.txt, Scenarios Thinprint.pptx / bob: debug_auth.go, gitlab.txt, SpaceMonger.exe)
- FR2->FR1 : 4 fichiers chiffres (alice: fix_hash.go, model_CLAUDE.md / bob: import_driver.ps1, test(4).md)
- Seuls les shares `backup_*` sont synchronises (data_* = local uniquement)
- 0 ERROR dans les logs des 2 serveurs

### Phase 7 - Resultats
- Corrige 6 pages manquantes : v2_users_token, v2_users_reset_token, v2_shares, v2_security, v2_system_update, v2_users_quota
- Ajoute `copyInput` handler generique dans common.js (reutilise par token + reset_token)
- Cree `shares.js` (deleteShare, syncShare avec event delegation)
- Cree `system_update.js` (checkForUpdates, confirmUpdate/installUpdate)
- Cree `quota.js` (calcul total quota auto)
- v2_security.html : onsubmit confirm → data-confirm
- Retire `'unsafe-inline'` de `script-src` dans CSP (`internal/web/router.go:533`)
- **0 inline handler, 0 `<script>` bloc inline** dans tous les templates
- Build OK

---

## Previous Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 29 | Externalisation JS Inline (CSP strict) | 2026-02-18 | Complete |
| 28 | Tests d'integration multi-serveurs | 2026-02-18 | Complete (95/95) |
| 27 | Groupe Anemone + Release v0.22.0-beta | 2026-02-17 | Complete |
| 26 | Retest Securite FR2 | 2026-02-15 | Complete (32/32) |
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

## Bugs connus (non corriges)
- (aucun)

---

Commencer par `"lire SESSION_STATE.md"` puis `"continue"`.
