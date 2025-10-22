# 📖 Guide Utilisateur Anemone

Guide complet pour utiliser Anemone au quotidien.

## 📋 Table des matières

- [Installation](#-installation)
- [Premier démarrage](#-premier-démarrage)
- [Ajouter des pairs](#-ajouter-des-pairs)
- [Accéder à vos fichiers](#-accéder-à-vos-fichiers)
- [Gérer la corbeille](#-gérer-la-corbeille)
- [Surveiller les backups](#-surveiller-les-backups)
- [Restauration d'urgence](#-restauration-durgence)
- [Maintenance](#-maintenance)

---

## 🚀 Installation

### Nouveau serveur

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./fr_start.sh  # ou ./en_start.sh pour anglais
```

Le script vous guide à travers :
1. Vérification des prérequis (Docker)
2. Génération des clés (WireGuard, SSH)
3. Configuration du serveur
4. Démarrage automatique

### Restauration d'un serveur

Si vous devez restaurer un serveur existant depuis un backup :

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./fr_restore.sh backup-SERVEUR-DATE.enc
```

Vous aurez besoin de :
- Le fichier de backup `.enc` (récupéré depuis un pair)
- Votre clé Restic (sauvegardée dans Bitwarden)

---

## 🔐 Premier démarrage

Après l'installation, accédez à l'interface web :

```
http://localhost:3000/setup
```

### Nouveau serveur

1. Choisissez **"Nouveau serveur"**
2. Une clé de chiffrement Restic est générée automatiquement
3. **⚠️ SAUVEGARDEZ-LA IMMÉDIATEMENT dans Bitwarden/KeePass**
4. Cochez "J'ai sauvegardé ma clé"
5. Validez

### Restauration

1. Choisissez **"Restauration"**
2. Collez votre clé Restic (depuis Bitwarden)
3. Validez

---

## 👥 Ajouter des pairs

Deux méthodes pour ajouter des serveurs pairs :

### Méthode 1 : Interface web (recommandé)

```
http://localhost:3000/peers
```

1. Cliquez sur "Générer une invitation"
2. Scannez le QR Code ou copiez le code
3. Envoyez-le à votre pair (Signal, email)
4. Votre pair clique "Accepter invitation" et colle le code
5. ✅ Connexion établie automatiquement !

### Méthode 2 : Script interactif

```bash
./scripts/add-peer.sh
```

Le script vous guide pas à pas pour l'échange de clés.

Voir [INTERCONNEXION_GUIDE.md](INTERCONNEXION_GUIDE.md) pour plus de détails.

---

## 📁 Accéder à vos fichiers

### Via SMB (Windows/macOS/Linux)

**Windows** :
```
\\SERVEUR\backup
ou
\\IP_SERVEUR\backup
```

**macOS** :
```
smb://SERVEUR/backup
ou
smb://IP_SERVEUR/backup
```

**Linux** :
```bash
smbclient //SERVEUR/backup -U anemone
```

### Via WebDAV

```
http://IP_SERVEUR:8080/
```

Identifiants par défaut (à changer dans `.env`) :
- Utilisateur : `anemone`
- Mot de passe : `changeme`

---

## 🗑️ Gérer la corbeille

Anemone intègre une corbeille automatique pour protéger contre les suppressions accidentelles.

### Fonctionnement

- Quand vous supprimez un fichier via SMB, il va dans la corbeille
- La corbeille conserve jusqu'à **10 GB** de fichiers
- Nettoyage automatique : les plus vieux fichiers sont supprimés quand la limite est atteinte
- **La corbeille est locale** : elle n'est PAS synchronisée vers les pairs

### Interface web

```
http://localhost:3000/trash
```

Fonctionnalités :
- 📋 Liste des fichiers supprimés (avec date et taille)
- ♻️ Restaurer un fichier en 1 clic
- 🗑️ Supprimer définitivement un fichier
- 🧹 Vider complètement la corbeille

### Emplacement physique

Sur le serveur : `/mnt/backup/.trash/`

---

## 📊 Surveiller les backups

### Dashboard principal

```
http://localhost:3000/
```

Affiche en temps réel :
- État des connexions VPN
- Dernière synchronisation vers chaque pair
- Espace disque utilisé
- Statut des services

### Page Recovery

```
http://localhost:3000/recovery
```

Fonctionnalités :
- 📦 Liste des backups de configuration disponibles
- 📊 Historique complet
- ✅ Vérification d'intégrité
- 🔄 Restauration depuis peer

### Logs

```bash
# Tous les services
docker-compose logs -f

# Service spécifique
docker-compose logs -f core
docker-compose logs -f api
```

---

## 🆘 Restauration d'urgence

### Scénario 1 : Fichier supprimé par erreur

**Solution** : Corbeille

```
http://localhost:3000/trash
→ Trouver le fichier
→ Cliquer "Restaurer"
```

### Scénario 2 : Disque dur crashe (données perdues)

**Solution** : Restauration depuis pair

```
http://localhost:3000/recovery
→ Onglet "Restaurer depuis peer"
→ Choisir le peer source
→ Mode simulation (pour prévisualiser)
→ Restaurer maintenant
```

Toutes vos données utilisateur seront récupérées depuis le pair.

### Scénario 3 : Serveur complètement détruit

**Solution** : Restauration complète

**Étape 1** : Récupérer le backup de configuration

Demandez à un ami avec un pair de :
```
http://localhost:3000/peer-configs
→ Télécharger votre dernier backup
→ Vous l'envoyer par email/Signal
```

**Étape 2** : Restaurer le serveur

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./fr_restore.sh backup-VOTRE_SERVEUR-DATE.enc
```

Entrez votre clé Restic (depuis Bitwarden).

**Étape 3** : Finaliser

```
http://localhost:3000/setup
→ Restauration
→ Coller la clé Restic
```

**Étape 4** : Récupérer vos données

```
http://localhost:3000/recovery
→ Restaurer depuis peer
→ Choisir le peer
→ Restaurer
```

✅ Votre serveur est complètement restauré !

---

## 🛠️ Maintenance

### Mettre à jour Anemone

```bash
cd anemone
git pull
docker-compose down
docker-compose up -d --build
```

### Vérifier l'état

```bash
# État des conteneurs
docker-compose ps

# Santé des services
curl http://localhost:3000/api/status

# Connexions VPN
docker exec anemone-core wg show
```

### Nettoyer les anciennes données

Les backups de configuration sont automatiquement nettoyés (3 dernières versions conservées).

Pour nettoyer manuellement la corbeille :

```
http://localhost:3000/trash
→ Vider la corbeille
```

### Changer les mots de passe

Éditez `.env` :

```bash
nano .env
```

Modifiez :
```
SMB_PASSWORD=VotreNouveauMotDePasse
WEBDAV_PASSWORD=AutreMotDePasse
```

Redémarrez :
```bash
docker-compose restart shares
```

### Sauvegarder votre clé Restic

**⚠️ CRITIQUE** : Sans votre clé, vos backups sont irrécupérables !

Sauvegardez dans **minimum 2 endroits** :
- Bitwarden / 1Password / KeePass
- Clé USB chiffrée dans un coffre
- Papier dans un lieu sûr physique

Pour récupérer la clé (en urgence uniquement) :

```bash
docker exec anemone-core python3 /scripts/decrypt_key.py
```

**Ensuite sauvegardez-la IMMÉDIATEMENT !**

---

## ❓ Problèmes courants

Voir [TROUBLESHOOTING.md](TROUBLESHOOTING.md) pour le guide de dépannage complet.

### Le dashboard affiche "ERROR"

```bash
# Vérifier les logs
docker-compose logs -f core

# Redémarrer les services
docker-compose restart
```

### Un pair est "déconnecté"

```bash
# Tester la connexion VPN
docker exec anemone-core ping IP_DU_PAIR

# Vérifier WireGuard
docker exec anemone-core wg show
```

### La corbeille est pleine

```
http://localhost:3000/trash
→ Vider la corbeille
```

Ou augmentez la limite dans `services/shares/scripts/trash-cleanup.sh` :
```bash
MAX_SIZE_GB=20  # au lieu de 10
```

---

## 📚 Documentation complémentaire

- [README.md](README.md) - Vue d'ensemble et installation
- [INTERCONNEXION_GUIDE.md](INTERCONNEXION_GUIDE.md) - Connecter les serveurs en détail
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Guide de dépannage
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribuer au projet
- [CLAUDE.md](CLAUDE.md) - Documentation technique pour développeurs

---

## 🆘 Besoin d'aide ?

- 📖 [Wiki](https://github.com/juste-un-gars/anemone/wiki)
- 💬 [Discussions](https://github.com/juste-un-gars/anemone/discussions)
- 🐛 [Issues](https://github.com/juste-un-gars/anemone/issues)

---

**Fait avec ❤️ pour partager des fichiers entre proches, sans dépendre du cloud.**
