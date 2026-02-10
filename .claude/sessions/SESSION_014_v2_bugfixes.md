# Session 14: v2 UI Bugfixes

## Meta
- **Date:** 2026-02-10
- **Goal:** Corriger tous les boutons/pages cassés dans l'interface v2 + release v0.15.1-beta
- **Status:** Complete ✅
- **Branch:** feature/v2-dashboard-admin

## Current Module
**Working on:** Session complete
**Progress:** All bugs fixed

## Bugs Fixed

### Bug 1: `/admin/users/add` → Internal Server Error ✅
- **Symptôme:** Clic sur "Nouvel utilisateur" → 500
- **Cause:** Template `v2_users_add.html` utilise `{{if .Error}}` mais le handler GET passe un struct sans champ `Error`
- **Fix:** Ajout `Error string` + lecture de `r.URL.Query().Get("error")` dans le data struct GET
- **Fichier:** `internal/web/handlers_admin_users.go`

### Bug 2: `/admin/usb-backup/add` → 405 Method Not Allowed ✅
- **Symptôme:** Clic sur "Ajouter" USB backup → erreur
- **Cause:** `handleAdminUSBBackupAdd` n'acceptait que POST, pas de GET handler pour afficher le formulaire
- **Fix:** Ajout GET handler avec liste des drives détectés + shares disponibles, nouveau template
- **Fichiers:** `internal/web/handlers_admin_usb.go`, `web/templates/v2/v2_usb_backup_add.html` (nouveau)

### Bug 3: `/shares/add` → 404 ✅
- **Symptôme:** Clic sur "Ajouter" dans la page Partages → redirige vers dashboard
- **Cause:** Route `/shares/add` n'existe pas (les partages sont créés automatiquement à l'activation utilisateur)
- **Fix:** Supprimé le bouton "Ajouter" de `v2_shares.html` (headerActions + empty state)
- **Fichier:** `web/templates/v2/v2_shares.html`

### Bug 4: P2P "Configurer" → 405 ✅
- **Symptôme:** Clic sur "Configurer" dans l'onglet P2P sync → erreur
- **Cause:** Lien `<a href="/admin/sync/config">` mais ce handler est POST-only
- **Fix:** Changé le lien vers `/admin/peers` (page de gestion des peers)
- **Fichier:** `web/templates/v2/v2_backups.html`

### Bug 5: `/restore` (user) → connexion coupée ✅
- **Symptôme:** Utilisateur clique sur "Restauration" → "Échec de la connexion sécurisée"
- **Cause:** JavaScript `result.replace(\`{{${key}}}\`, value)` — les `{{` sont interprétés par le Go template parser, `${key}` n'est pas du Go template valide → `template.Must` panique → goroutine meurt → connexion coupée
- **Fix:** Changé les placeholders de `{{key}}` vers `{key}` (simple accolades) dans le JS et les traductions i18n
- **Fichiers:** `web/templates/v2/v2_restore.html`, `internal/i18n/locales/{fr,en}.json`

## Release v0.15.1-beta
- Version bump dans `internal/updater/updater.go`
- CHANGELOG.md mis à jour
- Tag `v0.15.1-beta` créé et poussé
- Release GitHub : https://github.com/juste-un-gars/anemone/releases/tag/v0.15.1-beta
- Branche `feature/v2-dashboard-admin` mergée dans `main`

## Additional Changes
- `usb_backup.select_drive` i18n key (FR/EN)
- Redirections POST erreur USB backup → form d'ajout au lieu de redirection perdue

## Files Modified
- `internal/web/handlers_admin_users.go` — Error field in GET struct
- `internal/web/handlers_admin_usb.go` — GET handler, redirections
- `web/templates/v2/v2_usb_backup_add.html` — (nouveau)
- `web/templates/v2/v2_shares.html` — Supprimé bouton add
- `web/templates/v2/v2_backups.html` — Lien P2P configurer
- `web/templates/v2/v2_restore.html` — Fix template/JS conflict
- `internal/i18n/locales/fr.json` — select_drive + fix placeholders
- `internal/i18n/locales/en.json` — select_drive + fix placeholders
- `internal/updater/updater.go` — Version 0.15.1-beta
- `CHANGELOG.md` — Entry v0.15.1-beta
- `SESSION_STATE.md` — Updated

## Technical Decisions
- **Partages sans bouton Add:** Les partages sont auto-créés lors de l'activation utilisateur, pas de création manuelle
- **Placeholders i18n:** Format `{key}` (simple accolades) au lieu de `{{key}}` pour éviter conflit Go template
- **USB backup add:** Formulaire séparé de l'edit (plus simple, pas besoin de gérer les valeurs existantes du backup)

## Handoff Notes
- Rclone multi-provider (Session 13) reste à tester sur FR1 → FR2
- Étapes : configurer remote rclone sur FR1, ajouter destination Named Remote dans Anemone
- FR2 prêt (SSH installé, créer dossier réception `/srv/anemone-backups`)
