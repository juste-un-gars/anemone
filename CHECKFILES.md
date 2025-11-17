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
| `internal/activation/tokens.go` | 🔄 | - | Gestion tokens activation utilisateurs |

### Authentification
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/auth/middleware.go` | 🔄 | - | Middleware auth HTTP |
| `internal/auth/session.go` | 🔄 | - | Gestion sessions utilisateurs |

### Backup
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/backup/backup.go` | 🔄 | - | Système de backup utilisateur |
| `internal/bulkrestore/bulkrestore.go` | 🔄 | - | Restauration bulk (Session 18) |
| `internal/serverbackup/serverbackup.go` | 🔄 | - | Backups serveur (Session 15) |

### Configuration
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/config/config.go` | 🔄 | - | Configuration application |

### Crypto
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/crypto/crypto.go` | 🔄 | - | Chiffrement AES-256-GCM |

### Base de données
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/database/database.go` | 🔄 | - | Connexion SQLite |
| `internal/database/migrations.go` | 🔄 | - | Migrations schéma DB |

### i18n
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/i18n/i18n.go` | 🔄 | - | Traductions FR/EN (285 clés) |

### Incoming
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/incoming/incoming.go` | 🔄 | - | Gestion backups entrants |

### Peers
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/peers/peers.go` | 🔄 | - | Gestion serveurs pairs P2P |

### Quotas
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/quota/enforcement.go` | 🔄 | - | Enforcement quotas Btrfs |
| `internal/quota/quota.go` | 🔄 | - | Calcul et gestion quotas |

### Reset
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/reset/reset.go` | 🔄 | - | Réinitialisation mot de passe |

### Restore
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/restore/restore.go` | 🔄 | - | Restauration fichiers utilisateur |

### Scheduler
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/scheduler/scheduler.go` | 🔄 | - | Planification syncs automatiques |

### Shares
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/shares/shares.go` | 🔄 | - | Gestion partages SMB |

### SMB
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/smb/smb.go` | 🔄 | - | Configuration Samba |

### Sync
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/sync/manifest.go` | 🔄 | - | Manifests de synchronisation |
| `internal/sync/manifest_test.go` | 🔄 | - | Tests unitaires manifests |
| `internal/sync/sync.go` | 🔄 | - | Synchronisation P2P chiffrée |
| `internal/syncauth/syncauth.go` | 🔄 | - | Authentification P2P |
| `internal/syncconfig/syncconfig.go` | ✅ | 2025-11-17 | **OK** - Utilisé par router.go (ligne 3058) pour la configuration de sync automatique. Structure `SyncConfig` utilisée dans templates admin_sync.html. |

### TLS
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/tls/autocert.go` | 🔄 | - | Certificats auto-signés HTTPS |

### Trash
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/trash/trash.go` | 🔄 | - | Corbeille utilisateur |

### Users
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/users/users.go` | 🔄 | - | Gestion utilisateurs |

### Web
| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `internal/web/router.go` | 🔄 | - | Routes HTTP + handlers (4500+ lignes) |

---

## 🌐 Templates Web (web/templates/)

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `activate.html` | 🔄 | - | Activation compte utilisateur |
| `activate_success.html` | 🔄 | - | Succès activation |
| `admin_backup_export.html` | 🔄 | - | Export config serveur (obsolète?) |
| `admin_backup.html` | 🔄 | - | Liste backups serveur |
| `admin_incoming.html` | 🔄 | - | Backups entrants |
| `admin_peers_add.html` | 🔄 | - | Ajout pair |
| `admin_peers_edit.html` | 🔄 | - | Édition pair |
| `admin_peers.html` | 🔄 | - | Liste pairs |
| `admin_restore_users.html` | 🔄 | - | Restauration utilisateurs (Session 18) |
| `admin_settings.html` | 🔄 | - | Paramètres admin |
| `admin_shares.html` | 🔄 | - | Gestion partages |
| `admin_sync.html` | 🔄 | - | Forcer sync manuelle (obsolète?) |
| `admin_users_add.html` | 🔄 | - | Ajout utilisateur |
| `admin_users.html` | 🔄 | - | Liste utilisateurs |
| `admin_users_quota.html` | 🔄 | - | Édition quotas utilisateur |
| `admin_users_reset_token.html` | 🔄 | - | Token reset mdp |
| `admin_users_token.html` | 🔄 | - | Token activation |
| `base.html` | 🗑️ | 2025-11-17 | **MOVED** → `_audit_temp/web/templates/base.html` - Template de base non utilisé. Défini un layout mais jamais référencé par aucun autre template. Vestige de l'ancienne architecture. |
| `dashboard_admin.html` | 🔄 | - | Dashboard admin |
| `dashboard_user.html` | 🔄 | - | Dashboard utilisateur |
| `login.html` | 🔄 | - | Page login |
| `reset_password.html` | 🔄 | - | Réinitialisation mdp |
| `restore.html` | 🔄 | - | Restauration fichiers utilisateur |
| `restore_warning.html` | 🔄 | - | Avertissement serveur restauré |
| `settings.html` | 🔄 | - | Paramètres utilisateur |
| `setup.html` | 🔄 | - | Setup initial |
| `setup_success.html` | 🔄 | - | Succès setup |
| `trash.html` | 🔄 | - | Corbeille utilisateur |

---

## 🔧 Scripts (scripts/ et root)

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `dfree-wrapper.sh` | 🔄 | - | Wrapper quotas Samba |
| `install.sh` | 🔄 | - | Installation automatisée |
| `restore_server.sh` | 🔄 | - | Restauration serveur (Session 16-17) |
| `scripts/configure-smb-reload.sh` | 🔄 | - | Rechargement config SMB |
| `scripts/README.md` | 🔄 | - | Documentation scripts |

---

## 📖 Documentation

| Fichier | Statut | Date | Notes |
|---------|--------|------|-------|
| `QUICKSTART.md` | 🔄 | - | Guide démarrage rapide |
| `README.md` | 🔄 | - | Documentation principale |
| `SESSION_STATE.md` | 🔄 | - | État du projet (19 sessions) |

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

### Fichiers analysés
- **Commandes CLI (cmd/)** : 9/9 ✅ COMPLÉTÉ
  - ✅ OK : 8 fichiers essentiels
  - 🗑️ MOVED : 1 fichier (test-manifest)

- **Packages internes (internal/)** : 1/40 (en cours)
  - ✅ OK : 1 package (syncconfig)

- **Templates web** : 1/28 (en cours)
  - 🗑️ MOVED : 1 template (base.html)

### En attente
- **Packages internes (internal/)** : 39/40 restants
- **Templates web** : 27/28 restants
- **Scripts** : 0/5
- **Documentation** : 0/3

### Fichiers suspects déplacés dans _audit_temp/
- **cmd/** : 1 programme de test (test-manifest)
- **binaries/** : 1 binaire (test-manifest)
- **web/templates/** : 1 template (base.html)

### Nettoyage important recommandé
- **_old/** : ⚠️ 78 MB, 2675 fichiers obsolètes (Python/Docker, aucune référence active)

---

## 🎯 Plan d'Action

1. **Phase 1** : Vérifier tous les fichiers actifs (cmd/ + internal/ + templates)
2. **Phase 2** : Identifier le code mort dans router.go (4500+ lignes)
3. **Phase 3** : Confirmer suppression du répertoire `_old/`
4. **Phase 4** : Nettoyer les imports inutilisés
5. **Phase 5** : Documenter les résultats

---

**Dernière mise à jour** : 2025-11-17
