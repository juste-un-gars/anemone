# 🪸 Anemone - État du Projet

**Dernière session** : 2025-11-11 (Session 12 - Interface web de restauration avec sélection multiple)
**Status** : 🟢 RESTAURATION DISTANTE COMPLÈTE ET TESTÉE

> **Note** : L'historique des sessions 1-7 a été archivé dans `SESSION_STATE_ARCHIVE.md`
> **Note** : Les détails techniques des sessions 8-11 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`

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

9. **Synchronisation incrémentale** ✨ Session 8
   - Système de manifest pour tracking fichiers
   - Upload fichier par fichier (type rclone)
   - Seulement les fichiers modifiés sont transférés
   - Suppression automatique fichiers obsolètes
   - Chaque fichier chiffré individuellement
   - Stockage : `/srv/anemone/backups/incoming/{user_id}_{share_name}/`

10. **Scheduler automatique** ✨ Session 9
    - Goroutine background vérifiant toutes les 1 minute
    - Configuration par pair (interval/daily/weekly/monthly)
    - Bouton "Forcer la synchronisation" pour trigger manuel
    - Logs détaillés dans la console serveur
    - Dashboard utilisateur affiche "Dernière sauvegarde"

11. **Authentification P2P par mot de passe** 🔐 Session 10
    - **Mot de passe serveur** : Protège les endpoints `/api/sync/*` contre accès non autorisés
    - **Mot de passe pair** : Authentification auprès des serveurs distants
    - Middleware `syncAuthMiddleware` avec header `X-Sync-Password`
    - Interface admin `/admin/settings` pour configurer le mot de passe serveur
    - Champ mot de passe lors de l'ajout/édition de pairs
    - Hachage bcrypt côté serveur (stockage sécurisé)
    - Rétrocompatibilité : Sans mot de passe configuré = accès libre

12. **Gestion des backups entrants** 👥 Session 11
    - Vue `/admin/incoming` pour visualiser les pairs qui stockent des backups
    - Statistiques : nombre de pairs, fichiers, espace utilisé
    - Suppression de backups entrants
    - Carte dashboard pour accès rapide

13. **Édition de pairs** ✏️ Session 11
    - Interface `/admin/peers/{id}/edit` pour modifier la configuration
    - Modification nom, adresse, port, mot de passe, statut, fréquence sync
    - Gestion intelligente du mot de passe (conserver/modifier/supprimer)
    - Test d'authentification intégré au bouton "Test"
    - Détection automatique des erreurs d'authentification (401/403)

14. **Installation automatisée**
    - Script `install.sh` zéro-touch
    - Configuration complète système
    - Support multi-distro (Fedora/RHEL/Debian)

15. **Restauration de fichiers avec interface web** 📂 Session 12
    - Liste des backups disponibles sur tous les pairs distants
    - Navigation dans l'arborescence des fichiers chiffrés
    - Déchiffrement automatique côté serveur d'origine
    - **Sélection multiple** : Checkboxes pour fichiers et dossiers
    - **Téléchargement ZIP** : Plusieurs fichiers/dossiers en un clic
    - **Expansion récursive** : Sélection d'un dossier inclut tous les sous-fichiers
    - Barre d'outils avec compteur de sélection
    - Boutons "Tout sélectionner" / "Désélectionner tout"
    - Support des chemins avec espaces et caractères spéciaux
    - Streaming direct sans stockage temporaire

### 🚀 Déploiement

**DEV (192.168.83.99)** : ✅ Migration /srv/anemone complète + Quotas Btrfs actifs + Scheduler actif
**FR1 (192.168.83.96)** : ✅ Installation fraîche + Réception backups

**Tests validés** :
- ✅ Accès SMB depuis Windows : OK
- ✅ Accès SMB depuis Android : OK
- ✅ Création/lecture/écriture fichiers : OK
- ✅ **Blocage quota dépassé** : OK
- ✅ Privacy SMB (chaque user voit uniquement ses partages) : OK
- ✅ Multi-utilisateurs : OK
- ✅ SELinux (Fedora) : OK
- ✅ **Synchronisation automatique** : OK (Session 9)
- ✅ **Synchronisation incrémentale** : OK (fichiers modifiés/supprimés détectés)
- ✅ **Dashboard "Dernière sauvegarde"** : OK (affiche temps écoulé)
- ✅ **Authentification P2P** : OK (Session 10 - 401/403/200 selon mot de passe)
- ✅ **Vue backups entrants** : OK (Session 11 - affichage stats et backups)
- ✅ **Édition de pair** : OK (Session 11 - modification config complète)
- ✅ **Synchronisation avec authentification** : OK (Session 11 - DEV→FR1)
- ✅ **Fréquences par pair** : OK (Session 13 - interval/daily/weekly/monthly)
- ✅ **Restauration fichiers depuis pairs** : OK (Session 12 - liste, navigation, déchiffrement)
- ✅ **Téléchargement ZIP multiple** : OK (Session 12 - checkboxes, sélection, dossiers récursifs)
- ✅ **Encodage URL chemins spéciaux** : OK (Session 12 - espaces, caractères spéciaux)

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

## 📋 Résumé des sessions récentes

### Session 8 (7-8 Nov) - Synchronisation incrémentale
- ✅ Système de manifest pour tracking fichiers
- ✅ API endpoints pour sync fichier par fichier
- ✅ ~50% économie bande passante (seulement fichiers modifiés)
- ✅ Interface `/admin/sync` pour configuration
- **Détails** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`

### Session 9 (9 Nov) - Scheduler automatique + Bug fixes
- ✅ Goroutine background pour sync automatique
- ✅ Vérification toutes les 1 minute
- ✅ Fix dashboard "Dernière sauvegarde" (requête SQLite)
- ✅ Logs détaillés dans console
- **Détails** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`

### Session 10 (9 Nov) - Authentification P2P
- ✅ Mot de passe serveur (bcrypt) pour protéger `/api/sync/*`
- ✅ Mot de passe pair pour authentification sortante
- ✅ Middleware avec header `X-Sync-Password`
- ✅ Interface `/admin/settings` pour configuration
- ✅ Rétrocompatibilité (sans mot de passe = accès libre)
- **Détails** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`

### Session 11 (10 Nov) - Vue backups entrants + Édition pairs
- ✅ Vue `/admin/incoming` avec statistiques backups
- ✅ Interface `/admin/peers/{id}/edit` pour modification
- ✅ Gestion intelligente mot de passe (conserver/modifier/supprimer)
- ✅ Test d'authentification intégré
- ✅ Cartes dashboard (Paramètres serveur, Pairs connectés)
- **Détails** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`

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

**Cas d'usage** :
- Pair FR0 (interval 30min) : Backup très fréquent pour données critiques
- Pair FR1 (daily 23:00) : Backup quotidien pour récupération rapide
- Pair FR2 (weekly Samedi 23:00) : Snapshot hebdomadaire pour version intermédiaire
- Pair FR3 (monthly 1er 23:00) : Archive mensuelle pour rétention long terme

### 🔨 Composants créés/modifiés

**1. Database Migration** (`internal/database/migrations.go`)

Nouvelles colonnes ajoutées à la table `peers` :
```sql
sync_enabled BOOLEAN DEFAULT 1           -- Activer/désactiver sync pour ce pair
sync_frequency TEXT DEFAULT 'daily'      -- "interval", "daily", "weekly", "monthly"
sync_time TEXT DEFAULT '23:00'           -- Heure de sync (format HH:MM)
sync_day_of_week INTEGER                 -- 0-6 (0=dimanche), NULL si pas weekly
sync_day_of_month INTEGER                -- 1-31, NULL si pas monthly
sync_interval_minutes INTEGER DEFAULT 60 -- Intervalle en minutes pour "interval"
```

**2. Package peers** (`internal/peers/peers.go`)

Ajout de champs à la struct `Peer` :
```go
type Peer struct {
    // ... existing fields
    SyncEnabled         bool
    SyncFrequency       string   // "interval", "daily", "weekly", "monthly"
    SyncTime            string   // "HH:MM"
    SyncDayOfWeek       *int     // 0-6, NULL si pas weekly
    SyncDayOfMonth      *int     // 1-31, NULL si pas monthly
    SyncIntervalMinutes int      // Intervalle en minutes pour "interval"
}
```

Nouvelles fonctions :
- `UpdateLastSync(db, peerID)` : Met à jour le timestamp de dernière sync
- `ShouldSyncPeer(peer)` : Détermine si un pair doit être synchronisé maintenant
  - Interval : Vérifie si `now - lastSync >= interval` (en minutes)
  - Daily : Vérifie si on a passé l'heure de sync aujourd'hui et qu'on n'a pas encore sync aujourd'hui
  - Weekly : Vérifie le jour de la semaine + l'heure + qu'on n'a pas sync aujourd'hui
  - Monthly : Vérifie le jour du mois + l'heure + qu'on n'a pas sync aujourd'hui

**3. Package sync** (`internal/sync/sync.go`)

Nouvelle fonction `SyncPeer()` pour synchroniser tous les shares vers UN seul pair spécifique.

**4. Scheduler** (`internal/scheduler/scheduler.go`)

Parcourt tous les pairs individuellement et synchronise ceux qui doivent l'être selon leur fréquence configurée.

**5. Interfaces admin**

**Add Peer** (`web/templates/admin_peers_add.html`) :
- Checkbox "Activer la synchronisation automatique"
- Dropdown "Fréquence" (interval/daily/weekly/monthly)
- **Pour "Interval"** : Input numérique + dropdown unité (minutes/heures)
  - Valeur convertie en minutes avant stockage en base
  - Exemple : 2 heures → stocké comme 120 minutes
  - Masque le champ "Heure de synchronisation"
- Input time "Heure de synchronisation" (pour daily/weekly/monthly)
- Dropdown "Jour de la semaine" (affiché conditionnellement pour weekly)
- Input "Jour du mois" (affiché conditionnellement pour monthly)
- JavaScript pour affichage conditionnel des champs

**Edit Peer** (`web/templates/admin_peers_edit.html`) :
- Mêmes champs que Add Peer
- Valeurs pré-remplies depuis la base de données
- **Pour "Interval"** : Affiche la valeur en minutes depuis la base (utilisateur peut changer l'unité)
- JavaScript identique pour affichage conditionnel

**6. Handlers** (`internal/web/router.go`)

**handleAdminPeersAdd** :
- Récupère et parse les champs de sync depuis le formulaire
- Pour "interval" : Parse `sync_interval_value` et `sync_interval_unit`
- Convertit en minutes (heures × 60) avant création du pair

**handleAdminPeersActions (case "update")** :
- Récupère et parse les champs de sync pour mise à jour
- Logique de conversion identique pour "interval"

### 📝 Fichiers créés/modifiés

**Modifiés** :
- `internal/database/migrations.go` (~20 lignes) - Migration colonnes sync
- `internal/peers/peers.go` (~110 lignes) - Struct + ShouldSyncPeer + UpdateLastSync + interval logic
- `internal/sync/sync.go` (~70 lignes) - Fonction SyncPeer
- `internal/scheduler/scheduler.go` (~30 lignes) - Boucle sur peers au lieu de config globale
- `internal/web/router.go` (~70 lignes) - Parse champs sync + conversion minutes/heures
- `web/templates/admin_peers_add.html` (~120 lignes) - Section config sync + interval
- `web/templates/admin_peers_edit.html` (~120 lignes) - Section config sync + interval

**Total** : ~540 lignes ajoutées/modifiées

### 🧪 Tests validés

**Migration DB** :
- ✅ Compilation réussie
- ✅ Serveur démarre sans erreur
- ✅ Pair existant FR1 migré avec config par défaut (daily, 23:00, interval=60)
- ✅ Nouvelles colonnes présentes en base

**Interface admin** :
- ✅ Option "Interval" visible dans dropdown fréquence
- ✅ Champs interval (valeur + unité) s'affichent conditionnellement
- ✅ Conversion minutes/heures fonctionne correctement
- ✅ Édition d'un pair affiche les valeurs correctement

**Scheduler** :
- ✅ Scheduler démarre avec message "checks every 1 minute"
- ✅ Parcourt les pairs individuellement
- ✅ Logique interval fonctionne (vérifie temps écoulé depuis last_sync)

**Rétrocompatibilité** :
- ✅ Pairs existants migrés automatiquement
- ✅ Valeurs par défaut : sync_enabled=1, frequency=daily, time=23:00, interval=60
- ✅ Aucune régression sur les fonctionnalités existantes

### 📊 Exemple de configuration

**Topologie recommandée** :
```
Serveur DEV (192.168.83.99)
├── Pair FR0 (future) : Interval 30min → Backup très fréquent
├── Pair FR1 (192.168.83.96) : Daily 23:00 → Backup quotidien
├── Pair FR2 (future) : Weekly Samedi 23:00 → Snapshot hebdo
└── Pair FR3 (future) : Monthly 1er 23:00 → Archive mensuelle
```

**Avantages** :
- ✅ Pas de duplication des fichiers (chaque pair reçoit les mêmes données)
- ✅ Plusieurs points de restauration à différentes fréquences
- ✅ Optimisation réseau : syncs espacées dans le temps
- ✅ Flexibilité : Chaque pair peut avoir sa propre stratégie
- ✅ Option interval pour données critiques nécessitant backups très fréquents

### 🔄 Remplacement de fonctionnalités

**Ancienne approche (Session 9)** :
- Table `sync_config` avec configuration globale
- Tous les pairs synchronisés en même temps
- Intervalle global (30min, 1h, 2h, 6h, fixed)

**Nouvelle approche (Session 13)** :
- Configuration par pair dans la table `peers`
- Chaque pair synchronisé indépendamment
- Fréquences plus claires et flexibles (interval/daily/weekly/monthly)

**Note** : La table `sync_config` est conservée mais n'est plus utilisée par le scheduler. Elle peut être supprimée dans une future version.

**Commits** :
```
À venir : feat: Add interval frequency option to peer sync configuration (Session 13)
```

**Statut** : 🟢 COMPLÈTE ET TESTÉE

---

## 🔧 Session 12 - 11 Novembre 2025 - Interface web de restauration depuis pairs distants

### 🎯 Objectif

Permettre aux utilisateurs de restaurer leurs fichiers depuis les backups P2P chiffrés stockés sur les serveurs pairs, avec déchiffrement local sur le serveur d'origine.

### ⚠️ Correction architecturale majeure

**Problème identifié** : L'architecture initiale permettait aux utilisateurs de restaurer depuis n'importe quel serveur (y compris les pairs qui ne possèdent pas leurs clés de chiffrement).

**Architecture corrigée** :
- Les utilisateurs se connectent sur leur **serveur d'origine** (où leurs clés sont stockées)
- Le serveur d'origine **interroge les pairs** pour lister les backups disponibles
- Les pairs **retournent les fichiers chiffrés** sans les déchiffrer (ils n'ont pas les clés)
- Le serveur d'origine **déchiffre localement** avec la clé utilisateur
- Les clés ne quittent jamais le serveur d'origine

**Exemple** :
```
Utilisateur marc@DEV (serveur d'origine)
    ↓ Se connecte et demande ses backups
DEV interroge FR1, FR2, FR3...
    ↓ Chaque pair liste ses backups pour marc
marc sélectionne un fichier depuis FR1
    ↓ DEV télécharge le fichier chiffré depuis FR1
FR1 retourne fichier.enc (sans déchiffrer)
    ↓ DEV déchiffre avec la clé de marc
marc reçoit le fichier déchiffré
```

### 🔨 Implémentation en 3 paliers

#### **PALIER 1** : API sur serveurs pairs (commit `28c26d7`)

Nouveaux endpoints sur les pairs (FR1, FR2...) pour servir les fichiers chiffrés :

**`GET /api/sync/list-user-backups?user_id=X`**
- Liste les backups disponibles pour un utilisateur
- Retourne : share_name, file_count, total_size, last_modified
- Protégé par mot de passe P2P

**`GET /api/sync/download-encrypted-manifest?user_id=X&share_name=Y`**
- Télécharge le manifest chiffré **sans le déchiffrer**
- Le pair ne touche pas au chiffrement

**`GET /api/sync/download-encrypted-file?user_id=X&share_name=Y&path=Z`**
- Télécharge un fichier chiffré **sans le déchiffrer**
- Protection contre path traversal
- Le pair est un simple serveur de stockage

#### **PALIER 2** : Interface interroge les pairs (commit `d1c1de2`)

Modification de l'interface pour lister les backups depuis les pairs :

**`GET /api/restore/backups`** (modifié) :
- Récupère tous les pairs configurés
- Interroge chaque pair via `/api/sync/list-user-backups`
- Agrège les résultats : peer_id, peer_name, share_name, stats
- Interface affiche "FR1 - backup" au lieu de "backup"

**Interface `restore.html`** (modifiée) :
- Dropdown affiche la source du backup (nom du pair)
- Stocke "peer_id:share_name" comme valeur
- Passe peer_id ET share_name aux API suivantes

#### **PALIER 3** : Téléchargement et déchiffrement distant (commit `f679d9f`)

Implémentation de la restauration distante avec déchiffrement local :

**`GET /api/restore/files?peer_id=X&backup=Y`** (modifié) :
- Récupère les infos du pair depuis la base de données
- Télécharge le manifest chiffré depuis le pair
- Déchiffre le manifest localement avec la clé utilisateur
- Construit l'arbre de fichiers
- Retourne la structure au navigateur

**`GET /api/restore/download?peer_id=X&backup=Y&file=Z`** (modifié) :
- Récupère les infos du pair depuis la base de données
- Télécharge le fichier chiffré depuis le pair
- Déchiffre le fichier en streaming avec la clé utilisateur
- Stream directement au navigateur (pas de stockage temporaire)

### 📦 Fichiers créés/modifiés

**Nouveaux packages** :
- `internal/restore/restore.go` (~310 lignes) - Logique de restauration et déchiffrement

**Modifiés** :
- `internal/web/router.go` (~540 lignes ajoutées) - 6 nouveaux handlers
- `web/templates/restore.html` (~380 lignes) - Interface utilisateur complète
- `web/templates/dashboard_user.html` (~15 lignes) - Carte "Restauration"

**Total** : ~1245 lignes ajoutées

### 🔒 Sécurité

**Chiffrement bout-en-bout conservé** :
- ✅ Les clés utilisateurs ne quittent jamais le serveur d'origine
- ✅ Les pairs ne peuvent pas déchiffrer les données (ils n'ont pas les clés)
- ✅ Déchiffrement uniquement sur le serveur d'origine
- ✅ Streaming direct (pas de stockage en clair)

**Contrôle d'accès** :
- ✅ Authentification sur serveur d'origine (RequireAuth)
- ✅ Mot de passe P2P pour protéger les API des pairs
- ✅ Isolation par user_id (vérifié côté serveur)
- ✅ Validation des chemins de fichiers (path traversal protection)

#### **PALIER 4** : Sélection multiple et téléchargement ZIP (11 Nov)

Ajout de la fonctionnalité de sélection multiple avec téléchargement ZIP :

**Frontend `restore.html`** :
- Checkbox à côté de chaque fichier et dossier
- Checkbox "Tout sélectionner" dans l'en-tête du tableau
- Barre d'outils de sélection (apparaît quand des éléments sont sélectionnés)
- Compteur d'éléments sélectionnés
- Boutons "Tout sélectionner" et "Désélectionner tout"
- Bouton "Télécharger (ZIP)" pour créer une archive
- JavaScript pour gestion de l'état de sélection

**Backend `router.go`** :
- Nouvel endpoint `POST /api/restore/download-multiple`
- Construction d'un arbre de fichiers depuis le manifest
- Expansion récursive des dossiers sélectionnés
- Téléchargement et déchiffrement de chaque fichier
- Création d'un ZIP en streaming avec `archive/zip`
- Fonction `buildURL()` pour encoder correctement les URLs (support espaces et caractères spéciaux)

**Fix de sécurité master key** :
- ✅ Master key maintenant lue **uniquement depuis la base de données** (`system_config.master_key`)
- ✅ Plus de fichier `/srv/anemone/keys/master.key` (supprimé)
- ✅ Architecture cohérente : toute la configuration dans la DB
- ✅ Déployé sur DEV et FR1

### 🧪 Tests validés

✅ **Liste des backups** : Affichage correct depuis pairs distants
✅ **Navigation dans fichiers** : Arborescence et breadcrumb fonctionnels
✅ **Téléchargement simple** : Fichier individuel déchiffré correctement
✅ **Sélection multiple** : Checkboxes et compteur fonctionnent
✅ **Téléchargement ZIP** : Un seul fichier → ZIP OK
✅ **Téléchargement ZIP dossier** : Dossier avec sous-dossiers → Tous les fichiers inclus
✅ **Chemins avec espaces** : Encodage URL correct (ex: "ThinPrint Client Windows 13/Setup.exe")
✅ **Déchiffrement automatique** : Pas besoin de clé utilisateur, transparent

### 📊 Logs à vérifier

**Sur DEV** (serveur d'origine) :
```
User marc downloaded file documents/report.txt from peer FR1 backup backup
```

**Sur FR1** (serveur pair) :
```
Sent encrypted manifest for user 19 share backup
Sent encrypted file documents/report.txt for user 19 share backup
```

### 🔄 Déploiement

**DEV (192.168.83.5)** :
- ✅ Binaire compilé avec restauration distante + sélection multiple + ZIP
- ✅ Fix master key (lecture depuis DB)
- ✅ Templates à jour (restore.html, dashboard_user.html)
- ✅ Service redémarré et fonctionnel

**FR1 (192.168.83.16)** :
- ✅ Binaire compilé avec API de téléchargement chiffré
- ✅ Fix master key (lecture depuis DB)
- ✅ Support encodage URL pour chemins spéciaux
- ✅ Service redémarré et fonctionnel

### 📝 Commits

```
28c26d7 - feat: Add remote restore API endpoints on peer servers (Palier 1/4)
d1c1de2 - feat: Query peer servers for remote backups (Palier 2/4)
f679d9f - feat: Implement remote restore with local decryption (Palier 3/4)
4f54713 - fix: Add FormatBytes and FormatTime to global template functions
c596396 - feat: Add web interface for file restoration from encrypted backups (Session 12) [INITIAL]
À venir  - feat: Add multiple file selection and ZIP download (Palier 4/4)
À venir  - fix: Read master key from database instead of file (security)
À venir  - fix: URL encoding for paths with spaces and special characters
```

### ⚠️ Notes importantes

1. **Architecture P2P** : Chaque serveur peut être à la fois serveur d'origine (pour ses utilisateurs) et serveur pair (pour les utilisateurs d'autres serveurs)

2. **Pas d'interface sur les pairs** : Les pairs gardent leur interface `/restore` car ils peuvent aussi être des serveurs d'origine pour leurs propres utilisateurs

3. **Rétrocompatibilité** : Les anciennes API restent fonctionnelles, seules les nouvelles API de restauration distante ont été ajoutées

4. **Mot de passe P2P obligatoire** : Pour la sécurité, il est fortement recommandé de configurer un mot de passe P2P sur chaque pair

5. **Fix sécurité master key** : La master key est maintenant stockée et lue uniquement depuis la base de données, plus de fichier en clair

**Statut** : 🟢 **COMPLÈTE ET TESTÉE**

---

## 📝 Prochaines étapes (Roadmap)

### 🎯 Priorité 1 - Court terme

**Session 12 : Interface web de restauration** 📂
- ✅ **COMPLÈTE ET TESTÉE** - Voir section ci-dessus
- ✅ Sélection multiple et téléchargement ZIP
- ✅ Fix sécurité master key

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
- **Audit du chiffrement P2P**
  - Vérifier que les fichiers sont bien chiffrés sur les pairs
  - Tester le déchiffrement depuis le serveur d'origine uniquement
  - Vérifier l'impossibilité de déchiffrer depuis un pair
- **Audit des logs**
  - Vérifier qu'aucune donnée sensible n'est loggée
  - Vérifier l'absence de mots de passe en clair dans les logs
- **Tests de pénétration**
  - Brute force login
  - Tentatives d'élévation de privilèges
  - Tentatives d'accès aux données d'autres utilisateurs
  - Tests XSS et injections
- **Documentation**
  - Documenter les bonnes pratiques de sécurité
  - Créer un guide de déploiement sécurisé
  - Documenter les procédures d'urgence

**Session 15 : Export/Import configuration serveur** 💾
- Export complet de la configuration serveur (JSON chiffré)
  - Base de données (users, peers, shares, quotas, config)
  - Clés de chiffrement
  - Configuration Samba
  - Métadonnées système
- Script `restore_server.sh` pour restauration complète
  - Usage : `bash restore_server.sh config_backup.json.enc master_key`
  - Restauration automatique de tous les paramètres
  - Recréation des utilisateurs système et SMB
  - Régénération des certificats TLS
- Chiffrement AES-256-GCM avec clé admin

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

3. **Service systemd** 🔄
   - Démarrage automatique au boot
   - Gestion propre du service (start/stop/restart/status)
   - Logs systemd intégrés
   - Script d'installation automatique

4. **Rate limiting anti-bruteforce** 🛡️
   - Protection sur `/login` et `/api/sync/*`
   - Bannissement temporaire après X tentatives échouées
   - Whitelist IP de confiance

5. **Statistiques détaillées de synchronisation** 📊
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

4. **Interface de monitoring avancée**
   - Dashboard temps réel avec WebSocket
   - Alertes configurables
   - Intégration Prometheus/Grafana

5. **Chiffrement asymétrique**
   - Clés publiques/privées RSA ou Ed25519
   - Échange de clés sécurisé entre pairs
   - Signature des manifests

### 📝 Fonctionnalités à évaluer (impact ressources)

- **Versioning des fichiers** : Conservation de N versions d'un fichier lors des syncs, permettant de revenir en arrière en cas de corruption/suppression accidentelle. Nécessite tests de charge pour évaluer impact disque/performance.

- **Authentification 2FA/MFA** : Authentification à deux facteurs avec TOTP (Google Authenticator, etc.). Jugée trop lourde pour un contexte homelab avec certificats auto-signés.

### 📌 Notes

- **Bandwidth throttling** : Non prioritaire car les fréquences différenciées par pair (interval/daily/weekly/monthly) permettent déjà de planifier les syncs hors heures de pointe.

- **Politique de rétention automatique** : Remplacée par le système de fréquence de synchronisation par pair, permettant des snapshots à différentes fréquences sans complexité supplémentaire.

---

**État global** : 🟢 GESTION COMPLÈTE DES PAIRS AVEC FRÉQUENCES PERSONNALISABLES
**Prochaine étape** : Interface web de restauration (Session 12)
