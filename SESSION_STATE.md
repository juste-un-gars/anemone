# 🪸 Anemone - État du Projet

**Dernière session** : 2025-11-19 (Session 24 - Adaptation restauration après séparation serveurs)
**Prochaine session** : Session 25 - Tests disaster recovery complets
**Status** : 🟢 OPÉRATIONNELLE - Système de restauration adapté et sécurisé

> **Note** : Les sessions 1-19 ont été archivées (voir fichiers `SESSION_STATE_ARCHIVE*.md`)
> **Note** : Les détails techniques des sessions 20-24 sont dans `SESSION_STATE_ARCHIVE_SESSIONS_20_24.md`

---

## 🎯 État actuel

### ✅ Fonctionnalités complètes et testées

1. **Configuration initiale (Setup)**
   - Choix langue (FR/EN)
   - Création premier admin
   - **Génération automatique clé de chiffrement** (256 bits)
   - **Génération automatique mot de passe sync P2P** (192 bits) - Session 21

2. **Authentification & Sécurité**
   - Login/logout multi-utilisateurs
   - Sessions sécurisées (SameSite=Strict, HttpOnly, Secure)
   - HTTPS avec certificat auto-signé
   - Réinitialisation mot de passe par admin
   - **Validation stricte username** (prévention injection commandes) - Session 21
   - **Headers HTTP sécurité** (HSTS, CSP, X-Frame-Options) - Session 21
   - **Protection CSRF maximale** (SameSite=Strict) - Session 21

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
   - **Authentification P2P obligatoire** (mot de passe généré au setup) - Session 21

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

14. **Audit du code** (Session 20)
    - Fichier de tracking `CHECKFILES.md` avec statuts par fichier
    - Répertoire `_audit_temp/` pour fichiers suspects
    - **Commandes CLI** : 9/9 vérifiées (8 OK, 1 déplacé)
    - **Fichiers déplacés** : `cmd/test-manifest/`, `base.html`
    - **Nettoyage** : `_old/` archivé (78 MB, 2675 fichiers obsolètes)
    - **Résultat** : 96.5% code actif, très propre

15. **Sécurité renforcée** (Sessions 21-22)
    - **Validation username** : Regex stricte (prévention injection commandes)
    - **Headers HTTP** : HSTS, CSP, X-Frame-Options, X-Content-Type-Options
    - **Protection CSRF** : SameSite=Strict + Secure cookies
    - **Sync auth auto** : Mot de passe P2P généré automatiquement au setup (192 bits)
    - **bcrypt cost** : Augmenté de 10 à 12 (protection bruteforce renforcée)
    - **Score sécurité** : 10/10 (5/5 vulnérabilités corrigées) 🎉

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
- ✅ **Validation username** : OK (Session 21)
- ✅ **Headers HTTP sécurité** : OK (Session 21)
- ✅ **Protection CSRF** : OK (Session 21)
- ✅ **Sync password auto-généré** : OK (Session 21)

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
- **Audit sécurité** : `SECURITY_AUDIT.md`

---

## 📋 Sessions archivées

- **Sessions 1-7** : Voir `SESSION_STATE_ARCHIVE.md`
- **Sessions 8-11** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`
- **Sessions 12-16** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md`
- **Sessions 17-19** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md`
- **Sessions 20-24** : Voir `SESSION_STATE_ARCHIVE_SESSIONS_20_24.md`

---

## 📝 Sessions récentes (Résumé)

### 🔧 Session 20 - Audit du code (17 Nov 2025)
✅ **COMPLÉTÉ** - Code audit complet : 96.5% code actif, 3.5% obsolète archivé

### 🔒 Session 21 - Audit sécurité (17 Nov 2025)
✅ **COMPLÉTÉ** - 4/5 vulnérabilités corrigées (Score 9.5/10)
- Injection commandes username
- Headers HTTP sécurité
- Protection CSRF renforcée
- Sync password auto-généré

### 🔒 Session 22 - bcrypt cost (18 Nov 2025)
✅ **COMPLÉTÉ** - Score sécurité parfait 10/10
- bcrypt cost: 10 → 12

### 🐛 Session 23 - Correctifs bugs (18 Nov 2025)
✅ **COMPLÉTÉ** - 5 bugs critiques corrigés
- Bug critique: Collision backups multi-serveurs
- CSP bloquant CDN
- Répertoires invisibles corbeille
- Test P2P faux positif
- Permissions après restore

### ✅ Session 24 - Adaptation restauration (19 Nov 2025)
✅ **COMPLÉTÉ** - Système de restauration adapté à la nouvelle structure multi-serveurs
- Ajout paramètre `source_server` dans toutes les APIs de restauration
- Filtrage sécurisé : chaque serveur ne voit que ses propres backups
- Re-chiffrement password_encrypted avec nouvelle master key
- Désactivation auto-sync après disaster recovery
- Affichage nom serveur dans headers (identification visuelle)
- **7 commits** : 485eaee, 934e27c, ed62fcf, e3a1710, 1c49509, 9910126, 57e08b4

---

## 🧪 Session 25 - Tests disaster recovery complets

**Date** : À FAIRE
**Objectif** : Tester complètement le système de disaster recovery et la séparation multi-serveurs
**Statut** : 📋 **PLANIFIÉ**

### 🎯 Plan de test

#### Phase 1: Initial Setup (Verify Bug 5 fix)
```
FR1 (192.168.83.16) - Primary server
  └─ User: test / password: test
  └─ Create files: file1.txt, file2.txt

FR2 (192.168.83.37) - Primary server  
  └─ User: test / password: test
  └─ Create DIFFERENT files: fileA.txt, fileB.txt

FR3 (192.168.83.38) - Backup server for both
  └─ Add FR1 as peer, enable sync
  └─ Add FR2 as peer, enable sync
  └─ Force sync or wait

✅ Expected: FR3 should have:
   - /incoming/FR1/1_test/
   - /incoming/FR2/1_test/
```

#### Phase 2: Backup Visibility Test (Commit 934e27c)
```
On FR1:
  └─ Login as 'test'
  └─ Go to "Parcourir les backups"
  
✅ Expected: Only see backups "(from FR1)"
❌ Should NOT see: "(from FR2)"

Repeat on FR2 - should only see "(from FR2)"
```

#### Phase 3: Admin Filter Test (Commit 1c49509)
```
On FR1:
  └─ Login as admin
  └─ Go to "Restaurer tous les fichiers des utilisateurs"
  
✅ Expected: Only see backups "(from FR1)"
❌ Should NOT see: "(from FR2)"
```

#### Phase 4: Full Disaster Recovery (Main test)
```
FR4 (new clean server)
  └─ scp restore_server.sh to FR4
  └─ scp FR1 backup (.enc file) to FR4
  └─ Run: sudo bash restore_server.sh anemone_backup_XXX.enc "passphrase"
  
✅ Expected:
   - Script completes without errors
   - All users created (admin, test)
   - SMB users created with passwords
   - Database restored
```

#### Phase 5: Post-Restore Checks (All recent fixes)
```
On FR4 (after restore):
  └─ Login as admin
  └─ Verify restore warning page shows
  
  Check 1: Global Auto-Sync (NEW FIX - Commit 57e08b4)
    └─ Go to /admin/sync
    └─ ✅ "Activer la synchronisation automatique" checkbox should be UNCHECKED
  
  Check 2: Peer Sync Status
    └─ Go to /admin/peers
    └─ ✅ All peers should show "Désactivé" badge
  
  Check 3: Password Re-encryption (Commit ed62fcf)
    └─ Logout
    └─ Login as user 'test' with original password
    └─ ✅ Should work (password re-encrypted with new master key)
```

#### Phase 6: User File Restoration
```
On FR4 (as admin):
  └─ From restore warning or admin page
  └─ Click "Restaurer tous les fichiers des utilisateurs"
  └─ Select source: FR3 - FR1 (from FR1)
  └─ Launch restoration
  
✅ Expected:
   - Restoration completes successfully
   - Files appear in /home/test/anemone/
   - Files match original FR1 files (file1.txt, file2.txt)
   - Can access via SMB
```

#### Phase 7: Source Server Separation (Critical)
```
On FR4:
  └─ Go to peers, re-enable FR3
  └─ Go to /admin/sync, enable auto-sync
  └─ Create NEW file: file3.txt in test's share
  └─ Force sync to FR3

On FR3, check /data/incoming/:
  ✅ Should have 3 directories:
     - FR1/ (original files from FR1)
     - FR2/ (original files from FR2)  
     - FR4/ (new file3.txt from restored server)
  
  ❌ Should NOT mix FR4 files into FR1/ directory
```

#### Phase 8: Cross-Restoration Test (Bonus)
```
FR5 (new clean server)
  └─ Try to restore FR1 backup
  └─ Go to admin restore page
  └─ Try to restore files from FR3
  
✅ Expected: Only sees backups "(from FR1)"
❌ Should NOT see: "(from FR2)" or "(from FR4)"
```

### 📋 Checklist

- [ ] Phase 1: Initial setup (FR1, FR2, FR3)
- [ ] Phase 2: Backup visibility test
- [ ] Phase 3: Admin filter test
- [ ] Phase 4: Full disaster recovery (FR1 → FR4)
- [ ] Phase 5: Post-restore checks (sync, peers, passwords)
- [ ] Phase 6: User file restoration
- [ ] Phase 7: Source server separation validation
- [ ] Phase 8: Cross-restoration test (FR5)

### 🎯 Focus minimum

**Tests prioritaires** :
- Phase 4 + 5 (Full disaster recovery with new sync fix)
- Phase 7 (Verify source server separation still works)

---

## 📝 Prochaines étapes (Roadmap)

### 🎯 Priorité 1 - Court terme

**Session 25 : Tests disaster recovery complets** 🧪
- [ ] Exécuter le plan de test complet (8 phases)
- [ ] Documenter tous les résultats
- [ ] Corriger tout bug découvert
- [ ] Valider que le système est production-ready

### ⚙️ Priorité 2 - Améliorations futures

1. **Logs et audit trail** 📋
   - Table `audit_log` en base de données
   - Enregistrement actions importantes (login, création user, sync)
   - Interface admin pour consulter les logs

2. **Rate limiting anti-bruteforce** 🛡️
   - Protection sur `/login` et `/api/sync/*`
   - Bannissement temporaire après X tentatives échouées
   - Headers `X-RateLimit-*`

3. **Statistiques détaillées** 📊
   - Graphiques d'utilisation (espace, fichiers, bande passante)
   - Historique des syncs sur 30 jours
   - Export CSV/JSON

4. **Vérification intégrité backups** ✅
   - Commande `anemone-verify` pour vérification checksums
   - Vérification depuis manifests
   - Rapport d'intégrité

### 🚀 Priorité 3 - Évolutions futures

1. **Guide utilisateur complet** 📚
2. **Système de notifications** 📧 (email, webhook)
3. **Multi-peer redundancy** (2-of-3, 3-of-5)
4. **Support IPv6**
5. **Interface mobile (PWA)**

---

**Dernière mise à jour** : 2025-11-19 (Session 24 complétée - Session 25 planifiée)
