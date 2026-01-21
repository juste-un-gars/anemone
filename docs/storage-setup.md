# Configuration du stockage avec ZFS/Btrfs

## 📋 Table des matières

- [Pourquoi utiliser un stockage RAID ?](#pourquoi-utiliser-un-stockage-raid-)
- [Option 1 : ZFS (Recommandé)](#option-1--zfs-recommandé)
- [Option 2 : Btrfs](#option-2--btrfs)
- [Installation de Anemone](#installation-de-anemone)

---

## Pourquoi utiliser un stockage RAID ?

**Il est FORTEMENT recommandé** de configurer un pool de stockage RAID **AVANT** d'installer Anemone.

### Avantages de ZFS

- ✅ **Protection contre la corruption** : Checksums sur toutes les données
- ✅ **Redondance RAID intégrée** : Mirror, RaidZ, RaidZ2
- ✅ **Snapshots instantanés** : Sauvegardes incrémentielles sans temps d'arrêt
- ✅ **Compression transparente** : Économie d'espace automatique
- ✅ **Quotas natifs** : Limitation d'espace par utilisateur
- ✅ **Résilience** : Détection et correction automatique des erreurs

### Avantages de Btrfs

- ✅ **RAID intégré** : RAID0, RAID1, RAID10, RAID5, RAID6
- ✅ **Snapshots** : Points de restauration instantanés
- ✅ **Compression** : lzo, zstd
- ✅ **Quotas** : Support natif (utilisé par Anemone)
- ✅ **Plus simple** que ZFS sur certaines distributions

---

## Option 1 : ZFS (Recommandé)

### Installation avec Cockpit (Interface graphique)

#### Étape 1 : Installer les dépendances

**Debian/Ubuntu** :
```bash
sudo apt update
sudo apt install git zfsutils-linux cockpit -y
```

**Fedora/RHEL** :
```bash
# ZFS nécessite un repository externe sur Fedora
# Voir : https://openzfs.github.io/openzfs-docs/Getting%20Started/Fedora/index.html
sudo dnf install git cockpit -y
```

#### Étape 2 : Installer le module ZFS Manager pour Cockpit

```bash
git clone https://github.com/45drives/cockpit-zfs-manager.git
sudo cp -r cockpit-zfs-manager/zfs /usr/share/cockpit
```

**Note** : Pas besoin de redémarrer, Cockpit détecte automatiquement le nouveau module.

#### Étape 3 : Accéder à Cockpit

Ouvrez votre navigateur et allez sur :

```
https://votre-serveur:9090
```

Connectez-vous avec vos identifiants système.

#### Étape 4 : Créer votre pool ZFS

1. Dans le menu de gauche, cliquez sur **"ZFS"**
2. Cliquez sur **"Create Pool"**
3. Remplissez les informations :
   - **Nom du pool** : `anemone-pool` (ou autre nom)
   - **Point de montage** : `/srv/anemone` ⚠️ **IMPORTANT**
   - **Sélectionnez vos disques** dans la liste
4. Choisissez le **type de redondance** :

| Type | Disques min | Tolérance panne | Capacité utilisable | Recommandation |
|------|-------------|-----------------|---------------------|----------------|
| **Mirror** | 2 | N-1 disques | 50% | ⭐ Simple et fiable |
| **RaidZ** (RAID5) | 3 | 1 disque | (N-1)/N | Bon compromis |
| **RaidZ2** (RAID6) | 4 | 2 disques | (N-2)/N | ⭐ Production |
| **RaidZ3** | 5 | 3 disques | (N-3)/N | Haute sécurité |

5. Cliquez sur **"Create"**

#### Étape 5 : Vérifier le pool

```bash
# Vérifier l'état du pool
sudo zpool status

# Vérifier le montage
df -h | grep /srv/anemone
```

Vous devriez voir :
```
anemone-pool   X.XG   XX.XK   X.XG   1% /srv/anemone
```

### Installation en ligne de commande (Alternative)

Si vous préférez la ligne de commande :

#### Mirror (2 disques)
```bash
sudo zpool create -m /srv/anemone anemone-pool mirror /dev/sdb /dev/sdc
```

#### RaidZ (3+ disques, tolère 1 panne)
```bash
sudo zpool create -m /srv/anemone anemone-pool raidz /dev/sdb /dev/sdc /dev/sdd
```

#### RaidZ2 (4+ disques, tolère 2 pannes)
```bash
sudo zpool create -m /srv/anemone anemone-pool raidz2 /dev/sdb /dev/sdc /dev/sdd /dev/sde
```

#### RaidZ3 (5+ disques, tolère 3 pannes)
```bash
sudo zpool create -m /srv/anemone anemone-pool raidz3 /dev/sdb /dev/sdc /dev/sdd /dev/sde /dev/sdf
```

### Optimisations recommandées

#### Activer la compression (économise de l'espace)
```bash
sudo zfs set compression=lz4 anemone-pool
```

#### Désactiver atime (améliore les performances)
```bash
sudo zfs set atime=off anemone-pool
```

#### Activer les snapshots automatiques (optionnel)
```bash
# Installer zfs-auto-snapshot
sudo apt install zfs-auto-snapshot  # Debian/Ubuntu
sudo dnf install zfs-auto-snapshot  # Fedora

# Activer pour le pool
sudo zfs set com.sun:auto-snapshot=true anemone-pool
```

---

## Option 2 : Btrfs

### RAID1 (Mirror, 2+ disques)

```bash
# Créer le filesystem Btrfs
sudo mkfs.btrfs -L anemone-pool -m raid1 -d raid1 /dev/sdb /dev/sdc

# Créer le point de montage
sudo mkdir -p /srv/anemone

# Monter
sudo mount /dev/sdb /srv/anemone

# Ajouter au fstab pour montage automatique
UUID=$(sudo blkid -s UUID -o value /dev/sdb)
echo "UUID=$UUID /srv/anemone btrfs defaults 0 0" | sudo tee -a /etc/fstab
```

### RAID10 (4+ disques, striping + mirroring)

```bash
sudo mkfs.btrfs -L anemone-pool -m raid10 -d raid10 /dev/sd{b,c,d,e}
sudo mkdir -p /srv/anemone
sudo mount /dev/sdb /srv/anemone

# Ajouter au fstab
UUID=$(sudo blkid -s UUID -o value /dev/sdb)
echo "UUID=$UUID /srv/anemone btrfs defaults 0 0" | sudo tee -a /etc/fstab
```

### RAID5 (3+ disques, 1 panne tolérée)

```bash
sudo mkfs.btrfs -L anemone-pool -m raid5 -d raid5 /dev/sd{b,c,d}
sudo mkdir -p /srv/anemone
sudo mount /dev/sdb /srv/anemone

# Ajouter au fstab
UUID=$(sudo blkid -s UUID -o value /dev/sdb)
echo "UUID=$UUID /srv/anemone btrfs defaults 0 0" | sudo tee -a /etc/fstab
```

### RAID6 (4+ disques, 2 pannes tolérées)

```bash
sudo mkfs.btrfs -L anemone-pool -m raid6 -d raid6 /dev/sd{b,c,d,e}
sudo mkdir -p /srv/anemone
sudo mount /dev/sdb /srv/anemone

# Ajouter au fstab
UUID=$(sudo blkid -s UUID -o value /dev/sdb)
echo "UUID=$UUID /srv/anemone btrfs defaults 0 0" | sudo tee -a /etc/fstab
```

### Vérification Btrfs

```bash
# Voir les informations du filesystem
sudo btrfs filesystem show

# Voir l'utilisation
sudo btrfs filesystem usage /srv/anemone

# Vérifier le montage
df -h | grep /srv/anemone
```

---

## Installation de Anemone

Une fois votre pool de stockage créé et monté sur `/srv/anemone`, vous pouvez installer Anemone :

```bash
# Cloner le repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Lancer l'installation
sudo ./install.sh
```

Le script d'installation détectera automatiquement que `/srv/anemone` est monté et continuera l'installation.

---

## Commandes utiles

### ZFS

```bash
# Voir l'état des pools
zpool status

# Voir l'utilisation
zfs list

# Créer un snapshot
sudo zfs snapshot anemone-pool@$(date +%Y%m%d)

# Lister les snapshots
zfs list -t snapshot

# Restaurer un snapshot
sudo zfs rollback anemone-pool@20260112

# Scrub (vérification d'intégrité)
sudo zpool scrub anemone-pool
```

### Btrfs

```bash
# Voir les filesystems
sudo btrfs filesystem show

# Voir l'utilisation
sudo btrfs filesystem usage /srv/anemone

# Créer un snapshot
sudo btrfs subvolume snapshot /srv/anemone /srv/anemone-snapshot-$(date +%Y%m%d)

# Lister les subvolumes/snapshots
sudo btrfs subvolume list /srv/anemone

# Balance (optimisation)
sudo btrfs balance start /srv/anemone

# Scrub (vérification)
sudo btrfs scrub start /srv/anemone
```

---

## FAQ

### Puis-je migrer vers ZFS/Btrfs après installation ?

Oui, mais c'est plus complexe. Il faut :
1. Arrêter Anemone
2. Copier les données vers le nouveau pool
3. Modifier le service systemd
4. Redémarrer

Il est préférable de configurer le stockage AVANT l'installation.

### Que se passe-t-il si j'installe sans RAID ?

Anemone fonctionnera, mais vous n'aurez :
- ❌ Aucune redondance (perte d'un disque = perte de données)
- ❌ Pas de snapshots
- ❌ Pas de compression
- ❌ Pas de protection contre la corruption

### ZFS ou Btrfs ?

**ZFS** si :
- ✅ Vous voulez la meilleure protection des données
- ✅ Vous êtes sur Debian/Ubuntu (installation facile)
- ✅ Vous avez assez de RAM (recommandé : 1GB par TB)

**Btrfs** si :
- ✅ Vous êtes sur Fedora/RHEL
- ✅ Vous avez peu de RAM
- ✅ Vous voulez plus de simplicité

### Combien de disques faut-il ?

| Configuration | Disques min | Recommandation |
|---------------|-------------|----------------|
| **Test/Home** | 2 (Mirror) | Fiable et simple |
| **PME** | 4 (RaidZ2) | Bon compromis |
| **Production** | 6+ (RaidZ2) | Sécurité maximale |

### Dois-je utiliser Cockpit ?

**Non, c'est optionnel**. Cockpit fournit juste une interface graphique pratique pour :
- Créer des pools visuellement
- Monitorer la santé des disques
- Gérer les snapshots

Vous pouvez tout faire en ligne de commande si vous préférez.

---

## Ressources

- 📖 [Documentation ZFS](https://openzfs.github.io/openzfs-docs/)
- 📖 [Documentation Btrfs](https://btrfs.wiki.kernel.org/)
- 🖥️ [Cockpit Project](https://cockpit-project.org/)
- 🖥️ [Cockpit ZFS Manager](https://github.com/45drives/cockpit-zfs-manager)

---

**Prêt ?** Une fois votre pool configuré, retournez au [README principal](../README.md) pour installer Anemone ! 🚀
