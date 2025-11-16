# 🪸 Anemone - État du Projet

**Dernière session** : 2025-11-16 (Session 18 - Interface admin de restauration utilisateurs + Fix bulk restore)
**Prochaine session** : Tests interface utilisateur + Audit sécurité
**Status** : 🟢 COMPLÈTE - Restauration admin fonctionnelle à 100%

> **Note** : L'historique des sessions 1-7 a été archivé dans `SESSION_STATE_ARCHIVE.md`
> **Note** : Les détails techniques des sessions 8-11 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`
> **Note** : Les détails techniques des sessions 12-16 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md`

---

## 🎯 État actuel

### ✅ Fonctionnalités complètes et testées

1. **Configuration initiale (Setup)**
   - Choix langue (FR/EN)
   - Création premier admin
   - Génération clé de chiffrement

2. **Authentification & Sécurité**
   - Login/logout multi-utilisateurs
   - Sessions sécurisées
   - HTTPS avec certificat auto-signé
   - Réinitialisation mot de passe par admin

3. **Gestion utilisateurs**
   - Création utilisateurs par admin
   - Activation par lien temporaire (24h)
   - Création automatique user système + SMB
   - **Suppression complète** : Efface DB, fichiers disque, user SMB, user système
   - **Confirmation renforcée** : Double confirmation + saisie nom utilisateur
   - **Clé de chiffrement unique par utilisateur** : 32 bytes, générée à l'activation

4. **Partages SMB automatiques**
   - 2 partages par user : `backup_username` + `data_username`
   - Création auto lors activation
   - Permissions et ownership automatiques
   - Configuration SELinux automatique
   - **Privacy** : Chaque user ne voit que ses partages
   - **Corbeille intégrée** : VFS recycle module Samba

5. **Corbeille (Trash/Recycle Bin)**
   - Interception suppressions SMB via Samba VFS
   - Déplacement fichiers dans `.trash/%U/`
   - Interface web de gestion
   - Restauration fichiers
   - Suppression définitive
   - Vidage corbeille complet

6. **Gestion pairs P2P**
   - CRUD complet avec édition
   - Test connexion HTTPS avec authentification
   - Statuts (online/offline/error)
   - **Synchronisation manuelle** : Bouton sync par partage
   - **Synchronisation automatique** : Scheduler intégré avec fréquences personnalisables
   - **Chiffrement E2E** : AES-256-GCM par utilisateur
   - **Authentification P2P** : Protection endpoints par mot de passe

7. **Système de Quotas**
   - **Quotas Btrfs kernel** : Enforcement automatique au niveau filesystem
   - Subvolumes Btrfs par partage
   - Interface admin : Définition quotas backup + data
   - Dashboard user : Barres progression avec alertes (vert/jaune/orange/rouge)
   - Migration automatique : `anemone-migrate` pour convertir dirs existants
   - **Fallback mode** : ext4/XFS/ZFS fonctionnent sans enforcement

8. **Chiffrement End-to-End**
   - Clé unique 32 bytes par utilisateur
   - Chiffrement AES-256-GCM avec AEAD
   - Hiérarchie : Master key → User keys (chiffrées)
   - Backups P2P chiffrés automatiquement
   - Protection même si peer compromis

9. **Synchronisation incrémentale**
   - Système de manifest pour tracking fichiers
   - Upload fichier par fichier (type rclone)
   - Seulement les fichiers modifiés sont transférés
   - Suppression automatique fichiers obsolètes
   - Chaque fichier chiffré individuellement
   - Stockage : `/srv/anemone/backups/incoming/{user_id}_{share_name}/`

10. **Scheduler automatique**
    - Goroutine background vérifiant toutes les 1 minute
    - Configuration par pair (interval/daily/weekly/monthly)
    - Bouton "Forcer la synchronisation" pour trigger manuel
    - Logs détaillés dans la console serveur
    - Dashboard utilisateur affiche "Dernière sauvegarde"

11. **Authentification P2P par mot de passe**
    - **Mot de passe serveur** : Protège les endpoints `/api/sync/*` contre accès non autorisés
    - **Mot de passe pair** : Authentification auprès des serveurs distants
    - Middleware `syncAuthMiddleware` avec header `X-Sync-Password`
    - Interface admin `/admin/settings` pour configurer le mot de passe serveur
    - Champ mot de passe lors de l'ajout/édition de pairs
    - Hachage bcrypt côté serveur (stockage sécurisé)
    - Rétrocompatibilité : Sans mot de passe configuré = accès libre

12. **Gestion des backups entrants**
    - Vue `/admin/incoming` pour visualiser les pairs qui stockent des backups
    - Statistiques : nombre de pairs, fichiers, espace utilisé
    - Suppression de backups entrants
    - Carte dashboard pour accès rapide

13. **Édition de pairs**
    - Interface `/admin/peers/{id}/edit` pour modifier la configuration
    - Modification nom, adresse, port, mot de passe, statut, fréquence sync
    - Gestion intelligente du mot de passe (conserver/modifier/supprimer)
    - Test d'authentification intégré au bouton "Test"
    - Détection automatique des erreurs d'authentification (401/403)

14. **Installation automatisée**
    - Script `install.sh` zéro-touch
    - Configuration complète système
    - Support multi-distro (Fedora/RHEL/Debian)

15. **Restauration de fichiers avec interface web** (Session 12)
    - Liste des backups disponibles sur tous les pairs distants
    - Navigation dans l'arborescence des fichiers chiffrés
    - Déchiffrement automatique côté serveur d'origine
    - **Sélection multiple** : Checkboxes pour fichiers et dossiers
    - **Téléchargement ZIP** : Plusieurs fichiers/dossiers en un clic
    - **Expansion récursive** : Sélection d'un dossier inclut tous les sous-fichiers
    - Support des chemins avec espaces et caractères spéciaux

16. **Backups serveur automatiques** (Session 15)
    - Scheduler quotidien à 4h du matin
    - Rotation automatique (10 derniers backups)
    - Re-chiffrement à la volée pour téléchargement sécurisé
    - Interface admin `/admin/backup`

17. **Restauration complète du serveur** (Sessions 16-17)
    - Script `restore_server.sh` pour restauration complète
    - **Re-chiffrement automatique** des mots de passe SMB avec nouvelle master key
    - **Re-chiffrement automatique** des clés utilisateur avec nouvelle master key
    - Création automatique des utilisateurs système et SMB
    - Configuration automatique des partages
    - Flag `server_restored` pour afficher page d'avertissement

### 🚀 Déploiement

**DEV (localhost)** : ✅ Développement actif
**FR1 (192.168.83.16)** : ✅ Serveur source avec utilisateurs et fichiers
**FR2 (192.168.83.37)** : ✅ Serveur de backup (stockage pairs)
**FR3 (192.168.83.38)** : ✅ Serveur restauré (tests disaster recovery)

**Tests validés** :
- ✅ Accès SMB depuis Windows : OK
- ✅ Accès SMB depuis Android : OK
- ✅ Création/lecture/écriture fichiers : OK
- ✅ **Blocage quota dépassé** : OK
- ✅ Privacy SMB (chaque user voit uniquement ses partages) : OK
- ✅ Multi-utilisateurs : OK
- ✅ SELinux (Fedora) : OK
- ✅ **Synchronisation automatique** : OK
- ✅ **Synchronisation incrémentale** : OK (fichiers modifiés/supprimés détectés)
- ✅ **Dashboard "Dernière sauvegarde"** : OK
- ✅ **Authentification P2P** : OK (401/403/200 selon mot de passe)
- ✅ **Restauration fichiers depuis pairs** : OK (Session 12)
- ✅ **Téléchargement ZIP multiple** : OK (Session 12)
- ✅ **Backups serveur quotidiens** : OK (Session 15)
- ✅ **Restauration config serveur** : OK (Session 16-17)
- ✅ **Restauration mots de passe SMB** : OK (Session 16)
- ✅ **Re-chiffrement clés utilisateur** : OK (Session 17)

**Structure de production** :
- Code : `~/anemone/` (repo git, binaires)
- Données : `/srv/anemone/` (db, certs, shares, smb, backups)
- Base de données : `/srv/anemone/db/anemone.db`
- Binaires système : `/usr/local/bin/` (anemone, anemone-dfree, anemone-smbgen, anemone-migrate)
- Service : `systemd` (démarrage automatique)

### 📦 Liens utiles

- **GitHub** : https://github.com/juste-un-gars/anemone
- **Donation PayPal** : https://paypal.me/justeungars83

---

## 📋 Sessions archivées

- **Sessions 1-7** : Voir `SESSION_STATE_ARCHIVE.md`
- **Sessions 8-11** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`
- **Sessions 12-16** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md`

---

## 🔧 Session 13 - 10 Novembre 2025 - Fréquence de synchronisation par pair (avec Interval)

### 🎯 Objectif

Permettre de configurer une fréquence de synchronisation indépendante pour chaque pair, incluant une option "Interval" pour synchroniser toutes les X minutes ou heures.

### ✅ Architecture implémentée

**Avant** : Configuration globale dans `sync_config` → tous les pairs synchronisés en même temps
**Après** : Configuration individuelle par pair → chaque pair a sa propre fréquence et son propre timestamp de dernière sync

**Fréquences supportées** :
- **Interval** : Synchronisation à intervalle régulier (ex: toutes les 30 minutes, toutes les 2 heures)
- **Daily** : Synchronisation quotidienne à une heure fixe (ex: 23:00)
- **Weekly** : Synchronisation hebdomadaire un jour spécifique (ex: Samedi 23:00)
- **Monthly** : Synchronisation mensuelle un jour spécifique (ex: 1er du mois à 23:00)

**Statut** : 🟢 COMPLÈTE ET TESTÉE

---

## 🔧 Session 17 - 15 Novembre 2025 - Re-chiffrement des clés utilisateur lors de la restauration

**Date** : 2025-11-15
**Objectif** : Corriger le problème critique de restauration des fichiers après restauration serveur
**Priorité** : 🔴 CRITIQUE → 🟢 RÉSOLUE

### 🐛 Problème découvert

Lors des tests de restauration FR1 → FR3 avec backup sur FR2, la restauration automatique échouait avec :
```
Bulk restore failed: failed to decrypt user key:
failed to decrypt: cipher: message authentication failed
```

### 🔍 Analyse du problème

**Architecture du chiffrement** :
```
Master Key (unique par serveur)
    ↓ chiffre
User Encryption Key (unique par utilisateur)
    ↓ chiffre
Fichiers utilisateur (backup sur pairs distants)
```

**Problème** :
- FR1 génère une master key unique : `MK_FR1`
- `encryption_key_encrypted` est chiffré avec `MK_FR1`
- FR3 génère une NOUVELLE master key : `MK_FR3`
- Le script `restore_server.sh` restaure `encryption_key_encrypted` tel quel (chiffré avec `MK_FR1`)
- Quand FR3 essaie de restaurer les fichiers, impossible de déchiffrer la clé utilisateur

### ✅ Solution implémentée

**Principe** : Re-chiffrer `encryption_key_encrypted` avec la nouvelle master key lors de la restauration.

**Outil créé** : `cmd/anemone-reencrypt-key/main.go`
- Déchiffre la clé utilisateur avec l'ancienne master key
- Re-chiffre avec la nouvelle master key
- Retourne la clé re-chiffrée en base64

**Script modifié** : `restore_server.sh`
- Extrait l'ancienne master key du backup
- Génère une nouvelle master key pour le serveur restauré
- Re-chiffre `password_encrypted` ET `encryption_key_encrypted` pour chaque utilisateur
- Insère les valeurs re-chiffrées dans la base de données

### 🔨 Problèmes rencontrés et correctifs appliqués

#### Problèmes résolus (commits)
1. ✅ **Double encodage base64** (commit 4fb306d)
2. ✅ **Type de données dans export** (commit fbcf7b9)
3. ✅ **Lecture SQLite BLOB vs TEXT** (commit c09574d)
4. ✅ **Insertion TEXT au lieu de BLOB** (commit 2c93955)
5. ✅ **Format Manifest incompatible** (commit 7c48184)
6. ✅ **Share path hardcodé** (commit daaa39d)
7. ✅ **Convention de nommage shares** (commit 0335cdb)

### 📝 Commits

```
4fb306d - fix: Remove double base64 encoding in restore script
fbcf7b9 - fix: Change EncryptionKeyEncrypted type to string
c09574d - fix: Use sql.NullString to read encryption_key_encrypted as TEXT
2c93955 - fix: Insert encryption_key_encrypted as TEXT, not BLOB (Session 17)
7c48184 - fix: Fix manifest Files type mismatch (Session 17)
daaa39d - fix: Use database share path instead of hardcoded names (Session 17)
0335cdb - fix: Apply backup_{username} convention in list-user-backups API
```

**Statut** : 🟢 **COMPLÈTE - Tous les problèmes d'encodage et de manifest résolus**

---

## 🔧 Session 18 - 15-16 Novembre 2025 - Interface admin de restauration utilisateurs

**Date** : 2025-11-15 et 2025-11-16
**Objectif** : Créer une interface admin sécurisée pour restaurer les fichiers de tous les utilisateurs après disaster recovery
**Priorité** : 🔴 CRITIQUE → 🟢 COMPLÈTE

### 🎯 Contexte et Solution

**Problème initial** :
- Lors de la restauration serveur, le scheduler démarre automatiquement
- Le serveur restauré détecte "tous les fichiers supprimés" car les shares sont vides
- **Risque** : Envoi de commandes DELETE aux pairs → perte totale des backups

**Solution implémentée** :
1. **`restore_server.sh`** désactive automatiquement tous les pairs (`sync_enabled = 0`)
2. **Interface admin `/admin/restore-users`** pour restauration contrôlée
3. **Workflow sécurisé** : Restauration → Admin restaure fichiers → Réactivation pairs manuelle

### ✅ Problèmes résolus

**1. Erreurs 400 lors du téléchargement** (15 Nov)
- **Cause** : Le manifest utilise le chemin de fichier comme clé de map, mais `file.Path` était vide
- **Solution** : Utiliser `for filePath, file := range manifest.Files` au lieu de `for _, file`
- **Résultat** : 7 files, 280596 bytes, 0 errors ✅

**2. Ownership root:root sur fichiers restaurés** (15 Nov)
- **Cause** : Pas de changement d'ownership après création des fichiers
- **Solution** : Ajout fonction `setOwnership()` avec `os.Chown()`
- **Résultat** : Fichiers appartiennent à `test:test` ✅

**3. Interface web ne réagissait pas** (16 Nov)
- **Cause** : JavaScript invalide (`formData 2 _ 1` avec espaces)
- **Solution** : Réécriture `restoreAll()` avec tableau d'objets
- **Résultat** : Boutons cliquables, restauration fonctionne ✅

**4. Dossiers parents avec ownership root:root** (16 Nov)
- **Cause** : `os.MkdirAll()` appelé sans `setOwnership()` pour les dossiers parents
- **Solution** : Ajout `setOwnership(parentDir, user.Username)` après `MkdirAll()`
- **Résultat** : Suppression possible via SMB ✅

### 📝 Composants créés

- **Interface admin** : `/admin/restore-users` (liste tous les backups disponibles)
- **Handlers** : `handleAdminRestoreUsers()`, `handleAdminRestoreUsersRestore()`
- **Templates** : `admin_restore_users.html`, modification `restore_warning.html`
- **Script** : `restore_server.sh` désactive automatiquement les pairs
- **Corrections** : `bulkrestore.go` (clé map + ownership), `admin_restore_users.html` (JavaScript)

### 🧪 Tests validés

- ✅ **Workflow disaster recovery complet** : FR1 → FR2 → FR3 (restauration + fichiers)
- ✅ **Restauration API** : 7 files, 280596 bytes, 0 errors en ~0.3s
- ✅ **Ownership correct** : Tous fichiers/dossiers `test:test`
- ✅ **Interface web** : Boutons cliquables, JavaScript valide, aucune erreur console
- ✅ **SMB** : Suppression fichiers/dossiers possible
- ✅ **Synchronisation** : Nouveaux fichiers détectés et synchronisés (2 min)

### 📝 Commits

```
e13ab65 - fix: Fix JavaScript template and parent directory ownership in bulk restore (Session 18) [16 Nov]
c9a7d10 - fix: Fix bulk restore to use manifest map keys and set proper file ownership (Session 18) [16 Nov]
778fa32 - docs: Update SESSION_STATE.md with Session 18 completion details [16 Nov]
c869161 - feat: Add admin interface for user file restoration after disaster recovery (Session 18) [15 Nov]
```

**Détails des commits** :
1. **e13ab65** : Fix JavaScript + ownership dossiers parents
   - Réécriture `restoreAll()` avec tableau au lieu de variables dynamiques
   - Ajout `setOwnership()` pour dossiers parents créés par `MkdirAll()`
2. **c9a7d10** : Fix bulk restore avec clé map manifest
   - Utilisation clé map au lieu de `file.Path` vide
   - Ajout fonction `setOwnership()` pour fichiers/dossiers
3. **778fa32** : Documentation de la session 18
4. **c869161** : Interface admin de restauration (commit initial session 18)

**État session 18** : 🟢 **COMPLÈTE - Restauration admin fonctionnelle à 100%**

**Prochaine session** :
1. Tests complets de l'interface utilisateur (restauration depuis dashboard)
2. Audit de sécurité complet (priorité 1 roadmap)
3. Vérification d'intégrité des backups (priorité 2 roadmap)

---

## 📝 Prochaines étapes (Roadmap)

### 🎯 Priorité 1 - Court terme

**Session 18 : Interface admin de restauration utilisateurs** 🟢 COMPLÈTE
- ✅ Interface admin créée (`/admin/restore-users`)
- ✅ Fix bulk restore (utilisation clé map manifest)
- ✅ Fix ownership fichiers restaurés (test:test)
- ✅ Tests complets disaster recovery (7 files, 280596 bytes, 0 errors)

**Session 14 : Audit de sécurité complet** 🔒
- **Audit des permissions fichiers**
  - Vérifier permissions `/srv/anemone/` (600/700)
  - Vérifier ownership des fichiers sensibles
  - Vérifier permissions base de données
  - Vérifier permissions certificats TLS
- **Audit des clés de chiffrement**
  - Vérifier que la master key est uniquement en DB
  - Vérifier le chiffrement des clés utilisateurs
  - Vérifier l'absence de clés en clair sur le disque
  - Tester la rotation de clés
- **Audit des endpoints API**
  - Vérifier l'authentification sur tous les endpoints
  - Tester les tentatives d'accès non autorisées
  - Vérifier la protection CSRF
  - Tester les injections SQL
  - Vérifier la validation des inputs
  - Tester path traversal sur les endpoints de fichiers

### ⚙️ Priorité 2 - Améliorations

1. **Logs et audit trail** 📋
   - Table `audit_log` en base de données
   - Enregistrement actions importantes (user/peer CRUD, quotas, connexions)
   - Interface admin pour consulter les logs
   - Job de nettoyage automatique des anciens logs

2. **Vérification d'intégrité des backups** ✅
   - Commande `anemone-verify` pour vérification manuelle
   - Vérification checksums depuis manifests
   - Option vérification périodique en background
   - Alerte si corruption détectée

3. **Rate limiting anti-bruteforce** 🛡️
   - Protection sur `/login` et `/api/sync/*`
   - Bannissement temporaire après X tentatives échouées
   - Whitelist IP de confiance

4. **Statistiques détaillées de synchronisation** 📊
   - Graphiques d'utilisation (espace, fichiers, bande passante)
   - Historique des syncs sur 30 jours
   - Performance réseau par pair
   - Tableau de bord monitoring

### 🚀 Priorité 3 - Évolutions futures

1. **Guide utilisateur complet** 📚
   - Guide d'installation pas-à-pas avec captures d'écran
   - Guide d'utilisation pour chaque fonctionnalité
   - Exemples de configurations (topologies réseau)
   - FAQ et troubleshooting
   - Best practices sécurité et performance
   - Disponible en FR et EN

2. **Système de notifications** 📧
   - **Module Home Assistant** via webhooks
   - **Webhooks génériques** (Discord, Slack, custom)
   - **Email SMTP** (optionnel)
   - Événements notifiables : Sync réussie/échouée, quota 80%+, nouveau pair, auth échouée

3. **Multi-peer redundancy**
   - Stockage sur plusieurs pairs simultanément (2-of-3, 3-of-5)
   - Choix du niveau de redondance par partage
   - Reconstruction automatique en cas de perte d'un pair

### 📌 Notes

- **Bandwidth throttling** : Non prioritaire car les fréquences différenciées par pair permettent déjà de planifier les syncs hors heures de pointe.

- **Politique de rétention automatique** : Remplacée par le système de fréquence de synchronisation par pair, permettant des snapshots à différentes fréquences sans complexité supplémentaire.

---

**État global** : 🟡 INTERFACE ADMIN DE RESTAURATION EN COURS
**Prochaine étape** : Diagnostic et résolution problème restauration + permissions
