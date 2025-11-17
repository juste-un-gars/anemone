# 🗄️ Anemone - Archive Sessions 17-19

**Période** : 15-17 Novembre 2025
**Sessions archivées** : 17, 18, 19
**Raison** : Détails techniques complets, archivés pour alléger SESSION_STATE.md principal

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
2. **c9a7d10** : Fix bulk restore avec clé map manifest + setOwnership()
3. **778fa32** : Documentation de la session 18
4. **c869161** : Interface admin de restauration (commit initial session 18)

**État session 18** : 🟢 **COMPLÈTE - Restauration admin fonctionnelle à 100%**

---

## 🔧 Session 19 - 17 Novembre 2025 - Outil de décryptage manuel pour disaster recovery

**Date** : 2025-11-17
**Objectif** : Créer un outil CLI autonome pour décrypter manuellement les backups sans serveur
**Priorité** : 🟡 IMPORTANT → 🟢 COMPLÈTE

### 🎯 Contexte et Solution

**Problématique** :
- Les clés de chiffrement sont affichées une seule fois lors de la création/activation du compte
- Si le serveur principal est complètement perdu (panne matérielle, incendie, etc.)
- L'utilisateur possède toujours :
  1. Sa clé de chiffrement sauvegardée
  2. Les fichiers chiffrés sur les serveurs pairs (FR2, etc.)
- **Question** : Comment récupérer les fichiers sans le serveur principal ?

**Solution implémentée** :
- **Outil CLI standalone** : `anemone-decrypt`
- **Indépendance totale** : Fonctionne sans base de données, sans master key, sans serveur
- **Input** : Fichiers .enc + clé utilisateur (32 bytes base64)
- **Output** : Fichiers déchiffrés dans leur état original

### ✅ Architecture

**Hiérarchie de chiffrement existante** :
```
Master Key (unique par serveur, stockée en DB)
    ↓ chiffre
User Encryption Key (32 bytes, unique par utilisateur)
    ↓ chiffre
Fichiers de backup sur pairs distants
```

**Workflow disaster recovery** :
```
1. SSH sur serveur pair (ex: FR2)
2. Copier /srv/anemone/backups/incoming/X_sharename/*.enc
3. Exécuter: anemone-decrypt -key=<user_key> -dir=./backups -r
4. Récupération complète des fichiers déchiffrés ✅
```

### 🔨 Composants créés

**1. cmd/anemone-decrypt/main.go**
- Parser CLI avec flags (key, dir, out, recursive)
- Scan des fichiers .enc récursif ou non
- Décryptage batch avec barre de progression
- Gestion d'erreurs et cleanup automatique
- Affichage formaté (taille fichiers, statistiques)

**2. Fonctionnalités**
- `-key` : Clé de chiffrement base64 (obligatoire)
- `-dir` : Répertoire source (défaut: répertoire courant)
- `-out` : Répertoire destination (défaut: même que source)
- `-r` : Mode récursif pour sous-répertoires
- `-h` : Aide complète avec exemples

**3. Installation**
- Binaire compilé : `~/anemone/anemone-decrypt`
- Installation système : `/usr/local/bin/anemone-decrypt`
- Accessible partout : `anemone-decrypt -h`

### 🧪 Tests validés

**Test 1 : Fichiers générés localement**
- ✅ 5 fichiers de test créés avec clé connue
- ✅ Décryptage récursif réussi (5/5 fichiers)
- ✅ Contenu vérifié : identique à l'original

**Test 2 : Fichiers réels depuis FR2**
- ✅ 3 fichiers copiés depuis backup production (user "test")
- ✅ Clé utilisateur déchiffrée depuis DB : `0kMrSgGbiIWM8dggYP6nuCPcSAHlELQikuJz3LQvEec=`
- ✅ Décryptage réussi :
  - `printer-qrcode.pdf` : 5.5 KB (PDF 1 page)
  - `03.3mf` : 19.2 KB (fichier 3D printing)
  - `temp/multi_size_pages.pdf` : 4.0 KB (PDF 6 pages)
- ✅ Fichiers validés avec `file` : types corrects

**Test 3 : Gestion d'erreurs**
- ✅ Clé incorrecte détectée : "message authentication failed"
- ✅ Répertoire inexistant : erreur claire
- ✅ Absence de fichiers .enc : message informatif

### 💡 Points importants

**1. Sauvegarder la clé utilisateur**
- Affichée **UNE SEULE FOIS** lors de l'activation
- Stocker dans un gestionnaire de mots de passe
- Sans cette clé + sans serveur = données perdues

**2. Indépendance totale**
- Pas besoin de la master key
- Pas besoin de la base de données
- Pas besoin du serveur Anemone
- Juste : clé utilisateur + fichiers .enc

**3. Sécurité**
- Fichiers originaux .enc jamais modifiés
- En cas d'erreur, fichier output supprimé automatiquement
- Validation AEAD (AES-256-GCM) garantit l'intégrité

### 📝 Commits

```
e255d4d - feat: Add anemone-decrypt CLI tool for manual disaster recovery (Session 19) [17 Nov]
a93ab1a - fix: Correct admin dashboard stats and add server backup deletion [17 Nov]
```

**Détails** :
1. **e255d4d** : Outil anemone-decrypt CLI
   - Décryptage manuel sans serveur
   - Testé avec fichiers réels (3 PDF/3MF)
   - Installation système
2. **a93ab1a** : Corrections avant audit
   - Dashboard admin : stockage total tous users
   - Dernière sauvegarde : dernière sync globale
   - Ajout bouton suppression backups serveur

**État session 19** : 🟢 **COMPLÈTE - Outil de récupération manuelle opérationnel**

---

**Dernière mise à jour** : 2025-11-17
