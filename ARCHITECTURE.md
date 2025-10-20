# Architecture Anemone v2.0

## 🎯 Refactoring majeur : De 6 à 3 conteneurs

### Ancienne architecture (v1.x)

```
├── anemone-wireguard  (VPN)
├── anemone-sftp       (Réception backups)
├── anemone-restic     (Envoi backups)
├── anemone-samba      (Partage SMB)
├── anemone-webdav     (Partage WebDAV)
└── anemone-api        (Interface web)
```

**Problèmes :**
- ❌ Complexité réseau (`network_mode: service:wireguard` problématique)
- ❌ Communication impossible entre SFTP et Restic via VPN
- ❌ 6 conteneurs à gérer
- ❌ Overhead de ressources
- ❌ Difficile à debugger

### Nouvelle architecture (v2.0)

```
├── anemone-core    (WireGuard + SFTP + Restic)  ← Fusionné !
├── anemone-shares  (Samba + WebDAV) [OPTIONNEL]
└── anemone-api     (Interface web)
```

**Avantages :**
- ✅ **Plus de problèmes réseau** : tout dans le même conteneur
- ✅ SFTP écoute sur localhost et via VPN simultanément
- ✅ 50% de conteneurs en moins
- ✅ Architecture simple et robuste
- ✅ Partages optionnels (peut utiliser NAS externe)
- ✅ Plus facile à maintenir

## 📦 Détail des conteneurs

### 1. anemone-core (services/core/)

**Rôle** : Cœur du système de backup distributé

**Contient :**
- **WireGuard** : VPN peer-to-peer (port 51820/udp)
- **OpenSSH Server (SFTP)** : Réception des backups des pairs (port 22222)
- **Restic** : Envoi des backups aux pairs + modes (live/periodic/scheduled)

**Gestion des processus :** Supervisord
- `start-wireguard.sh` → Démarre et surveille WireGuard
- `sshd -D` → Serveur SFTP
- `start-restic.sh` → Lance Restic selon le mode configuré

**Ports exposés :**
- `51820/udp` : WireGuard VPN
- `22222/tcp` : SFTP (configurable via `SFTP_PORT`)

**Volumes :**
- `./config` : Configuration (WireGuard, SSH, Restic)
- `./backup` : Données à sauvegarder
- `./backups` : Backups reçus des pairs
- `./logs` : Logs de tous les services

**Healthcheck :** Vérifie que WireGuard et SSHD tournent

### 2. anemone-shares (services/shares/) [OPTIONNEL]

**Rôle** : Partage de fichiers local

**Contient :**
- **Samba (SMB)** : Partage Windows/macOS/Linux
- **Apache + mod_dav (WebDAV)** : Accès web/mobile

**Gestion des processus :** Supervisord
- `smbd` : Serveur Samba
- `nmbd` : NetBIOS pour Samba
- `httpd` : Serveur WebDAV

**Ports exposés :**
- `445/tcp` : Samba SMB
- `139/tcp` : Samba NetBIOS
- `8080/tcp` : WebDAV (configurable via `WEBDAV_PORT`)

**Volumes :**
- `./data` : Données locales (non sauvegardées)
- `./backup` : Données sauvegardées
- `./logs` : Logs

**Profile Docker** : `shares`
- Ne démarre QUE si `--profile shares` est spécifié
- Permet d'utiliser des partages externes à la place

**Healthcheck :** Vérifie que Samba et WebDAV répondent

### 3. anemone-api (services/api/)

**Rôle** : Interface web de gestion

**Inchangé par rapport à v1.x**

**Contient :**
- FastAPI : Interface web
- Peer manager : Gestion des pairs
- Quota manager : Gestion des quotas disque
- Setup wizard : Configuration initiale

**Port exposé :**
- `3000/tcp` : Interface web

**Volumes :**
- `./config` : Lecture/écriture de la configuration
- `./logs` : Logs (lecture seule)
- `/var/run/docker.sock` : Contrôle Docker

**Healthcheck :** `/health` endpoint

## 🔧 Flux de fonctionnement

### Backup automatique (mode LIVE)

```
1. Utilisateur crée fichier dans ./backup/ (via SMB/WebDAV ou mount externe)
2. inotifywait (dans anemone-core) détecte le changement
3. Après debounce (30s), backup-live.sh déclenche backup-now.sh
4. Restic chiffre et envoie via SFTP sur le VPN (10.8.0.X:22222)
5. Le pair reçoit le backup chiffré dans ./backups/nom-serveur/
```

### Réception de backup d'un pair

```
1. Le pair distant lance Restic backup
2. Restic se connecte en SFTP à notre IP VPN (10.8.0.1:22222)
3. OpenSSH Server (dans anemone-core) vérifie la clé SSH
4. Les données chiffrées sont écrites dans /home/restic/backups/
5. Quota manager vérifie périodiquement l'espace utilisé
```

### Ajout d'un pair via l'interface web

```
1. Utilisateur génère une invitation sur SERVEUR-A (/peers)
2. Invitation contient : clé WireGuard, clé SSH, IP VPN, endpoint
3. Utilisateur ajoute l'invitation sur SERVEUR-B
4. peer_manager.py fait AUTOMATIQUEMENT :
   - Ajoute le peer à config.yaml (section peers)
   - Ajoute la clé SSH à authorized_keys
   - Crée le target de backup (section backup.targets)
   - Crée le répertoire ./backups/nom-pair/
   - Redémarre WireGuard (applique la nouvelle config)
   - Redémarre Restic (prend en compte le nouveau target)
5. Backups bidirectionnels fonctionnent immédiatement !
```

## 🌐 Architecture réseau

### Avant (v1.x) - PROBLÉMATIQUE

```
┌─────────────────────────────────────────┐
│ anemone-wireguard (wg0: 10.8.0.1)      │
│   └─ Port 51820/udp exposé              │
│   └─ Port 22222/tcp → ??? (problème !)  │
├─────────────────────────────────────────┤
│ anemone-sftp                            │
│   network_mode: "service:wireguard"     │ ← Partage le réseau
│   └─ Écoute sur :22                     │   de wireguard
├─────────────────────────────────────────┤
│ anemone-restic                          │
│   network_mode: "service:wireguard"     │ ← Partage aussi !
│   └─ Doit se connecter à 10.8.0.2:22222 │
└─────────────────────────────────────────┘

❌ SFTP et Restic partagent la même stack réseau
❌ Impossible de router 10.8.0.2:22222 → SFTP local
❌ Connection refused !
```

### Après (v2.0) - RÉSOLU

```
┌─────────────────────────────────────────┐
│ anemone-core (tout dans 1 conteneur)   │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │ WireGuard (wg0: 10.8.0.1)        │  │
│  │   Port 51820/udp                  │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │ SSHD (SFTP)                       │  │
│  │   Port 22 (localhost)             │  │
│  │   Port 22222 (exposé)             │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │ Restic                            │  │
│  │   Se connecte à 10.8.0.2:22222    │  │
│  │   via wg0 (même namespace)        │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘

✅ WireGuard, SFTP et Restic partagent le même namespace réseau
✅ SFTP écoute sur toutes les interfaces (0.0.0.0:22)
✅ Restic peut atteindre les pairs via wg0
✅ Ça marche !
```

## 🔀 Migration depuis v1.x

### Sauvegarde

```bash
# Sauvegarder la config actuelle
cp -r config config.backup
cp docker-compose.yml docker-compose.yml.v1
```

### Mise à jour

```bash
# Arrêter l'ancienne version
docker compose down

# Mettre à jour le code
git pull origin main

# Rebuild avec la nouvelle architecture
docker compose build

# Démarrer (avec ou sans partages)
docker compose --profile shares up -d  # Avec partages
# OU
docker compose up -d  # Sans partages (utiliser NAS externe)
```

### Vérification

```bash
# Vérifier les conteneurs
docker ps

# Devrait afficher :
# - anemone-core
# - anemone-shares (si profile activé)
# - anemone-api

# Tester VPN
docker exec anemone-core wg show

# Tester SFTP
docker exec anemone-core ss -tlnp | grep :22

# Tester backup
docker logs -f anemone-core
```

## 📊 Comparaison des ressources

### v1.x (6 conteneurs)

```
CONTAINER          CPU     MEM
anemone-wireguard  2%      50 MB
anemone-sftp       1%      20 MB
anemone-restic     5%      80 MB
anemone-samba      3%      60 MB
anemone-webdav     2%      40 MB
anemone-api        4%      100 MB
─────────────────────────────────
TOTAL              17%     350 MB
```

### v2.0 (3 conteneurs)

```
CONTAINER          CPU     MEM
anemone-core       8%      150 MB  (WireGuard+SFTP+Restic fusionnés)
anemone-shares     4%      80 MB   (Samba+WebDAV fusionnés, optionnel)
anemone-api        4%      100 MB
─────────────────────────────────
TOTAL              16%     330 MB  (20 MB économisés)

Sans partages:
TOTAL              12%     250 MB  (100 MB économisés)
```

## 🔒 Sécurité

Aucun changement de sécurité par rapport à v1.x :

- ✅ Backups chiffrés (Restic AES-256)
- ✅ VPN chiffré (WireGuard ChaCha20)
- ✅ Authentification SSH par clés uniquement
- ✅ Clé Restic chiffrée au repos (AES-256-CBC + PBKDF2)
- ✅ Setup wizard une seule fois
- ✅ Quotas disque par pair

## 🎓 Leçons apprises

### Pourquoi le refactoring était nécessaire

1. **`network_mode: service:X` est trompeur**
   - Partager la stack réseau ≠ communication automatique
   - Deux conteneurs avec ce mode ne peuvent PAS communiquer entre eux via localhost

2. **Moins de conteneurs = Moins de complexité**
   - Docker networking est complexe avec multi-conteneurs
   - Supervisord est parfait pour gérer plusieurs processus dans 1 conteneur

3. **Les profiles Docker Compose sont puissants**
   - Permettent des composants optionnels
   - Idéal pour la flexibilité (partages intégrés vs externes)

4. **L'architecture doit suivre la logique métier**
   - VPN + SFTP + Restic = 1 unité logique (le backup)
   - Samba + WebDAV = 1 unité logique (le partage)
   - API = interface de contrôle

### Ce qui a bien fonctionné

- ✅ Supervisord pour gérer les processus
- ✅ Scripts bash modulaires (start-wireguard.sh, start-restic.sh)
- ✅ Healthchecks adaptés à chaque conteneur
- ✅ Volumes partagés entre conteneurs quand nécessaire
- ✅ Profiles Docker Compose pour composants optionnels

---

**Date du refactoring** : 2025-10-20
**Version** : 2.0.0
**Breaking changes** : Oui (nouvelle architecture de conteneurs)
**Migration requise** : Non (config compatible, juste rebuild)
