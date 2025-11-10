# 🪸 Anemone - État du Projet

**Dernière session** : 2025-11-10 (Session 11 - Vue "Pairs connectés à moi" + Édition de pair)
**Status** : 🟢 GESTION COMPLÈTE DES PAIRS ET BACKUPS ENTRANTS

> **Note** : L'historique des sessions 1-7 a été archivé dans `SESSION_STATE_ARCHIVE.md`

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
   - CRUD complet
   - Test connexion HTTPS
   - Statuts (online/offline/error)
   - **Synchronisation manuelle** : Bouton sync par partage
   - **Synchronisation automatique** : Scheduler intégré ✨ Session 9
   - **Chiffrement E2E** : AES-256-GCM par utilisateur

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
    - Intervalles configurables : 30min, 1h, 2h, 6h, heure fixe
    - Interface admin `/admin/sync` pour configuration
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
    - Logs d'authentification dans la console serveur

12. **Installation automatisée**
    - Script `install.sh` zéro-touch
    - Configuration complète système
    - Support multi-distro (Fedora/RHEL/Debian)

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

## 🔧 Session 8 - 7-8 Novembre 2025 - Synchronisation incrémentale

### 🎯 Objectif

Remplacer la synchronisation monolithique (tar.gz complet) par une synchronisation incrémentale fichier par fichier (type rclone).

### ✅ Phases complétées

**Phase 1 : Système de manifest**
- Fichier `internal/sync/manifest.go` (210 lignes)
- Fonctions : `BuildManifest()`, `CompareManifests()`, `CalculateChecksum()`
- Tests unitaires : 7/7 PASS

**Phase 2 : Synchronisation incrémentale**
- 4 nouveaux API endpoints : GET/PUT manifest, POST/DELETE file
- Fonction `SyncShareIncremental()` pour upload fichier par fichier
- Stockage : `/srv/anemone/backups/incoming/{user_id}_{share_name}/`
- Serveur distant n'a plus besoin que l'utilisateur existe localement

**Phase 3 : Interface admin**
- Page `/admin/sync` pour configuration
- Table `sync_config` en base de données
- Package `internal/syncconfig/` pour gestion configuration
- Fonction `SyncAllUsers()` pour synchronisation globale
- Bouton "Forcer la synchronisation"
- Tableau des 20 dernières synchronisations

### 📊 Résultats

- ✅ Seulement les fichiers modifiés sont transférés (~50% économie bande passante)
- ✅ Chaque fichier chiffré individuellement (AES-256-GCM)
- ✅ Architecture simplifiée (serveur distant = simple stockage)
- ✅ Sécurité end-to-end maintenue

**Commits** :
```
368faa1 - feat: Implement automatic sync configuration interface (Phase 3/4)
c95f7a6 - feat: Implement incremental P2P sync with file-by-file transfer (Phase 2/4)
1322625 - feat: Implement manifest system for incremental P2P sync (Phase 1/4)
```

**Statut** : 🟢 COMPLÈTE

---

## 🔧 Session 9 - 9 Novembre 2025 - Scheduler automatique + Bug fixes

### 🎯 Objectif

Implémenter le scheduler automatique pour déclencher les synchronisations selon l'intervalle configuré.

### ✅ Implémentation

**1. Package scheduler** (`internal/scheduler/scheduler.go`)
- Goroutine background lancée au démarrage du serveur
- Vérifie toutes les 1 minute s'il faut synchroniser
- Lit la configuration depuis `sync_config` en base
- Appelle `sync.SyncAllUsers()` si nécessaire
- Met à jour `sync_config.last_sync` après chaque sync
- Logs détaillés dans la console

**2. Intégration dans main.go**
- Import du package `scheduler`
- Appel de `scheduler.Start(db)` avant le serveur HTTP
- Le scheduler tourne en parallèle du serveur web

**3. Logique de déclenchement** (`syncconfig.ShouldSync()`)
- Si `last_sync` est NULL → première sync (trigger immédiat)
- Si intervalle = "fixed" → vérifie l'heure quotidienne
- Sinon → vérifie si `now - last_sync >= interval`

**Intervalles supportés** :
- `30min` : Toutes les 30 minutes
- `1h` : Toutes les heures
- `2h` : Toutes les 2 heures
- `6h` : Toutes les 6 heures
- `fixed` : Heure fixe quotidienne (0-23)

### 🐛 Bug fixes

**Bug 1 : Dashboard "Dernière sauvegarde" affichait toujours "Jamais"**

**Cause** : Requête SQL incorrecte
```sql
-- AVANT (ne fonctionnait pas avec SQLite)
SELECT MAX(completed_at) FROM sync_log ...
```
SQLite retourne `MAX(completed_at)` comme une **string**, pas un **time.Time**.

**Solution** :
```sql
-- APRÈS (fonctionne parfaitement)
SELECT completed_at FROM sync_log
WHERE user_id = ? AND status = 'success'
ORDER BY completed_at DESC
LIMIT 1
```

**Fichier modifié** : `internal/web/router.go:395-413`

**Amélioration bonus** : Affichage en minutes si < 1h
```go
if duration < time.Hour {
    stats.LastBackup = fmt.Sprintf("Il y a %d minutes", int(duration.Minutes()))
} else if duration < 24*time.Hour {
    stats.LastBackup = fmt.Sprintf("Il y a %d heures", int(duration.Hours()))
} else {
    stats.LastBackup = fmt.Sprintf("Il y a %d jours", int(duration.Hours()/24))
}
```

### 🧪 Tests validés

**Test 1 : Synchronisation automatique**
- ✅ Configuration activée avec intervalle 30min
- ✅ Scheduler démarre au lancement du serveur
- ✅ Première sync déclenchée automatiquement (last_sync=NULL)
- ✅ Synchronisations suivantes toutes les 30 minutes
- ✅ Logs visibles dans la console :
  ```
  2025/11/09 09:43:25 🔄 Scheduler: Triggering automatic synchronization...
  2025/11/09 09:43:26 ✅ Scheduler: Sync completed successfully - 2 shares synchronized
  ```

**Test 2 : Dashboard utilisateur**
- ✅ "Dernière sauvegarde" affiche "Il y a X minutes"
- ✅ Mise à jour en temps réel après chaque sync
- ✅ Plus d'erreur "Jamais" pour utilisateurs avec syncs

**Test 3 : Synchronisation incrémentale**
- ✅ Fichiers ajoutés à 8h57 → synchronisés à 9h13
- ✅ Ajout/modification détectés correctement
- ✅ Suppression répliquée sur le pair distant
- ✅ Fichiers stockés chiffrés sur FR1

### 📝 Fichiers créés/modifiés

**Créés** :
- `internal/scheduler/scheduler.go` (+56 lignes)

**Modifiés** :
- `cmd/anemone/main.go` (+3 lignes - import + appel scheduler)
- `internal/web/router.go` (+10 lignes - fix requête SQL)

### 📊 Logs de production

```
2025/11/09 10:02:31 🪸 Starting Anemone NAS...
2025/11/09 10:02:31 🔄 Starting automatic synchronization scheduler...
2025/11/09 10:02:31 ✅ Automatic synchronization scheduler started (checks every 1 minute)
2025/11/09 10:02:31 🔒 HTTPS server listening on https://localhost:8443
```

**Commits** :
```
À venir : feat: Implement automatic sync scheduler (Session 9)
          fix: Dashboard last backup display with SQLite-compatible query
```

**Statut** : 🟢 COMPLÈTE ET TESTÉE

---

## 🔧 Session 10 - 9 Novembre 2025 - Authentification P2P par mot de passe

### 🎯 Objectif

Sécuriser les endpoints de synchronisation P2P pour empêcher les connexions non autorisées. Problème identifié : n'importe quel serveur pouvait stocker des backups sans authentification.

### ✅ Architecture implémentée

**Système à deux niveaux** :

1. **Mot de passe SERVEUR** (dans `system_config.sync_auth_password`)
   - Protège les endpoints `/api/sync/*` de CE serveur
   - Stocké hashé avec bcrypt (sécurité maximale)
   - Configurable via `/admin/settings`
   - Les pairs doivent fournir ce mot de passe pour se connecter

2. **Mot de passe PAIR** (dans `peers.password`)
   - Utilisé pour s'authentifier auprès des AUTRES serveurs
   - Stocké en clair (transmis via HTTPS chiffré)
   - Configurable lors de l'ajout/édition d'un pair

**Rétrocompatibilité** : Si aucun mot de passe serveur n'est configuré, les endpoints restent accessibles sans authentification.

### 🔨 Composants créés/modifiés

**1. Database Migration** (`internal/database/migrations.go`)
- Ajout colonne `password TEXT` à la table `peers`
- Migration automatique au démarrage

**2. Package syncauth** (`internal/syncauth/syncauth.go` - NOUVEAU)
- `GetSyncAuthPassword(db)` : Récupère le hash du mot de passe serveur
- `SetSyncAuthPassword(db, password)` : Configure/modifie le mot de passe (avec bcrypt)
- `CheckSyncAuthPassword(db, password)` : Vérifie si le mot de passe fourni est correct
- `IsConfigured(db)` : Vérifie si un mot de passe est configuré

**3. Middleware d'authentification** (`internal/web/router.go`)
```go
func (s *Server) syncAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    // 1. Vérifie si un mot de passe est configuré
    // 2. Si non → accès libre (backward compatibility)
    // 3. Si oui → exige header X-Sync-Password
    // 4. Valide le mot de passe avec bcrypt
    // 5. Retourne 401 (pas de header) ou 403 (mauvais mot de passe)
}
```

Appliqué sur :
- `/api/sync/manifest` (GET/PUT)
- `/api/sync/file` (POST/DELETE)
- `/api/sync/receive` (ancien endpoint)

**4. Client de synchronisation** (`internal/sync/sync.go`)
- Modification de `SyncAllUsers()` pour récupérer le mot de passe du pair
- Ajout du header `X-Sync-Password` sur toutes les requêtes HTTP :
  - GET manifest (vérifier état distant)
  - POST file (upload fichier chiffré)
  - DELETE file (supprimer fichier obsolète)
  - PUT manifest (mettre à jour manifest distant)

**5. Structure Peer** (`internal/peers/peers.go`)
```go
type Peer struct {
    ID        int
    Name      string
    Address   string
    Port      int
    PublicKey *string
    Password  *string  // NOUVEAU - Can be NULL
    Enabled   bool
    // ...
}
```
Toutes les fonctions CRUD mises à jour (Create, GetByID, GetAll, Update).

**6. Interface admin - Settings** (`web/templates/admin_settings.html` - NOUVEAU)
- Page `/admin/settings` pour configurer le mot de passe serveur
- Indicateur de statut (configuré / non configuré)
- Formulaire avec confirmation du mot de passe
- Validation : minimum 8 caractères
- Messages de succès/erreur
- Info-box expliquant le fonctionnement

**7. Interface admin - Add Peer** (`web/templates/admin_peers_add.html`)
- Ajout du champ "Mot de passe de synchronisation" (optionnel)
- Type `password` pour masquer la saisie
- Texte d'aide explicatif

**8. Handlers** (`internal/web/router.go`)
- `handleAdminSettings()` : Affiche la page de configuration
- `handleAdminSettingsSyncPassword()` : Traite le formulaire de configuration

### 🧪 Tests validés

**Test 1 : Sans mot de passe (attendu: 401)**
```bash
curl https://localhost:8443/api/sync/manifest?user_id=1&share_name=backup
→ HTTP 401: "Unauthorized: X-Sync-Password header required" ✅
```

**Test 2 : Mauvais mot de passe (attendu: 403)**
```bash
curl -H "X-Sync-Password: wrongpassword" ...
→ HTTP 403: "Forbidden: Invalid password" ✅
```

**Test 3 : Bon mot de passe (attendu: succès)**
```bash
curl -H "X-Sync-Password: testpass123" ...
→ HTTP 404: "No manifest found" (authentification OK, pas de manifest) ✅
```

**Logs serveur** :
```
2025/11/09 11:59:45 Sync auth failed: No X-Sync-Password header from [::1]:46814
2025/11/09 11:59:50 Sync auth failed: Invalid password from [::1]:46828
```
(Le 3ème test réussit sans log d'erreur)

### 📝 Fichiers créés/modifiés

**Créés** :
- `internal/syncauth/syncauth.go` (+76 lignes) - Package d'authentification
- `web/templates/admin_settings.html` (+191 lignes) - Interface de configuration

**Modifiés** :
- `internal/database/migrations.go` - Migration `password` column
- `internal/peers/peers.go` - Peer struct + CRUD avec password
- `internal/web/router.go` - Middleware + routes `/admin/settings`
- `internal/sync/sync.go` - Envoi header `X-Sync-Password`
- `web/templates/admin_peers_add.html` - Champ password

**Total** : ~350 lignes ajoutées/modifiées

### 📊 Détails techniques

**Flux d'authentification** :
1. Admin configure mot de passe via `/admin/settings` → stocké hashé en DB
2. Admin ajoute pair FR1 avec le mot de passe de FR1 → stocké en clair
3. Lors de la sync, le serveur DEV envoie `X-Sync-Password: password_de_fr1`
4. FR1 reçoit la requête → middleware vérifie le mot de passe
5. Si valide → accepte le backup, sinon → rejette avec 401/403

**Sécurité** :
- ✅ Mot de passe serveur hashé avec bcrypt (cost 10)
- ✅ Transmission HTTPS chiffrée (header en clair dans HTTPS)
- ✅ Logs d'authentification pour monitoring
- ✅ Pas de rate limiting (TODO pour production)

**Commits** :
```
À venir : feat: Implement P2P password authentication (Session 10)
```

**Statut** : 🟢 COMPLÈTE ET TESTÉE

---

## 🔧 Session 11 - 10 Novembre 2025 - Vue "Pairs connectés" + Édition de pair

### 🎯 Objectif

Permettre aux admins de visualiser quels serveurs distants stockent des backups sur leur serveur, et de modifier la configuration des pairs existants.

### ✅ Fonctionnalités implémentées

**1. Vue "Pairs connectés à moi"** (`/admin/incoming`)
- **Package** `internal/incoming/incoming.go` (192 lignes)
  - `ScanIncomingBackups()` : Scanne `/srv/anemone/backups/incoming/`
  - `DeleteIncomingBackup()` : Supprime un backup
  - `FormatBytes()`, `FormatTimeAgo()` : Utilitaires de formatage
- **Interface admin** avec statistiques :
  - Nombre de pairs connectés
  - Nombre total de fichiers stockés
  - Espace disque utilisé
- **Tableau détaillé** par backup :
  - Username + User ID
  - Nom du partage (backup/data)
  - Nombre de fichiers
  - Taille totale
  - Date de dernière modification
  - Indicateur de présence du manifest
  - Bouton "Supprimer" avec confirmation
- État vide si aucun backup reçu

**2. Interface d'édition de pair** (`/admin/peers/{id}/edit`)
- **Handlers** dans `router.go` :
  - Case `"edit"` : Affiche le formulaire (GET)
  - Case `"update"` : Traite la soumission (POST)
- **Formulaire pré-rempli** avec :
  - Nom du pair
  - Adresse
  - Port
  - Mot de passe (optionnel)
  - Statut activé/désactivé
- **Gestion intelligente du mot de passe** :
  - Laisser vide = conserver l'actuel
  - Remplir = modifier
  - Checkbox "Supprimer le mot de passe" = effacer
- **Section infos** affichant :
  - ID, statut, dates de création/modification
- **Bouton "Éditer"** ajouté sur `/admin/peers`

### 📝 Fichiers créés/modifiés

**Créés** :
- `internal/incoming/incoming.go` (+192 lignes)
- `web/templates/admin_incoming.html` (+226 lignes)
- `web/templates/admin_peers_edit.html` (+232 lignes)

**Modifiés** :
- `internal/web/router.go` (+150 lignes)
  - Import package `incoming`
  - Routes `/admin/incoming`, `/admin/incoming/delete`
  - Handlers `handleAdminIncoming()`, `handleAdminIncomingDelete()`
  - Cases `"edit"` et `"update"` dans `handleAdminPeersActions()`
- `web/templates/admin_peers.html` (+3 lignes)
  - Lien "Éditer" ajouté pour chaque pair

**Total** : ~650 lignes ajoutées

### 🔒 Sécurité

- Vérification que les chemins à supprimer sont bien dans `/srv/anemone/`
- Authentification admin requise pour toutes les opérations
- Logs des actions administratives
- Protection contre les path traversal attacks

### 📊 Architecture

**Structure des backups entrants** :
```
/srv/anemone/backups/incoming/
├── 1_backup/           # user_id=1, share=backup
│   ├── manifest.json.enc
│   ├── file1.txt.enc
│   └── file2.txt.enc
└── 2_data/             # user_id=2, share=data
    ├── manifest.json.enc
    └── file3.txt.enc
```

**Flux d'édition de pair** :
1. Admin clique "Éditer" → GET `/admin/peers/{id}/edit`
2. Formulaire pré-rempli affiché
3. Admin modifie et soumet → POST `/admin/peers/{id}/update`
4. Validation et mise à jour en DB
5. Redirection vers `/admin/peers`

### 🧪 Tests à effectuer

**Vue "Pairs connectés"** :
- ✅ Compilation réussie
- ⏳ Accès à `/admin/incoming`
- ⏳ Affichage correct avec/sans backups
- ⏳ Suppression d'un backup
- ⏳ Vérification des statistiques

**Édition de pair** :
- ✅ Compilation réussie
- ⏳ Bouton "Éditer" visible sur `/admin/peers`
- ⏳ Formulaire pré-rempli correctement
- ⏳ Modification des champs (nom, adresse, port)
- ⏳ Modification du mot de passe
- ⏳ Suppression du mot de passe
- ⏳ Changement du statut activé/désactivé

**Commits** :
```
6dfe2dd - feat: Implement incoming backups view and peer edit interface (Session 11)
```

**Statut** : 🟢 IMPLÉMENTÉE - TESTS EN ATTENTE

---

## 📝 Prochaines étapes (Roadmap)

### Court terme (Session 12 - Prochaine)

1. 🔜 **Interface web de restauration** (Phase 4 - Session 8)
   - Explorateur de fichiers pour naviguer dans les backups
   - Téléchargement sélectif de fichiers
   - Restauration avec confirmation

### Moyen terme
1. 🔜 Notifications (email/web) pour sync réussies/échouées
2. 🔜 Bandwidth throttling (limite bande passante)
3. 🔜 Statistiques détaillées de synchronisation
4. 🔜 Service systemd pour démarrage automatique
5. 🔜 Rate limiting sur l'authentification (anti-bruteforce)

### Long terme
1. 🔜 Tests production sur multiples serveurs
2. 🔜 Multi-peer redundancy (plusieurs pairs pour un user)
3. 🔜 Backup/restore configuration complète
4. 🔜 Interface de monitoring avancée
5. 🔜 Chiffrement asymétrique avec clés publiques (RSA/Ed25519)

**État global** : 🟢 GESTION COMPLÈTE DES PAIRS ET BACKUPS ENTRANTS
**Prochaine étape** : Interface web de restauration (explorateur de backups)
