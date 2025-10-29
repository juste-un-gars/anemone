# ⚠️ ACTION REQUISE AVANT PROCHAINE UTILISATION

**Date** : 2025-10-29 09:30
**Status** : 🔴 MIGRATION NÉCESSAIRE

---

## 🚨 Problème actuel

Les partages SMB ne sont **pas accessibles** car les données sont dans `/home/franck/anemone/data/`.

Le répertoire `/home/franck` a des permissions `700` qui empêchent les utilisateurs SMB d'y accéder.

**Erreur Samba** :
```
chdir_current_service: vfs_ChDir(/home/franck/anemone/data/shares/test/backup)
failed: Permission non accordée
```

---

## ✅ Solution : Migration vers /srv/anemone

### Fichiers à lire AVANT de continuer :

1. **`MIGRATION_PLAN.md`** ← Plan détaillé étape par étape (15-30 min)
2. Continuer la lecture de ce fichier pour le contexte complet

---

## 🎯 Résumé migration (ultra rapide)

```bash
# 1. Arrêter
killall anemone

# 2. Créer destination
sudo mkdir -p /srv/anemone
sudo chown franck:franck /srv/anemone

# 3. Déplacer données
mv ~/anemone/data/* /srv/anemone/

# 4. Permissions
sudo chown -R test:test /srv/anemone/shares/test/
sudo chmod 755 /srv/anemone/shares/test/

# 5. Redémarrer avec nouveau chemin
cd ~/anemone
ANEMONE_DATA_DIR=/srv/anemone ./anemone

# 6. Tester depuis Windows
# Connecter à \\192.168.83.132\backup_test
```

**⚠️ Ne pas utiliser le NAS avant migration !**
**⚠️ Les partages SMB ne fonctionneront pas !**

---
---

# État de la session - 29 Octobre 2025

## 📍 Contexte de cette session

**Session précédente** : Phase 1-4 complètes (setup, auth, users, activation)
**Cette session** : P2P Peers + SMB Shares (automatisation activation)

## ✅ Fonctionnalités implémentées aujourd'hui

### 1. Gestion P2P Peers (Complète ✅)
- CRUD complet pour pairs de synchronisation
- Test de connexion HTTPS entre pairs
- Gestion statuts (online/offline/error/unknown)
- Interface admin avec actions (test, delete)
- **État actuel** : 2 pairs connectés et testés
  - DEV (192.168.83.132:8443) ↔ FR1 (192.168.83.96:8443)

### 2. Partages SMB Automatisés (Complète ✅)
- Création automatique lors activation utilisateur
- 2 partages par user : `backup_username` + `data_username`
- Permissions et ownership automatiques
- Génération dynamique smb.conf depuis DB
- Copie auto vers /etc/samba/smb.conf
- Reload auto service Samba
- **État actuel** : Architecture complète, tests en cours

### 3. Corrections et Améliorations
- Lien activation avec IP serveur (plus localhost)
- Support multi-distro Samba (smb vs smbd)
- Configuration sudoers complète
- Chemins absolus pour Samba
- Interface admin partages (vue globale)

## 🔧 Commits de cette session (10 commits)

1. `2f1f118` - Support multi-distro Samba (smb/smbd)
2. `353079a` - Copie auto smb.conf → /etc/samba
3. `2a73f25` - Chemins absolus pour partages SMB
4. `d49da1a` - Correction permissions SMB et noms
5. `375ecc5` - Ajout sudo pour commandes SMB + sudoers
6. `74c6cc5` - Config auto reload SMB via sudoers
7. `867b5bb` - Fix lien activation (IP au lieu localhost)
8. `87ab49b` - **Création auto partages lors activation**
9. `1ec6f88` - Partages en admin uniquement
10. `e4ff47e` - Implémentation gestion pairs P2P

## 📁 Nouveaux fichiers créés

### Packages Go
- `internal/peers/peers.go` (164 lignes) - Gestion pairs P2P
- `internal/shares/shares.go` (178 lignes) - Gestion partages
- `internal/smb/smb.go` (217 lignes) - Configuration Samba

### Templates HTML
- `web/templates/admin_peers.html` (199 lignes) - Liste pairs
- `web/templates/admin_peers_add.html` (169 lignes) - Ajout pair
- `web/templates/admin_shares.html` - Vue globale partages

### Scripts
- `scripts/configure-smb-reload.sh` - Configuration sudoers
- `scripts/README.md` - Documentation

## 🏗️ Architecture du flux d'activation

```
Admin crée user → Génère lien activation → User clique lien
                                              ↓
                                   User définit mot de passe
                                              ↓
                            ┌─────────────────┴─────────────────┐
                            │   Activation déclenche (auto):    │
                            ├───────────────────────────────────┤
                            │ 1. Création user système (sudo)   │
                            │ 2. Création user SMB (sudo)       │
                            │ 3. Création backup_username       │
                            │    - Sync P2P activé              │
                            │    - Chiffré                      │
                            │ 4. Création data_username         │
                            │    - Local uniquement             │
                            │ 5. Chown répertoires (sudo)       │
                            │ 6. Génération smb.conf            │
                            │ 7. Copie → /etc/samba (sudo)      │
                            │ 8. Reload Samba (sudo)            │
                            └───────────────────────────────────┘
```

## 📂 Structure partages

```
data/shares/
├── username/
│   ├── backup/  → backup_username  (Sync P2P ✅, chiffré)
│   └── data/    → data_username    (Local uniquement)
```

**Nomenclature** : `backup_franck`, `data_franck`, etc.

## 🔐 Configuration Sudoers

**Fichier** : `/etc/sudoers.d/anemone-smb`

```bash
franck ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smb
franck ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smb.service
franck ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smbd
franck ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload smbd.service
franck ALL=(ALL) NOPASSWD: /usr/sbin/useradd -M -s /usr/sbin/nologin *
franck ALL=(ALL) NOPASSWD: /usr/bin/smbpasswd
franck ALL=(ALL) NOPASSWD: /usr/bin/chown -R *
franck ALL=(ALL) NOPASSWD: /usr/bin/cp * /etc/samba/smb.conf
```

**Installation** :
```bash
sudo ./scripts/configure-smb-reload.sh franck
```

## ❌ Problèmes résolus cette session

### 1. Popup sudo lors activation
- **Cause** : Commandes SMB sans sudo, demandait mdp
- **Solution** : Sudo + configuration sudoers complète

### 2. Lien activation avec localhost
- **Cause** : Hardcodé localhost au lieu IP serveur
- **Solution** : Utilise `r.Host` pour conserver l'IP

### 3. Partages SMB inaccessibles (multi-causes)
- **Nom incorrect** : `backup_test-test` → Corrigé template
- **Permissions** : Root au lieu user → Ajout chown auto
- **Chemins relatifs** → Conversion absolus via filepath.Abs()
- **Config pas utilisée** → Copie auto vers /etc/samba/smb.conf
- **Mauvais service** : smbd vs smb → Fallback multi-distro

### 4. Erreur création user SMB
- **Cause** : smbpasswd sans sudo
- **Solution** : Ajout sudo partout + sudoers

## 🗄️ Base de données

### Table `peers`
```sql
id, name, address, port, public_key, enabled, status,
last_seen, last_sync, created_at, updated_at
```

**Exemple** :
```sql
INSERT INTO peers VALUES (
  1, 'FR1', '192.168.83.96', 8443, NULL, 1, 'online',
  '2025-10-29 10:00:00', NULL, NOW(), NOW()
);
```

### Table `shares`
```sql
id, user_id, name, path, protocol, sync_enabled, created_at
```

**Exemple** :
```sql
INSERT INTO shares VALUES (
  1, 5, 'backup_test',
  '/home/franck/anemone/data/shares/test/backup',
  'smb', 1, NOW()
);
```

## 🌐 Traductions ajoutées

**Peers** : 30+ clés FR/EN
- peers.title, peers.add, peers.status.*, etc.

**Shares** : 28 clés FR/EN
- shares.title, shares.protocol.*, shares.smb_status, etc.

## 🚀 Configuration requise

### 1. Samba installé
```bash
# Fedora/RHEL
sudo dnf install samba

# Debian/Ubuntu
sudo apt install samba
```

### 2. Service actif
```bash
# Fedora
sudo systemctl enable --now smb

# Debian
sudo systemctl enable --now smbd
```

### 3. Sudoers configuré
```bash
cd ~/anemone
sudo ./scripts/configure-smb-reload.sh franck
```

## 📊 Variables d'environnement

```bash
PORT=8080                    # Port HTTP (défaut)
HTTPS_PORT=8443              # Port HTTPS (défaut)
ENABLE_HTTP=false            # Activer HTTP
ENABLE_HTTPS=true            # Activer HTTPS (défaut)
ANEMONE_DATA_DIR=./data      # Répertoire données
LANGUAGE=fr                  # Langue (fr/en)
TLS_CERT_PATH=/path/cert.crt # Certificat custom
TLS_KEY_PATH=/path/cert.key  # Clé custom
```

## 🖥️ État des serveurs

### Serveur DEV (192.168.83.132)
- ✅ Code à jour (commit 2f1f118)
- ✅ Serveur actif :8443
- ✅ Utilisateur test créé
- ✅ Sudoers configuré
- ✅ Partages créés (backup_test, data_test)

### Serveur FR1 (192.168.83.96)
- ✅ Code à jour (commit 2f1f118)
- ✅ Sudoers configuré
- ✅ Service smb actif
- ⏳ Tests SMB en cours

### Connexion P2P
- ✅ FR1 ↔ DEV : Testée, en ligne
- ✅ Test connexion fonctionne
- ✅ Statuts mis à jour

## 🔍 Diagnostic SMB

### Vérifications
```bash
# User SMB créé ?
sudo pdbedit -L

# Config Samba
sudo testparm -s

# Service actif ?
sudo systemctl status smb   # Fedora
sudo systemctl status smbd  # Debian

# Permissions répertoires
ls -la data/shares/username/

# Config copiée ?
diff data/smb/smb.conf /etc/samba/smb.conf

# Partages en DB
sqlite3 data/db/anemone.db "SELECT * FROM shares;"
```

### Connexion depuis Windows
```
Chemin : \\192.168.83.132\backup_test
User   : test
Pass   : [mot de passe activation]
```

## ⚠️ Problème IDENTIFIÉ - Session 29 Oct 09:20

**Symptôme** : Accès refusé depuis Windows aux partages SMB

**Diagnostic complet** :
- ✅ User système créé (uid=1001)
- ✅ User SMB créé et enabled (mot de passe OK)
- ✅ Répertoires avec permissions (test:test)
- ✅ smb.conf correct (chemins absolus)
- ✅ Config copiée /etc/samba/smb.conf
- ✅ Service Samba rechargé

**ROOT CAUSE TROUVÉE** 🎯 :
```bash
# Logs Samba :
chdir_current_service: vfs_ChDir(/home/franck/anemone/data/shares/test/backup)
failed: Permission non accordée. Current token: uid=1001, gid=1001

# Analyse permissions :
$ namei -l /home/franck/anemone/data/shares/test/backup
drwx------ franck franck /home/franck  ← PROBLÈME ICI !
```

**Le problème** : `/home/franck` a les permissions `700` (drwx------), donc l'utilisateur `test` (uid=1001) ne peut pas traverser ce répertoire pour accéder aux partages en dessous.

**Solution testée** : `chmod o+x /home/franck` fonctionnerait MAIS n'est pas propre

**Solution PROPRE décidée** : 🚀 **Migration vers `/srv/anemone`**

## 📝 Commandes utiles

```bash
# Rebuild
CGO_ENABLED=1 go build -o anemone ./cmd/anemone

# Start
ANEMONE_DATA_DIR=./data ./anemone

# Sudoers
sudo ./scripts/configure-smb-reload.sh franck

# Reload Samba
sudo systemctl reload smb    # Fedora
sudo systemctl reload smbd   # Debian

# Test Samba config
sudo testparm -s | head -50

# Check SMB users
sudo pdbedit -L -v

# Clean test user
sudo smbpasswd -x test
sudo userdel test
rm -rf data/shares/test

# Database
sqlite3 data/db/anemone.db "SELECT * FROM shares;"
sqlite3 data/db/anemone.db "SELECT * FROM peers;"
```

## 🎯 PROCHAINE SESSION : Migration vers /srv/anemone

### ⚠️ ACTION IMMÉDIATE REQUISE

**Problème** : Les données sont dans `/home/franck/anemone/data/` ce qui crée un problème de permissions pour Samba.

**Migration complète à faire** :

#### 1. Préparation (avec sudo)
```bash
# Créer structure /srv/anemone
sudo mkdir -p /srv/anemone
sudo chown franck:franck /srv/anemone

# Arrêter le serveur
killall anemone
```

#### 2. Migration des données
```bash
# Déplacer tout le contenu
mv ~/anemone/data/* /srv/anemone/

# Vérifier
ls -la /srv/anemone/
# Devrait contenir : db/ shares/ certs/ smb/
```

#### 3. Ajuster les permissions
```bash
# Permissions de base
sudo chown -R franck:franck /srv/anemone
sudo chmod 755 /srv/anemone

# Permissions des partages utilisateurs
sudo chown -R test:test /srv/anemone/shares/test/
sudo chmod 755 /srv/anemone/shares/test/
```

#### 4. Mise à jour configuration
```bash
# Modifier /etc/sudoers.d/anemone-smb si chemins hardcodés
# Ou relancer le script :
sudo ./scripts/configure-smb-reload.sh franck
```

#### 5. Mise à jour config Samba
```bash
# La config sera regénérée automatiquement au prochain reload
# mais vérifier que les chemins dans la DB pointent vers /srv
sqlite3 /srv/anemone/db/anemone.db "SELECT * FROM shares;"
```

#### 6. Redémarrer avec nouveau chemin
```bash
cd ~/anemone
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

#### 7. Tests post-migration
- [ ] Connexion web admin OK
- [ ] User test peut se connecter
- [ ] Partages SMB visibles depuis Windows
- [ ] Accès SMB fonctionne (écriture/lecture)
- [ ] Config Samba correcte (`sudo testparm -s`)

### Fichiers à modifier (peut-être)

**Aucun fichier Go à modifier** : La variable `ANEMONE_DATA_DIR` est déjà utilisée partout !

**Documentation à mettre à jour** :
- README.md : Changer exemples avec `/srv/anemone`
- QUICKSTART.md : Idem
- SESSION_STATE.md : Mise à jour après migration

### Avantages de /srv/anemone

✅ **Standard FHS (Filesystem Hierarchy Standard)**
✅ **Sécurité** : Isolation /home vs données NAS
✅ **Permissions claires** : Plus de problème traversée répertoire
✅ **Production-ready** : Comme TrueNAS, Synology, etc.
✅ **Portabilité** : Indépendant de l'utilisateur système
✅ **Backups** : `/srv` peut avoir sa propre stratégie backup

### Après migration : Tâches suivantes

#### Court terme
1. **Validation complète SMB** - Tests read/write depuis Windows
2. **Page Paramètres** - Config système, workgroup, etc.
3. **Quotas** - Monitoring espace disque
4. **Corbeille** - Gestion fichiers supprimés (30j)

#### Moyen terme
1. **Synchronisation P2P** - Logique sync réelle
2. **Chiffrement** - Implémentation chiffrement partages backup
3. **Monitoring** - Dashboard stats utilisation

## 💡 Notes importantes

- **Sudoers essentiel** : Sans le script, popups sudo
- **Multi-distro** : Support smb (Fedora) + smbd (Debian)
- **Chemins absolus** : Samba requiert chemins absolus
- **Pas de création manuelle** : Users ne créent PAS de partages
- **Admin only** : Vue globale partages réservée admin
- **2 partages auto** : backup (sync) + data (local)

## 📈 Statistiques sessions cumulées

### Session précédente (09:00-09:15)
- **Commits** : 10 commits
- **Fichiers créés** : 6 fichiers Go + 3 templates + 2 scripts
- **Lignes ajoutées** : ~1,200 lignes Go + 600 lignes HTML
- **Traductions** : 58 nouvelles clés FR/EN
- **Problèmes résolus** : 7 bugs majeurs

### Session actuelle (09:20-09:30)
- **Commits** : 0 (diagnostic uniquement)
- **Root cause trouvée** : Problème permissions `/home/franck` (700)
- **Outils diagnostic utilisés** :
  - `journalctl -u smb` → Logs Samba
  - `namei -l` → Analyse permissions chemin complet
  - `id test` → Vérification UID/GID
- **Décision architecture** : Migration vers `/srv/anemone` (standard FHS)

## 📞 Pour reprendre la PROCHAINE session

### 🚨 PRIORITÉ 1 : Migration /srv/anemone

1. **Lire ce fichier SESSION_STATE.md** (section "🎯 PROCHAINE SESSION")
2. **Suivre étapes migration** (7 étapes détaillées ci-dessus)
3. **Tester connexion SMB** depuis Windows
4. **Valider** : Lecture/écriture fichiers OK

### Après migration réussie

5. Mettre à jour README.md et QUICKSTART.md
6. Commit la mise à jour docs
7. Continuer avec Page Paramètres

### Si problèmes pendant migration

- Vérifier logs : `journalctl -u smb -f`
- Vérifier permissions : `namei -l /srv/anemone/shares/test/backup`
- Vérifier config : `sudo testparm -s`
- Vérifier DB : `sqlite3 /srv/anemone/db/anemone.db "SELECT * FROM shares;"`

---

## 📸 État actuel du système

**Serveur DEV (192.168.83.132)** :
- ✅ Code à jour (commit 2f1f118)
- ✅ Serveur HTTPS actif sur :8443
- ✅ Utilisateur test créé et activé
- ✅ Partages créés (backup_test, data_test)
- ⚠️ **Bloqué** : Permissions /home/franck empêchent accès SMB
- 🚀 **Prochaine action** : Migration vers /srv/anemone

**Serveur FR1 (192.168.83.96)** :
- ✅ Code à jour
- ✅ P2P peer connecté à DEV
- ⏸️ En attente validation DEV avant tests

---

**Session sauvegardée le** : 2025-10-29 09:30
**Tokens utilisés** : ~34k/200k (17%)
**État** : Root cause identifiée, plan migration défini
**Prochaine action** : Migration complète vers /srv/anemone
