# 🪸 Anemone - État du Projet

**Dernière session** : 2025-11-17 (Session 20 - Audit du code et nettoyage)
**Prochaine session** : Audit sécurité + Tests finaux
**Status** : 🟢 EN COURS - Audit des fichiers démarré

> **Note** : L'historique des sessions 1-7 a été archivé dans `SESSION_STATE_ARCHIVE.md`
> **Note** : Les détails techniques des sessions 8-11 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`
> **Note** : Les détails techniques des sessions 12-16 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md`
> **Note** : Les détails techniques des sessions 17-19 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md`

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

6. **Quotas utilisateur**
   - Quotas par utilisateur (backup + data)
   - Enforcement via Btrfs qgroups
   - Fallback via `dfree` script pour non-Btrfs
   - Interface admin pour édition quotas
   - Dashboard affichant utilisation temps réel

7. **Pairs P2P (Peer-to-Peer)**
   - Ajout/édition/suppression de pairs
   - Configuration URL + mot de passe + fréquence sync
   - Authentification mutual TLS
   - Test de connectivité
   - Dashboard avec statut de chaque pair

8. **Synchronisation P2P chiffrée**
   - **Chiffrement** : AES-256-GCM (chaque utilisateur a sa clé unique)
   - **Manifests** : Détection fichiers modifiés/supprimés (checksums SHA-256)
   - **Synchronisation incrémentale** : Seuls les fichiers modifiés sont envoyés
   - **Authentification P2P** : Vérification mot de passe avant sync
   - **Fréquence par pair** : Interval (30min, 1h, 2h, 6h), Daily, Weekly, Monthly
   - **Scheduler automatique** : Syncs planifiées selon fréquence configurée
   - **Logs de sync** : Table `sync_log` (status, files, bytes, duration)
   - **Dashboard** : Affichage "Dernière sauvegarde" par utilisateur

9. **Restauration fichiers utilisateur**
   - Interface utilisateur `/restore` pour voir backups disponibles
   - Arborescence de fichiers avec navigation
   - Téléchargement fichier individuel
   - Téléchargement ZIP multiple
   - Décryptage à la volée côté serveur
   - Support des chemins avec espaces et caractères spéciaux

10. **Backups serveur automatiques**
    - Scheduler quotidien à 4h du matin
    - Rotation automatique (10 derniers backups)
    - Re-chiffrement à la volée pour téléchargement sécurisé
    - Interface admin `/admin/backup`
    - **Suppression manuelle** : Bouton pour supprimer les anciens backups

11. **Restauration complète du serveur**
    - Script `restore_server.sh` pour restauration complète
    - **Re-chiffrement automatique** des mots de passe SMB avec nouvelle master key
    - **Re-chiffrement automatique** des clés utilisateur avec nouvelle master key
    - Création automatique des utilisateurs système et SMB
    - Configuration automatique des partages
    - Flag `server_restored` pour afficher page d'avertissement

12. **Interface admin de restauration utilisateurs** (Session 18)
    - Page `/admin/restore-users` listant tous les backups disponibles
    - Restauration contrôlée après disaster recovery
    - Workflow sécurisé : désactivation auto pairs → restauration → réactivation manuelle
    - Ownership automatique (fichiers appartiennent aux users)

13. **Outil de décryptage manuel** (Session 19)
    - **Commande CLI** : `anemone-decrypt` pour récupération manuelle des backups
    - **Décryptage sans serveur** : Utilise uniquement la clé utilisateur sauvegardée
    - **Mode récursif** : Support des sous-répertoires avec option `-r`
    - **Batch processing** : Déchiffre automatiquement tous les fichiers .enc
    - **Use case critique** : Récupération d'urgence si serveur complètement perdu
    - **Indépendance totale** : Fonctionne sans base de données ni master key

14. **Audit du code** (Session 20 - En cours)
    - Fichier de tracking `CHECKFILES.md` avec statuts par fichier
    - Répertoire `_audit_temp/` pour fichiers suspects
    - **Commandes CLI** : 9/9 vérifiées (8 OK, 1 déplacé)
    - **Fichiers déplacés** : `cmd/test-manifest/`, `base.html`
    - **Nettoyage recommandé** : `_old/` (78 MB, 2675 fichiers obsolètes)

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
- ✅ **Décryptage manuel sans serveur** : OK (Session 19)

**Structure de production** :
- Code : `~/anemone/` (repo git, binaires)
- Données : `/srv/anemone/` (db, certs, shares, smb, backups)
- Base de données : `/srv/anemone/db/anemone.db`
- Binaires système : `/usr/local/bin/` (anemone, anemone-dfree, anemone-smbgen, anemone-migrate, anemone-decrypt)
- Service : `systemd` (démarrage automatique)

### 📦 Liens utiles

- **Quickstart** : `QUICKSTART.md`
- **Readme principal** : `README.md`
- **Audit fichiers** : `CHECKFILES.md`

---

## 📋 Sessions archivées

- **Sessions 1-7** : Voir `SESSION_STATE_ARCHIVE.md`
- **Sessions 8-11** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`
- **Sessions 12-16** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md`
- **Sessions 17-19** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md`

---

## 🔧 Session 13 - 10 Novembre 2025 - Fréquence de synchronisation par pair

### Résumé

**Objectif** : Permettre de configurer une fréquence de synchronisation indépendante pour chaque pair.

**Architecture implémentée** :
- **Avant** : Configuration globale → tous les pairs synchronisés en même temps
- **Après** : Configuration individuelle par pair → chaque pair a sa propre fréquence

**Fréquences supportées** :
- **Interval** : Synchronisation régulière (30 min, 1h, 2h, 6h)
- **Daily** : Quotidienne à une heure fixe
- **Weekly** : Hebdomadaire un jour spécifique
- **Monthly** : Mensuelle un jour spécifique

**Statut** : 🟢 COMPLÈTE

---

## 🔧 Session 17 - 15 Novembre 2025 - Re-chiffrement clés utilisateur

### Résumé

**Problème** : Après restauration serveur, impossible de restaurer les fichiers (nouvelle master key).

**Solution** : Re-chiffrement automatique des clés utilisateur lors de la restauration.

**Outil créé** : `cmd/anemone-reencrypt-key/main.go`

**Statut** : 🟢 COMPLÈTE

---

## 🔧 Session 18 - 15-16 Novembre 2025 - Interface admin restauration

### Résumé

**Objectif** : Interface admin sécurisée pour restaurer les fichiers de tous les utilisateurs après disaster recovery.

**Solution** :
- `restore_server.sh` désactive automatiquement tous les pairs
- Interface admin `/admin/restore-users` pour restauration contrôlée
- Ownership automatique des fichiers restaurés

**Statut** : 🟢 COMPLÈTE (7 files, 280 KB, 0 errors)

---

## 🔧 Session 19 - 17 Novembre 2025 - Outil décryptage manuel

### Résumé

**Objectif** : Permettre la récupération des fichiers sans serveur (disaster recovery ultime).

**Solution** :
- CLI `anemone-decrypt` autonome
- Décryptage avec clé utilisateur uniquement
- Mode récursif, batch processing

**Tests** : 3 fichiers réels depuis FR2 (100% succès)

**Statut** : 🟢 COMPLÈTE

**Commits** :
```
e255d4d - feat: Add anemone-decrypt CLI tool (Session 19)
a93ab1a - fix: Correct admin dashboard stats and add backup deletion
```

---

## 🔧 Session 20 - 17 Novembre 2025 - Audit du code et nettoyage

**Date** : 2025-11-17
**Objectif** : Auditer tous les fichiers du projet pour identifier le code mort et les fichiers obsolètes
**Priorité** : 🟡 IMPORTANT → 🔄 EN COURS

### 🎯 Contexte

Après 19 sessions et de nombreuses modifications, nécessité de :
- Vérifier que tous les fichiers sont utilisés
- Identifier le code mort
- Nettoyer les vestiges des anciennes versions
- Préparer l'audit de sécurité

### ✅ Système mis en place

**1. CHECKFILES.md**
- Fichier de tracking pour l'audit
- Statuts par fichier : ✅ OK, 🗑️ MOVED, ❌ DELETE, 🔄 IN_PROGRESS
- Date de vérification et notes pour chaque fichier
- Statistiques de progression

**2. Répertoire _audit_temp/**
- Stockage temporaire des fichiers suspects
- Permet validation avant suppression définitive
- Structure : `cmd/`, `binaries/`, `web/templates/`, `internal/`
- Documentation dans `_audit_temp/README.md`

### 🔍 Audit réalisé

**Commandes CLI (9/9 complété)** ✅
- ✅ **8 outils essentiels validés** :
  - `cmd/anemone/main.go` - Serveur principal
  - `cmd/anemone-decrypt/main.go` - Décryptage manuel (Session 19)
  - `cmd/anemone-decrypt-password/main.go` - Déchiffrement mdp SMB (restore)
  - `cmd/anemone-dfree/main.go` - Quotas Samba
  - `cmd/anemone-migrate/main.go` - Migration Btrfs
  - `cmd/anemone-reencrypt-key/main.go` - Re-chiffrement clés (Session 17)
  - `cmd/anemone-restore-decrypt/main.go` - Déchiffrement backups (restore)
  - `cmd/anemone-smbgen/main.go` - Génération config Samba

- 🗑️ **1 fichier test déplacé** :
  - `cmd/test-manifest/main.go` → Programme de démo système manifest
  - Binaire `test-manifest` → Non utilisé en production

**Templates web (1/28 en cours)**
- 🗑️ **1 template obsolète déplacé** :
  - `web/templates/base.html` → Jamais référencé, vestige ancien

**Packages internes (1/40 en cours)**
- ✅ **1 package vérifié** :
  - `internal/syncconfig/syncconfig.go` → Utilisé par router.go (ligne 3058)

### 🗑️ Fichiers obsolètes identifiés

**Répertoire _old/** ⚠️
- **Taille** : 78 MB
- **Fichiers** : 2675 fichiers
- **Contenu** : Ancien système Python/Docker, scripts Restic/Wireguard, ancienne doc
- **Statut** : Aucune référence dans le code actif
- **Recommandation** : Suppression complète

**Fichiers déplacés dans _audit_temp/** (3 fichiers)
- `cmd/test-manifest/` - Programme de test
- `binaries/test-manifest` - Binaire compilé
- `web/templates/base.html` - Template non utilisé

### ✅ Vérification

- ✅ Compilation réussie après nettoyage
- ✅ Aucune régression introduite
- ✅ Tous les outils essentiels identifiés et documentés

### 📝 Commits

```
6ce431f - audit: Start code audit and move unused files to _audit_temp
```

**Détails** :
- Création `CHECKFILES.md` pour tracking audit
- Création `_audit_temp/` pour stockage temporaire
- Déplacement 3 fichiers obsolètes
- Documentation du répertoire `_old/` (78 MB à supprimer)

### 🔄 En attente

**Prochaines vérifications** :
- Packages internes : 39/40 restants
- Templates web : 27/28 restants
- Scripts : 5 fichiers
- Documentation : 3 fichiers

**Session suivante** : Continuer l'audit fichier par fichier avec `CHECKFILES.md`

**État session 20** : 🔄 **EN COURS - Audit démarré, premiers résultats positifs**

---

## 📝 Prochaines étapes (Roadmap)

### 🎯 Priorité 1 - Court terme

**Session 20 : Audit du code** 🔄 EN COURS
- ✅ CHECKFILES.md créé
- ✅ Commandes CLI auditées (9/9)
- 🔄 Packages internes (1/40)
- 🔄 Templates web (1/28)
- ⚠️ Répertoire _old/ identifié (78 MB à supprimer)

**Session 21 : Audit de sécurité complet** 🔒
- **Audit des permissions fichiers**
  - Vérifier permissions `/srv/anemone/` (600/700)
  - Vérifier ownership des fichiers sensibles
  - Vérifier permissions base de données
  - Vérifier permissions certificats TLS
- **Audit des clés de chiffrement**
  - Vérifier que la master key est uniquement en DB
  - Vérifier le chiffrement des clés utilisateurs
  - Vérifier l'absence de clés en clair sur le disque
- **Audit des endpoints API**
  - Vérifier l'authentification sur tous les endpoints
  - Tester les tentatives d'accès non autorisées
  - Vérifier la protection CSRF
  - Tester les injections SQL
  - Tester path traversal

### ⚙️ Priorité 2 - Améliorations

1. **Logs et audit trail** 📋
   - Table `audit_log` en base de données
   - Enregistrement actions importantes
   - Interface admin pour consulter les logs

2. **Vérification d'intégrité des backups** ✅
   - Commande `anemone-verify` pour vérification manuelle
   - Vérification checksums depuis manifests

3. **Rate limiting anti-bruteforce** 🛡️
   - Protection sur `/login` et `/api/sync/*`
   - Bannissement temporaire après X tentatives échouées

4. **Statistiques détaillées de synchronisation** 📊
   - Graphiques d'utilisation (espace, fichiers, bande passante)
   - Historique des syncs sur 30 jours

### 🚀 Priorité 3 - Évolutions futures

1. **Guide utilisateur complet** 📚
2. **Système de notifications** 📧
3. **Multi-peer redundancy** (2-of-3, 3-of-5)

---

**Dernière mise à jour** : 2025-11-17 (Session 20)
