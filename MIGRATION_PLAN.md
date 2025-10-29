# Plan de Migration vers /srv/anemone

**Date** : 2025-10-29
**Problème** : `/home/franck` (permissions 700) empêche Samba d'accéder aux partages
**Solution** : Migration complète vers `/srv/anemone` (standard FHS)

---

## 🚀 Étapes de migration (30 min max)

### 1️⃣ Préparation
```bash
# Arrêter le serveur
killall anemone

# Créer /srv/anemone
sudo mkdir -p /srv/anemone
sudo chown franck:franck /srv/anemone
```

### 2️⃣ Migration données
```bash
# Déplacer tout
mv ~/anemone/data/* /srv/anemone/

# Vérifier contenu
ls -la /srv/anemone/
# Attendu : db/ shares/ certs/ smb/
```

### 3️⃣ Permissions
```bash
# Permissions de base
sudo chmod 755 /srv/anemone

# Permissions partages utilisateurs
sudo chown -R test:test /srv/anemone/shares/test/
sudo chmod 755 /srv/anemone/shares/test/
```

### 4️⃣ Vérifier config Samba
```bash
# Config sera regénérée automatiquement
# Vérifier que chemins DB sont corrects
sqlite3 /srv/anemone/db/anemone.db "SELECT id, name, path FROM shares;"
```

### 5️⃣ Redémarrer serveur
```bash
cd ~/anemone
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

### 6️⃣ Vérifications
```bash
# Service Samba OK ?
systemctl status smb

# Config Samba OK ?
sudo testparm -s | grep -A 5 backup_test

# Permissions OK ?
namei -l /srv/anemone/shares/test/backup
```

### 7️⃣ Test Windows
- Connecter à `\\192.168.83.132\backup_test`
- User : `test`
- Créer un fichier test
- ✅ Si OK → Migration réussie !

---

## ✅ Checklist post-migration

- [ ] Web admin accessible (https://192.168.83.132:8443)
- [ ] User test peut se connecter au web
- [ ] Partages SMB visibles depuis Windows
- [ ] Lecture fichiers OK depuis Windows
- [ ] Écriture fichiers OK depuis Windows
- [ ] Pas d'erreurs dans `journalctl -u smb`
- [ ] Mise à jour README.md
- [ ] Mise à jour QUICKSTART.md
- [ ] Commit "chore: Migrate data to /srv/anemone"

---

## 🐛 Troubleshooting

### Erreur "Permission denied" Samba
```bash
# Vérifier permissions chemin complet
namei -l /srv/anemone/shares/test/backup

# Tous les répertoires doivent être au moins 755
# Les répertoires utilisateur doivent être owned par l'user
```

### Erreur "Path not found"
```bash
# Vérifier chemins dans DB
sqlite3 /srv/anemone/db/anemone.db "SELECT * FROM shares;"

# Si chemins incorrects, update manuel :
sqlite3 /srv/anemone/db/anemone.db
UPDATE shares SET path = '/srv/anemone/shares/test/backup' WHERE name = 'backup_test';
UPDATE shares SET path = '/srv/anemone/shares/test/data' WHERE name = 'data_test';
.quit
```

### Service Samba ne démarre pas
```bash
# Logs détaillés
journalctl -u smb -n 100 --no-pager

# Test config
sudo testparm -s

# Reload
sudo systemctl reload smb
```

---

## 📝 Notes importantes

- **Aucun changement de code Go requis** : `ANEMONE_DATA_DIR` est déjà utilisé partout
- **Sudoers reste valide** : `/etc/sudoers.d/anemone-smb` ne contient pas de chemins hardcodés
- **Certificats TLS** : Seront dans `/srv/anemone/certs/` automatiquement
- **Base de données** : `/srv/anemone/db/anemone.db`

---

## 🔄 Rollback (si problème)

```bash
# Arrêter serveur
killall anemone

# Remettre données dans ~/anemone/data
mv /srv/anemone/* ~/anemone/data/

# Redémarrer avec ancien chemin
cd ~/anemone
ANEMONE_DATA_DIR=./data ./anemone
```

---

**Temps estimé** : 15-30 minutes
**Risque** : Faible (rollback simple si besoin)
**Impact** : Résout définitivement le problème SMB
