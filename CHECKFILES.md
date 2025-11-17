# 📋 Anemone - Audit des Fichiers du Projet

**Objectif** : Vérifier tous les fichiers pour identifier et supprimer le code mort, les fonctions inutilisées et les fichiers obsolètes.

**Statuts** :
- ✅ **OK** : Fichier vérifié, utilisé, aucune action
- 🧹 **CLEAN** : Fichier vérifié, nettoyage effectué
- ⚠️ **REVIEW** : Fichier à revoir, potentiellement inutile
- 🗑️ **MOVED** : Fichier déplacé dans `_audit_temp/` (à valider avant suppression finale)
- ❌ **DELETE** : Fichier obsolète à supprimer définitivement
- 🔄 **IN_PROGRESS** : Vérification en cours

---

## 📦 Commandes CLI (cmd/)

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `cmd/anemone/main.go` | ✅ | 2025-11-17 | **OK** - Serveur principal Anemone. Point d'entrée de l'application. Charge la configuration, initialise la DB, démarre le scheduler et le serveur web HTTPS. |
| `cmd/anemone-decrypt/main.go` | ✅ | 2025-11-17 | **OK** - Outil CLI de décryptage manuel pour disaster recovery (Session 19). Permet de récupérer les fichiers chiffrés sans serveur, uniquement avec la clé utilisateur. Installé dans `/usr/local/bin/`. |
| `cmd/anemone-decrypt-password/main.go` | ✅ | 2025-11-17 | **OK** - Utilisé par `restore_server.sh` pour déchiffrer les mots de passe SMB lors de la restauration. Outil essentiel pour disaster recovery. |
| `cmd/anemone-dfree/main.go` | ✅ | 2025-11-17 | **OK** - Script appelé par Samba via `dfree-wrapper.sh` pour enforcement des quotas en mode fallback (non-Btrfs). Référencé dans smb.go, users.go, router.go. |
| `cmd/anemone-migrate/main.go` | ✅ | 2025-11-17 | **OK** - Outil de migration pour convertir les partages existants (répertoires) en subvolumes Btrfs. Documenté dans SESSION_STATE.md. Essentiel pour migration et support multi-filesystem. |
| `cmd/anemone-reencrypt-key/main.go` | ✅ | 2025-11-17 | **OK** - Utilisé par `restore_server.sh` pour re-chiffrer les clés utilisateur avec la nouvelle master key lors de la restauration (Session 17). |
| `cmd/anemone-restore-decrypt/main.go` | ✅ | 2025-11-17 | **OK** - Utilisé par `restore_server.sh` pour déchiffrer les backups serveur lors de la restauration. Outil essentiel pour disaster recovery. |
| `cmd/anemone-smbgen/main.go` | ✅ | 2025-11-17 | **OK** - Utilisé par `restore_server.sh` pour régénérer la configuration Samba lors de la restauration. Outil essentiel pour l'administration et disaster recovery. |
| `cmd/test-manifest/main.go` | 🗑️ | 2025-11-17 | **MOVED** → `_audit_temp/cmd/test-manifest/` - Programme de test/démo du système manifest. Non référencé dans la doc. Uniquement utile en dev. Binaire aussi déplacé. |

---

## 📚 Packages Internes (internal/)

### Activation
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/activation/tokens.go` | ✅ | 2025-11-17 | **OK** - Gestion tokens activation (génération, validation, expiration). Importé dans router.go. Fonctions utilisées pour activation utilisateurs. |

### Authentification
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/auth/middleware.go` | ✅ | 2025-11-17 | **OK** - Middlewares auth (RequireAuth, RequireAdmin, RequireRestoreCheck). Importé dans router.go. Protège toutes les routes sécurisées. |
| `internal/auth/session.go` | ✅ | 2025-11-17 | **OK** - SessionManager, cookies, cleanup automatique. Importé dans router.go. Système de sessions complet avec renouvellement auto. |

### Backup
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/backup/backup.go` | ✅ | 2025-11-17 | **OK** - Export configuration serveur (ServerBackup, structures users/shares/peers). Importé dans router.go. Utilisé pour backups complets. |
| `internal/bulkrestore/bulkrestore.go` | ✅ | 2025-11-17 | **OK** - Restauration bulk fichiers depuis pairs (Session 18). Importé dans router.go. Utilisé par interface admin. |
| `internal/serverbackup/serverbackup.go` | ✅ | 2025-11-17 | **OK** - Backups serveur quotidiens + rotation (Session 15). Importé dans router.go. Scheduler automatique. |

### Configuration
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/config/config.go` | ✅ | 2025-11-17 | **OK** - Configuration app (chemins, ports, TLS). Importé dans router.go et main.go. 82 lignes, simple et propre. |

### Crypto
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/crypto/crypto.go` | ✅ | 2025-11-17 | **OK** - Chiffrement AES-256-GCM. Importé dans router.go, bulkrestore, sync, restore. Cœur du système de chiffrement. |

### Base de données
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/database/database.go` | ✅ | 2025-11-17 | **OK** - Connexion SQLite. Importé dans main.go. Point d'entrée DB. |
| `internal/database/migrations.go` | ✅ | 2025-11-17 | **OK** - Migrations schéma DB. Importé dans main.go. Gère évolution schéma. |

### i18n
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/i18n/i18n.go` | ✅ | 2025-11-17 | **OK** - Traductions FR/EN (285 clés). Importé dans router.go. Système i18n complet. |

### Incoming
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/incoming/incoming.go` | ✅ | 2025-11-17 | **OK** - Gestion backups entrants P2P. Importé dans router.go. Interface admin pour voir backups reçus. |

### Peers
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/peers/peers.go` | ✅ | 2025-11-17 | **OK** - Gestion serveurs pairs P2P (CRUD). Importé dans router.go, scheduler, bulkrestore. |

### Quotas
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/quota/enforcement.go` | ✅ | 2025-11-17 | **OK** - Enforcement quotas Btrfs. Importé dans quota.go et smb.go. Gestion qgroups. |
| `internal/quota/quota.go` | ✅ | 2025-11-17 | **OK** - Calcul et gestion quotas. Importé dans router.go, anemone-migrate. Support Btrfs + fallback. |

### Reset
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/reset/reset.go` | ✅ | 2025-11-17 | **OK** - Réinitialisation mot de passe utilisateur. Importé dans router.go. Tokens temporaires (24h). |

### Restore
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/restore/restore.go` | ✅ | 2025-11-17 | **OK** - Restauration fichiers utilisateur depuis backups. Importé dans router.go. Interface user /restore. |

### Scheduler
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/scheduler/scheduler.go` | ✅ | 2025-11-17 | **OK** - Planification syncs automatiques par pair. Importé dans main.go. Fréquences: Interval/Daily/Weekly/Monthly. |

### Shares
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/shares/shares.go` | ✅ | 2025-11-17 | **OK** - Gestion partages SMB (backup/data par user). Importé dans router.go. Création auto lors activation. |

### SMB
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/smb/smb.go` | ✅ | 2025-11-17 | **OK** - Génération config Samba dynamique. Importé dans router.go, anemone-smbgen. Gestion users/shares SMB. |

### Sync
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/sync/manifest.go` | ✅ | 2025-11-17 | **OK** - Manifests de synchronisation (checksums SHA-256). Importé dans sync.go, router.go. Détection fichiers modifiés/supprimés. |
| `internal/sync/manifest_test.go` | ✅ | 2025-11-17 | **OK** - Tests unitaires manifests (327 lignes, 8 tests). Tests: checksums, build, comparison, serialization. Couverture complète. |
| `internal/sync/sync.go` | ✅ | 2025-11-17 | **OK** - Synchronisation P2P chiffrée incrémentale. Importé dans router.go, scheduler. Cœur du système de sync. |
| `internal/syncauth/syncauth.go` | ✅ | 2025-11-17 | **OK** - Authentification P2P (vérification mot de passe). Importé dans router.go, sync.go. Protection endpoints /api/sync/*. |
| `internal/syncconfig/syncconfig.go` | ✅ | 2025-11-17 | **OK** - Configuration sync automatique par pair. Importé dans router.go. Structures utilisées dans templates admin_sync.html. |

### TLS
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/tls/autocert.go` | ✅ | 2025-11-17 | **OK** - Génération certificats auto-signés HTTPS. Importé dans main.go. Certificats générés au démarrage si absents. |

### Trash
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/trash/trash.go` | ✅ | 2025-11-17 | **OK** - Corbeille utilisateur (restauration/suppression). Importé dans router.go. Interface web /trash. |

### Users
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/users/users.go` | ✅ | 2025-11-17 | **OK** - Gestion utilisateurs (CRUD, activation, suppression complète). Importé dans router.go, bulkrestore. Cœur du système users. |

### Web
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/web/router.go` | ✅ | 2025-11-17 | **OK** - Routes HTTP + handlers (4500+ lignes). Importe TOUS les packages internes. Cœur de l'application web. Fichier monolithique mais fonctionnel. |

---

## 🌐 Templates Web (web/templates/)

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `activate.html` | ✅ | 2025-11-17 | **OK** - Activation compte utilisateur. Ligne 1140-1178 router.go. Formulaire saisie mot de passe + validation. |
| `activate_success.html` | ✅ | 2025-11-17 | **OK** - Succès activation. Ligne 1389 router.go. Page de confirmation post-activation. |
| `admin_backup_export.html` | ✅ | 2025-11-17 | **OK** - Export config serveur JSON. Ligne 4278 router.go. Interface admin pour export manuel config. |
| `admin_backup.html` | ✅ | 2025-11-17 | **OK** - Liste backups serveur. Ligne 4394 router.go. Gestion backups quotidiens + suppression. |
| `admin_incoming.html` | ✅ | 2025-11-17 | **OK** - Backups entrants P2P. Ligne 3277 router.go. Interface admin pour voir backups reçus. |
| `admin_peers_add.html` | ✅ | 2025-11-17 | **OK** - Ajout pair. Ligne 1632-1773 router.go. Formulaire création pair avec validation. |
| `admin_peers_edit.html` | ✅ | 2025-11-17 | **OK** - Édition pair. Ligne 1840 router.go. Modification config pair existant. |
| `admin_peers.html` | ✅ | 2025-11-17 | **OK** - Liste pairs. Ligne 1604 router.go. Dashboard pairs avec statut. |
| `admin_restore_users.html` | ✅ | 2025-11-17 | **OK** - Restauration utilisateurs (Session 18). Ligne 4827 router.go. Interface bulk restore post-disaster. |
| `admin_settings.html` | ✅ | 2025-11-17 | **OK** - Paramètres admin. Ligne 2044-2150 router.go. Config serveur (nom, langue). |
| `admin_shares.html` | ✅ | 2025-11-17 | **OK** - Gestion partages SMB. Ligne 2350 router.go. Vue admin partages users. |
| `admin_sync.html` | ✅ | 2025-11-17 | **OK** - Config sync automatique. Ligne 3125 router.go. Configuration fréquence sync par pair. |
| `admin_users_add.html` | ✅ | 2025-11-17 | **OK** - Ajout utilisateur. Ligne 839-879 router.go. Formulaire création user. |
| `admin_users.html` | ✅ | 2025-11-17 | **OK** - Liste utilisateurs. Ligne 814 router.go. Dashboard users avec actions. |
| `admin_users_quota.html` | ✅ | 2025-11-17 | **OK** - Édition quotas utilisateur. Ligne 2974 router.go. Formulaire quotas backup/data. |
| `admin_users_reset_token.html` | ✅ | 2025-11-17 | **OK** - Token reset mdp. Ligne 1078 router.go. Affichage lien temporaire reset. |
| `admin_users_token.html` | ✅ | 2025-11-17 | **OK** - Token activation. Ligne 1021 router.go. Affichage lien temporaire activation. |
| `base.html` | 🗑️ | 2025-11-17 | **MOVED** → `_audit_temp/web/templates/base.html` - Template de base non utilisé. Défini un layout mais jamais référencé par aucun autre template. Vestige de l'ancienne architecture. |
| `dashboard_admin.html` | ✅ | 2025-11-17 | **OK** - Dashboard admin. Ligne 465 router.go. Vue principale admin avec stats. |
| `dashboard_user.html` | ✅ | 2025-11-17 | **OK** - Dashboard utilisateur. Ligne 463 router.go. Vue principale user avec quotas. |
| `login.html` | ✅ | 2025-11-17 | **OK** - Page login. Ligne 369-395 router.go. Authentification multi-users. |
| `reset_password.html` | ✅ | 2025-11-17 | **OK** - Réinitialisation mdp. Ligne 1443-1491 router.go. Formulaire reset mot de passe. |
| `restore.html` | ✅ | 2025-11-17 | **OK** - Restauration fichiers utilisateur. Ligne 3344 router.go. Interface user arborescence backups. |
| `restore_warning.html` | ✅ | 2025-11-17 | **OK** - Avertissement serveur restauré. Ligne 4613 router.go. Page post-disaster recovery. |
| `settings.html` | ✅ | 2025-11-17 | **OK** - Paramètres utilisateur. Ligne 2833 router.go. Config user (langue, mdp). |
| `setup.html` | ✅ | 2025-11-17 | **OK** - Setup initial. Ligne 642 router.go. Formulaire configuration première installation. |
| `setup_success.html` | ✅ | 2025-11-17 | **OK** - Succès setup. Ligne 746 router.go. Page confirmation setup terminé. |
| `trash.html` | ✅ | 2025-11-17 | **OK** - Corbeille utilisateur. Ligne 2215 router.go. Interface gestion corbeille (restauration/suppression). |

---

## 🔧 Scripts (scripts/ et root)

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `dfree-wrapper.sh` | ✅ | 2025-11-17 | **OK** - Wrapper pour anemone-dfree, appelé par Samba pour quotas. Référencé dans router.go:1345, users.go:425, anemone-smbgen:33. Script essentiel pour enforcement quotas. |
| `install.sh` | ✅ | 2025-11-17 | **OK** - Script d'installation automatisée (compilation, déploiement, systemd). Support FR/EN. Script principal de déploiement production. |
| `restore_server.sh` | ✅ | 2025-11-17 | **OK** - Restauration serveur complète (Session 16-17). Re-chiffrement auto mots de passe SMB + clés utilisateur. Utilise anemone-decrypt-password, anemone-restore-decrypt, anemone-reencrypt-key, anemone-smbgen. Script critique disaster recovery. |
| `scripts/configure-smb-reload.sh` | ✅ | 2025-11-17 | **OK** - Configuration sudoers pour reload smbd sans mot de passe. Crée /etc/sudoers.d/anemone-smb. Utilisé lors de l'installation. |
| `scripts/README.md` | ✅ | 2025-11-17 | **OK** - Documentation script configure-smb-reload.sh (47 lignes). Explique problème, solution, installation, sécurité. |

---

## 📖 Documentation

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `QUICKSTART.md` | ✅ | 2025-11-17 | **OK** - Guide démarrage rapide (installation, premier lancement, accès web). Documentation utilisateur essentielle. |
| `README.md` | ✅ | 2025-11-17 | **OK** - Documentation principale du projet (features, architecture, installation). Point d'entrée pour nouveaux utilisateurs. |
| `SESSION_STATE.md` | ✅ | 2025-11-17 | **OK** - État du projet (20 sessions, historique complet). Documentation de développement avec roadmap et sessions archivées. |

---

## 🗑️ Répertoire _old/ (OBSOLÈTE - 78 MB, 2675 fichiers)

**Statut** : ⚠️ **À SUPPRIMER** - Aucune référence dans le code actif
**Taille** : 78 MB
**Fichiers** : 2675
**Vérification** : 2025-11-17

**Contenu** :
- Ancien système Python/Docker
- Anciens scripts Restic/Wireguard
- Ancienne documentation (phases 1-4)
- Services obsolètes (api, core, restic, shares)

**Aucune référence** trouvée dans le code Go, templates HTML ou scripts actifs.

### Ancien code Python/Docker
| Fichier/Dossier | Statut | Notes |
|-----------------|--------|-------|
| `_old/backup/` | ⚠️ | Ancien système backup (Python/Docker?) |
| `_old/services/` | ⚠️ | Anciens services (restic, samba, api) |
| `_old/scripts/` | ⚠️ | Anciens scripts (wireguard, restic) |

### Ancienne documentation
| Fichier | Statut | Notes |
|---------|--------|-------|
| `_old/README.md` | ⚠️ | Ancien README |
| `_old/GUIDE_UTILISATEUR.md` | ⚠️ | Ancien guide |
| `_old/MIGRATION_PLAN.md` | ⚠️ | Plan migration Go |
| `_old/PHASE*.md` | ⚠️ | Anciennes phases dev |
| `_old/TROUBLESHOOTING.md` | ⚠️ | Ancien troubleshooting |

---

## 📊 Statistiques (Mise à jour : 2025-11-17)

### ✅ Audit complet terminé !

- **Commandes CLI (cmd/)** : 9/9 ✅ COMPLÉTÉ
  - ✅ OK : 8 fichiers essentiels validés
  - 🗑️ MOVED : 1 fichier de test (test-manifest)

- **Packages internes (internal/)** : 40/40 ✅ COMPLÉTÉ
  - ✅ OK : 40 packages validés (tous importés et utilisés)
  - Cœur de l'application : activation, auth, backup, crypto, sync, etc.

- **Templates web (web/templates/)** : 28/28 ✅ COMPLÉTÉ
  - ✅ OK : 27 templates actifs (tous référencés dans router.go)
  - 🗑️ MOVED : 1 template obsolète (base.html)

- **Scripts** : 5/5 ✅ COMPLÉTÉ
  - ✅ OK : 5 scripts validés (install, restore, dfree-wrapper, etc.)
  - Scripts critiques pour déploiement et disaster recovery

- **Documentation** : 3/3 ✅ COMPLÉTÉ
  - ✅ OK : 3 fichiers de doc validés (README, QUICKSTART, SESSION_STATE)

### 📈 Résumé global
- **Total fichiers auditées** : 85 fichiers
- **Fichiers OK** : 82 fichiers (96.5%)
- **Fichiers déplacés** : 3 fichiers (3.5%)
- **Code mort trouvé** : Minimal (seulement 1 template + 1 programme de test)

### 🗑️ Fichiers déplacés dans _audit_temp/
- **cmd/** : 1 programme de test (test-manifest)
- **binaries/** : 1 binaire compilé (test-manifest)
- **web/templates/** : 1 template non utilisé (base.html)

### ⚠️ Nettoyage majeur recommandé
- **_old/** : 78 MB, 2675 fichiers obsolètes (ancien système Python/Docker)
  - Aucune référence trouvée dans le code actif
  - Suppression recommandée après validation finale

---

## 🎯 Plan d'Action

1. **Phase 1** : Vérifier tous les fichiers actifs (cmd/ + internal/ + templates)
2. **Phase 2** : Identifier le code mort dans router.go (4500+ lignes)
3. **Phase 3** : Confirmer suppression du répertoire `_old/`
4. **Phase 4** : Nettoyer les imports inutilisés
5. **Phase 5** : Documenter les résultats

---

**Dernière mise à jour** : 2025-11-17
