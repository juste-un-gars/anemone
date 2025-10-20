# Utilisation de partages réseau externes avec Anemone

Si vous avez choisi de **ne PAS utiliser les partages intégrés** (Samba/WebDAV) lors de l'installation, vous devez monter vos propres partages réseau sur les répertoires d'Anemone.

## 📂 Répertoires à monter

Anemone utilise 3 répertoires principaux :

| Répertoire | Usage | Requis |
|-----------|-------|---------|
| `./data` | Données locales uniquement (non sauvegardées) | Optionnel |
| `./backup` | Données à sauvegarder chez les pairs | **REQUIS** |
| `./backups` | Backups reçus des pairs | **REQUIS** |

## 🔧 Options de montage

### Option 1 : Partage SMB/CIFS (Windows, NAS)

```bash
# Installer cifs-utils si nécessaire
sudo apt-get install cifs-utils  # Debian/Ubuntu
sudo dnf install cifs-utils       # Fedora

# Créer le point de montage
sudo mkdir -p /mnt/nas-backup

# Monter le partage
sudo mount -t cifs //192.168.1.100/backup /mnt/nas-backup \
    -o username=votre_utilisateur,password=votre_mdp,uid=1000,gid=1000

# Créer le lien symbolique
cd ~/anemone
rm -rf backup  # Supprimer le répertoire local
ln -s /mnt/nas-backup backup

# Pour un montage permanent, ajouter à /etc/fstab :
# //192.168.1.100/backup /mnt/nas-backup cifs username=user,password=pass,uid=1000,gid=1000 0 0
```

### Option 2 : Partage NFS (Linux, NAS)

```bash
# Installer nfs-common si nécessaire
sudo apt-get install nfs-common  # Debian/Ubuntu
sudo dnf install nfs-utils        # Fedora

# Créer le point de montage
sudo mkdir -p /mnt/nfs-backup

# Monter le partage
sudo mount -t nfs 192.168.1.100:/export/backup /mnt/nfs-backup

# Créer le lien symbolique
cd ~/anemone
rm -rf backup
ln -s /mnt/nfs-backup backup

# Pour un montage permanent, ajouter à /etc/fstab :
# 192.168.1.100:/export/backup /mnt/nfs-backup nfs defaults 0 0
```

### Option 3 : SSHFS (Serveur distant via SSH)

```bash
# Installer sshfs
sudo apt-get install sshfs  # Debian/Ubuntu
sudo dnf install fuse-sshfs  # Fedora

# Créer le point de montage
mkdir -p /mnt/ssh-backup

# Monter le partage
sshfs user@192.168.1.100:/home/user/backup /mnt/ssh-backup

# Créer le lien symbolique
cd ~/anemone
rm -rf backup
ln -s /mnt/ssh-backup backup

# Pour démonter plus tard :
# fusermount -u /mnt/ssh-backup
```

### Option 4 : Bind mount (autre partition locale)

```bash
# Si vous avez un disque dur dédié monté sur /data/backup
cd ~/anemone
rm -rf backup
ln -s /data/backup backup

# Ou avec bind mount
sudo mount --bind /data/backup ~/anemone/backup
```

## 📝 Exemple complet : Monter 3 partages SMB

```bash
#!/bin/bash
# Script de montage des partages réseau pour Anemone

NAS_IP="192.168.1.100"
NAS_USER="anemone"
NAS_PASS="votre_mdp"

# Monter les 3 partages
sudo mount -t cifs //$NAS_IP/anemone-data /mnt/anemone-data \
    -o username=$NAS_USER,password=$NAS_PASS,uid=1000,gid=1000

sudo mount -t cifs //$NAS_IP/anemone-backup /mnt/anemone-backup \
    -o username=$NAS_USER,password=$NAS_PASS,uid=1000,gid=1000

sudo mount -t cifs //$NAS_IP/anemone-backups /mnt/anemone-backups \
    -o username=$NAS_USER,password=$NAS_PASS,uid=1000,gid=1000

# Créer les liens symboliques
cd ~/anemone
rm -rf data backup backups
ln -s /mnt/anemone-data data
ln -s /mnt/anemone-backup backup
ln -s /mnt/anemone-backups backups

echo "✅ Partages montés avec succès"
```

## 🔒 Sécurité : Fichier credentials

Pour éviter de mettre les mots de passe en clair dans `/etc/fstab` :

```bash
# Créer un fichier credentials
sudo nano /root/.anemone-credentials

# Contenu :
username=votre_utilisateur
password=votre_mdp

# Protéger le fichier
sudo chmod 600 /root/.anemone-credentials

# Dans /etc/fstab, utiliser :
# //192.168.1.100/backup /mnt/nas-backup cifs credentials=/root/.anemone-credentials,uid=1000,gid=1000 0 0
```

## ⚙️ Configuration /etc/fstab complète

Exemple pour monter automatiquement au démarrage :

```bash
# Éditer fstab
sudo nano /etc/fstab

# Ajouter ces lignes :
//192.168.1.100/anemone-data    /mnt/anemone-data     cifs  credentials=/root/.anemone-credentials,uid=1000,gid=1000,_netdev 0 0
//192.168.1.100/anemone-backup  /mnt/anemone-backup   cifs  credentials=/root/.anemone-credentials,uid=1000,gid=1000,_netdev 0 0
//192.168.1.100/anemone-backups /mnt/anemone-backups  cifs  credentials=/root/.anemone-credentials,uid=1000,gid=1000,_netdev 0 0

# L'option _netdev attend que le réseau soit prêt avant de monter
```

## ✅ Vérification

Après avoir monté les partages :

```bash
# Vérifier que les partages sont montés
df -h | grep anemone

# Vérifier les liens symboliques
ls -la ~/anemone/ | grep -E "data|backup"

# Tester l'écriture
touch ~/anemone/backup/test.txt
ls -la ~/anemone/backup/test.txt
rm ~/anemone/backup/test.txt

# Démarrer Anemone
cd ~/anemone
./start.sh
```

## 🚨 Problèmes courants

### Permission denied

```bash
# Vérifier les permissions du point de montage
ls -ld /mnt/anemone-backup

# Si nécessaire, corriger les UIDs
sudo chown -R 1000:1000 /mnt/anemone-backup
```

### Le partage ne monte pas au démarrage

```bash
# Vérifier que _netdev est présent dans fstab
# Vérifier les logs
sudo journalctl -u systemd-mount

# Tester le montage manuellement
sudo mount -a
```

### Liens symboliques cassés

```bash
# Vérifier la cible du lien
ls -la ~/anemone/backup

# Si cassé, le recréer
cd ~/anemone
rm backup
ln -s /mnt/anemone-backup backup
```

## 📚 Ressources

- [Documentation CIFS](https://www.kernel.org/doc/html/latest/filesystems/cifs/index.html)
- [Documentation NFS](https://nfs.sourceforge.net/)
- [Documentation SSHFS](https://github.com/libfuse/sshfs)

---

**💡 Conseil** : Préférez NFS pour de meilleures performances si vous avez un NAS Linux/Unix. CIFS/SMB est plus universel mais légèrement plus lent.
