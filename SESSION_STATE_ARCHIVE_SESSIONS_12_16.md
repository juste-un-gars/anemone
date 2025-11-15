# 🪸 Anemone - Archive Sessions 12-16

> Archive des sessions complètes et testées (Session 12 à 16)

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

**Statut** : 🟢 **COMPLÈTE ET TESTÉE**

---

## 🔧 Session 15 - 12 Novembre 2025 - Backups serveur automatiques

### 🎯 Objectif

Implémenter un système de sauvegarde automatique de la configuration du serveur (disaster recovery) avec backups quotidiens, rotation automatique, et téléchargement sécurisé avec re-chiffrement.

### ✅ Architecture implémentée

**Fonctionnalités** :
- **Backups automatiques quotidiens** : Scheduler qui s'exécute chaque jour à 4h du matin
- **Rotation automatique** : Conservation des 10 derniers backups, suppression automatique des anciens
- **Backups manuels** : Bouton "Sauvegarder maintenant" dans l'interface admin
- **Téléchargement sécurisé** : Re-chiffrement à la volée avec mot de passe utilisateur (min 12 caractères)
- **Stockage chiffré** : Backups stockés chiffrés avec la master key du serveur
- **Interface dédiée** : Page `/admin/backup` avec liste des sauvegardes et métadonnées

**Contenu des backups** :
- Configuration complète du serveur
- Utilisateurs et leurs clés de chiffrement
- Partages et configuration SMB
- Pairs P2P et configuration de synchronisation
- Quotas et paramètres système
- Clés système (master key)

**Architecture de sécurité** :
```
Création backup → Chiffrement avec master_key → Stockage /srv/anemone/backups/server/
Téléchargement → Déchiffrement avec master_key → Re-chiffrement avec mot de passe utilisateur → Download
```

**Statut** : 🟢 **IMPLÉMENTÉ ET TESTÉ**

---

## 🔧 Session 16 - 14 Novembre 2025 - Restauration des mots de passe SMB après backup/restore

### 🎯 Objectif

Permettre la restauration automatique des mots de passe SMB lors d'une restauration serveur, en stockant les mots de passe chiffrés avec la master key.

### ⚠️ Problème identifié

Lors des tests de restauration sur un serveur propre (FR2), un problème critique a été découvert :
- Les utilisateurs peuvent se connecter à l'interface web après restauration ✅
- **MAIS** : Les mots de passe SMB ne fonctionnent pas ❌
- Le script de restauration utilisait un mot de passe temporaire "anemone123" pour tous les utilisateurs
- Problème : Le hash bcrypt stocké en base est à sens unique, impossible de récupérer le mot de passe original

### ✅ Solution implémentée

**Architecture de double stockage** :
- **Bcrypt hash** : Pour l'authentification web (sécurité maximale, à sens unique)
- **Encrypted password** : Pour la restauration SMB (réversible avec master key)

**Flux de données** :
```
Création/Modification mot de passe
    ↓
Génère bcrypt hash (auth web)
    +
Chiffre mot de passe avec master_key (AES-256-GCM)
    ↓
Stockage DB : password_hash + password_encrypted
    ↓
Backup serveur → Inclut password_encrypted
    ↓
Restauration → Déchiffre avec master_key → Configure SMB
```

### 🧪 Tests effectués

**Sur FR1 (serveur source)** :
- ✅ Compiler le nouveau code avec password_encrypted
- ✅ Créer un nouvel utilisateur (le mot de passe doit être chiffré automatiquement)
- ✅ Changer un mot de passe existant (doit mettre à jour password_encrypted)
- ✅ Créer un backup serveur
- ✅ Vérifier que password_encrypted est présent dans le backup

**Sur FR2 (serveur cible - propre)** :
- ✅ Lancer le script de restauration
- ✅ Vérifier la compilation de anemone-decrypt-password
- ✅ Vérifier que les mots de passe SMB sont restaurés
- ✅ Tester connexion SMB avec les vrais mots de passe
- ✅ Tester connexion web avec les vrais mots de passe

**Statut** : 🟢 **COMPLÈTE ET TESTÉE AVEC SUCCÈS**

---

## 🔧 Session 17 (Partie 1) - 15 Novembre 2025 - Re-chiffrement des clés utilisateur lors de la restauration

### 🎯 Objectif

Corriger le problème critique de restauration des fichiers après restauration serveur en re-chiffrant les clés utilisateur avec la nouvelle master key.

### 🐛 Problème découvert

Lors des tests de restauration FR1 → FR3 avec backup sur FR2 :
- ✅ La configuration serveur est restaurée correctement
- ✅ Les comptes utilisateurs sont restaurés correctement
- ✅ Les mots de passe SMB sont restaurés et re-chiffrés (Session 16)
- ❌ **La restauration automatique des fichiers ÉCHOUE** avec l'erreur :
   ```
   Bulk restore failed: failed to decrypt user key:
   failed to decrypt: cipher: message authentication failed
   ```

### ✅ Solution implémentée

**Principe** : Re-chiffrer `encryption_key_encrypted` avec la nouvelle master key lors de la restauration, exactement comme pour `password_encrypted`.

**Fichiers créés** :
- `cmd/anemone-reencrypt-key/main.go` - Outil CLI de re-chiffrement
- Modifications de `restore_server.sh` pour re-chiffrer les clés

### 🔨 Problèmes rencontrés et correctifs appliqués

1. **Double encodage base64** dans `encryption_key_encrypted`
2. **Type de données** dans export backup
3. **Lecture SQLite BLOB vs TEXT**
4. **Binaire incorrect exécuté** sur FR1 et FR3
5. **Insertion BLOB au lieu de TEXT**
6. **Format Manifest incompatible**
7. **Nom de share hardcodé** au lieu de lookup DB
8. **Share manquant** dans la base de données
9. **Convention de nommage** des shares de backup

Tous ces problèmes ont été résolus avec succès.

**Statut** : 🟢 **COMPLÈTE - Tous les problèmes d'encodage et de manifest résolus**
