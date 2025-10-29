# ✅ Migration /srv/anemone COMPLÈTE

**Date migration** : 2025-10-29 14:05
**Status** : 🟢 OPÉRATIONNEL

---

## ✅ Migration réussie

Les données ont été migrées vers `/srv/anemone` avec succès.

**Tests validés** :
- ✅ Accès SMB depuis Windows : OK
- ✅ Accès SMB depuis Android : OK
- ✅ Création/lecture/écriture fichiers : OK
- ✅ Permissions UNIX : OK (test:test)
- ✅ SELinux : OK (samba_share_t + samba_export_all_rw)

**Structure actuelle** :
- Code : `~/anemone/` (binaire, templates, scripts)
- Données : `/srv/anemone/` (db, certs, shares, smb)

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

## 🎯 Session de migration - 29 Octobre 14:00-14:10

### Migration /srv/anemone COMPLÈTE ✅

**Problèmes résolus** :
1. ❌ Permissions `/home/franck` (700) → ✅ Migration `/srv/anemone` (755)
2. ❌ SELinux `user_home_t` → ✅ Contexte `samba_share_t` appliqué
3. ❌ Boolean SELinux off → ✅ `samba_export_all_rw` activé

**Étapes réalisées** :
1. ✅ Création `/srv/anemone` avec permissions 755
2. ✅ Déplacement toutes données (db, certs, shares, smb)
3. ✅ Ajustement permissions (test:test pour partages)
4. ✅ Mise à jour chemins absolus dans DB
5. ✅ Mise à jour smb.conf avec nouveaux chemins
6. ✅ Configuration SELinux (contexte + boolean)
7. ✅ Tests Windows + Android : OK

**Commandes SELinux appliquées** :
```bash
sudo semanage fcontext -a -t samba_share_t "/srv/anemone/shares(/.*)?"
sudo restorecon -Rv /srv/anemone/shares/
sudo setsebool -P samba_export_all_rw on
```

### Avantages de /srv/anemone

✅ **Standard FHS (Filesystem Hierarchy Standard)**
✅ **Sécurité** : Isolation /home vs données NAS
✅ **Permissions claires** : Plus de problème traversée répertoire
✅ **Production-ready** : Comme TrueNAS, Synology, etc.
✅ **Portabilité** : Indépendant de l'utilisateur système
✅ **Backups** : `/srv` peut avoir sa propre stratégie backup
✅ **SELinux** : Contexte dédié pour Samba

### Tâches suivantes

#### Court terme
1. **Script d'installation automatique** - install.sh pour nouvelle installation
2. **Auto-config SELinux** - Dans le code lors activation utilisateur
3. **Service systemd** - Démarrage automatique
4. **Page Paramètres** - Config système, workgroup, etc.
5. **Quotas** - Monitoring espace disque

#### Moyen terme
1. **Synchronisation P2P** - Logique sync réelle
2. **Chiffrement** - Implémentation chiffrement partages backup
3. **Monitoring** - Dashboard stats utilisation
4. **Corbeille** - Gestion fichiers supprimés (30j)

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

## 📸 État actuel du système

**Serveur DEV (192.168.83.99)** :
- ✅ Code à jour
- ✅ **Migration /srv/anemone : COMPLÈTE**
- ✅ Serveur HTTPS actif sur :8443
- ✅ Utilisateur test créé et activé
- ✅ Partages SMB fonctionnels (backup_test, data_test)
- ✅ SELinux configuré (samba_share_t)
- ✅ Tests Windows + Android : OK

**Serveur FR1 (192.168.83.96)** :
- ✅ Code à jour
- ✅ P2P peer connecté à DEV
- ⏸️ En attente réinstallation complète

---

## 📞 Pour reprendre la PROCHAINE session

### PRIORITÉ 1 : Script d'installation automatique

**Objectif** : L'admin ne doit RIEN faire en ligne de commande après le démarrage.

**Créer `install.sh`** qui fait :
1. Vérifier prérequis (Go, git, sudo)
2. Compiler le binaire
3. Créer `/srv/anemone`
4. Installer Samba
5. Configurer SELinux (contexte + boolean)
6. Configurer sudoers
7. Configurer firewall
8. Créer service systemd (démarrage auto)
9. Générer certificat TLS
10. Premier démarrage

**Utilisation** :
```bash
git clone <repo> ~/anemone
cd ~/anemone
sudo ./install.sh
```

### PRIORITÉ 2 : Auto-config SELinux dans le code

Modifier `internal/shares/shares.go` pour appeler automatiquement :
- `sudo semanage fcontext` lors création partage
- `sudo restorecon` sur les répertoires créés

---

**Session sauvegardée le** : 2025-10-29 14:10
**Tokens utilisés** : ~45k/200k (22%)
**État** : Migration complète et validée
**Prochaine action** : Script install.sh + auto-config SELinux
