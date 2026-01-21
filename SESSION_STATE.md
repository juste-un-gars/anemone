# Anemone - Session State

**Current Version:** v0.9.23-beta
**Last Updated:** 2026-01-21

---

## Current Session

**Session 67** - Tests VM & Bug Fixes Setup Wizard
- **Status:** In Progress
- **Date:** 2026-01-20 to 2026-01-21
- **Details:** [SESSION_067_vm_tests.md](.claude/sessions/SESSION_067_vm_tests.md)

### Summary
Tests du setup wizard sur VM et corrections de bugs chemins personnalisés.

### Completed (2026-01-21)
- [x] Fix boucle HTTP/2 sur page "Configuration terminée" (suppression polling JS)
- [x] Fix Samba non démarré après installation (ajout `systemctl enable --now` dans install.sh)
- [x] Test installation propre sur VM Ubuntu (FR1) - OK
- [x] Fix validation chemin custom data directory (vérifiait parent au lieu du chemin cible)
- [x] Fix incoming directory séparé ignoré par le backend (SeparateIncoming non traité)
- [x] Utilisation de sudo pour création de répertoires (cohérent avec le reste du code)
- [x] Message d'aide clair si sudo échoue (commandes manuelles à exécuter)
- [x] Pré-remplissage du nom admin avec "admin" dans le wizard
- [x] Fix SetupIncomingDirectory vérifie si répertoire existe avant sudo mkdir
- [x] Fix FinalizeSetup utilise sudo pour créer le répertoire db (était os.MkdirAll)
- [x] Fix ownership des répertoires créés avec sudo (chown vers user courant)
- [x] Ajout EnvironmentFile au service systemd pour lire anemone.env (ANEMONE_INCOMING_DIR)
- [x] Ajout champ "Nom du serveur" au setup wizard (étape Admin)
- [x] Fix chown toujours appliqué même si répertoire existe déjà
- [x] Test synchro P2P avec incoming séparé (FR2 -> FR1) - OK
- [x] Fix API restauration utilisait DataDir au lieu de IncomingDir (4 fonctions corrigées)
- [x] Mise à jour CLAUDE.md avec nouveau modèle v2.0 (philosophie incrémentale, audit sécurité)
- [x] Fix création répertoires sans sudo si parent accessible (évite demandes sudo répétées)
- [x] Ajout chmod aux instructions manuelles de création répertoires
- [x] Déplacement env file vers /etc/anemone/anemone.env (emplacement fixe)
- [x] Création /etc/anemone pendant installation (install.sh)
- [x] Mise à jour sudoers quand chemin custom utilisé (remplace /srv/anemone par chemin custom)
- [x] Pré-remplissage nom serveur avec hostname dans wizard
- [x] Fix timing : mise à jour sudoers AVANT création utilisateur admin
- [x] Fix lecture sudoers avec sudo cat (permissions 440)
- [x] Fix sélection disques ZFS - checkbox click ne fonctionnait pas (stopPropagation)
- [x] Options RAID dynamiques selon nombre de disques sélectionnés
- [x] Popup confirmation rouge avant création pool ZFS (avertissement effacement disques)
- [x] Ajout flag Force à création pool ZFS (user a confirmé)
- [x] Affichage de TOUTES les commandes sudo à exécuter si échec (pas une par une)
- [x] Fix mapping "single" → "stripe" pour vdev type ZFS
- [x] Détection pool existant au retry (évite erreur "disk in use")
- [x] UX : Séparation sélection disques et configuration RAID en 2 sous-étapes
- [x] Suppression sudo chown inutile quand répertoire existe déjà

### Remaining
- [ ] Test complet sur VM Fedora
- [ ] Test ZFS new pool (en cours - à valider après retry)
- [ ] Test ZFS existing pool
- [x] Vérifier flux restauration (accès backup pair + téléchargement fichier OK)

### Commits de cette session
- `6b8af6d` fix: Use sudo to create database directory in FinalizeSetup
- `c731c17` fix: Set ownership of created directories to current user
- `381f0a9` fix: Add EnvironmentFile to systemd service and server name to setup wizard
- `4dd7ae1` fix: Always set ownership on directories even if they already exist
- `843d0b7` fix: Try creating directories without sudo first
- `f44bb9d` fix: Add chmod to manual directory creation instructions
- `dae1764` fix: Use /etc/anemone/anemone.env for service configuration
- `a8c2417` fix: Create /etc/anemone during installation
- `3b16e6d` fix: Update sudoers paths when custom data directory is used
- `2bd5c53` feat: Pre-fill server name with hostname in setup wizard
- `2c48198` fix: Update sudoers BEFORE creating admin user
- `534b4e7` fix: Use sudo to read sudoers file (has 440 permissions)
- `5078675` fix: Fix disk selection and improve ZFS RAID options UX
- `681050b` feat: Add ZFS disk erasure confirmation popup
- `eafd4b8` fix: Show all required commands at once when ZFS directory setup fails
- `c850172` fix: ZFS pool creation - fix vdev type and ownership handling
- `91391a0` fix: Detect existing ZFS pool on retry and skip creation
- `e265f69` feat: Separate ZFS disk selection and RAID configuration into two steps
- `7cb2ffb` fix: Don't try to set ownership via sudo after user runs manual commands
- `be053d7` fix: Remove unnecessary sudo chown when directory already exists

---

## Previous Session

**Session 66** - Tests d'intégration Setup Wizard
- **Status:** Completed
- **Date:** 2026-01-20
- **Details:** [SESSION_066_integration_tests.md](.claude/sessions/SESSION_066_integration_tests.md)

### Summary
Corrigé bug `go vet` : mutex copié dans SetupState. Créé SetupStateView pour l'usage externe.

---

## Recent Sessions

| # | Name | Date | Status | Details |
|---|------|------|--------|---------|
| 67 | Tests VM & Bug Fixes Setup Wizard | 2026-01-21 | In Progress | [Link](.claude/sessions/SESSION_067_vm_tests.md) |
| 66 | Tests d'intégration Setup Wizard | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_066_integration_tests.md) |
| 65 | Mode Restauration Serveur | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_065_restore_mode.md) |
| 64 | Nouveau install.sh simplifié | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_064_install_script.md) |
| 63 | Mode Setup - Frontend | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_063_setup_frontend.md) |
| 62 | Mode Setup - Backend | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_062_setup_backend.md) |
| 61 | Refactoring permissions | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_061_permissions_refactoring.md) |
| 60 | Refactoring chemins configurables | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_060_chemins_configurables.md) |
| 59 | Corrections urgentes (pré-refactoring) | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_059_corrections_urgentes.md) |
| 58 | Storage Bug Fixes & Mountpoint | 2026-01-20 | Completed | [Link](.claude/sessions/SESSION_058_storage_bug_fixes.md) |

---

## Session Archives

Older sessions are archived in `.claude/sessions/`:

| Sessions | Description | File |
|----------|-------------|------|
| 55-57 | Storage Management | Archive |
| 37-39 | Security Audit & Fixes | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_SESSIONS_37_39.md) |
| 31-34 | Update System | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_31_32_33_34.md) |
| 27-30 | Restore Interface | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_SESSIONS_27_30.md) |
| 26 | Internationalization FR/EN | [Link](.claude/sessions/SESSIONS_ARCHIVE.md) |
| 20-24 | P2P Sync & Scheduler | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_SESSIONS_20_24.md) |
| 17-19 | Trash & Quotas | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md) |
| 12-16 | SMB Automation | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_SESSIONS_12_16.md) |
| 8-11 | P2P Foundation | [Link](.claude/sessions/SESSION_STATE_ARCHIVE_SESSIONS_8_11.md) |
| 1-7 | Initial Setup & Auth | [Link](.claude/sessions/SESSION_STATE_ARCHIVE.md) |

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[.claude/REFERENCE.md](.claude/REFERENCE.md)** - Quick reference
- **[README.md](README.md)** - Installation guide
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

---

## Next Steps

**Session en cours : Session 67** - Tests VM & Bug Fixes Setup Wizard

Objectif : Continuer les tests d'installation sur différentes configurations.

**Sessions planifiées :**
- Session 68 : Mode Import Pool ZFS (récupération après réinstallation)
- Session 69 : Documentation mise à jour

Commencer par `"continue"` ou `"session 67"`.
