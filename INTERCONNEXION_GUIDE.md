# 🔗 Guide d'interconnexion entre serveurs Anemone

Ce guide explique comment connecter plusieurs serveurs Anemone ensemble pour qu'ils se sauvegardent mutuellement leurs données.

## 📋 Vue d'ensemble

**Scénario typique** : Vous (Alice) et votre ami (Bob) voulez vous échanger des backups.

```
┌─────────────────┐                    ┌─────────────────┐
│  Serveur Alice  │◄──── WireGuard ───►│  Serveur Bob    │
│  10.8.0.1       │     VPN tunnel     │  10.8.0.2       │
│                 │                    │                 │
│  Backup vers ─────────────────────────► Reçoit backup  │
│  Bob (SFTP)     │                    │  d'Alice        │
│                 │                    │                 │
│  Reçoit backup ◄─────────────────────── Backup vers    │
│  de Bob         │                    │  Alice (SFTP)   │
└─────────────────┘                    └─────────────────┘
```

## 🚀 Procédure complète

### Prérequis

Les deux serveurs doivent avoir :
- ✅ Anemone installé et démarré (`docker compose up -d`)
- ✅ Setup web complété (clé Restic configurée)
- ✅ Un nom de domaine DynDNS ou IP publique fixe
- ✅ Port 51820/UDP ouvert sur le routeur (port-forwarding)

---

## ÉTAPE 1 : Échange des informations

### Sur le serveur d'Alice

```bash
cd ~/anemone

# 1. Afficher la clé publique WireGuard
echo "=== Ma clé publique WireGuard ==="
cat config/wireguard/public.key

# 2. Afficher la clé publique SSH
echo "=== Ma clé publique SSH ==="
cat config/ssh/id_rsa.pub

# 3. Afficher l'IP VPN (dans config.yaml)
echo "=== Mon IP VPN ==="
grep "address:" config/config.yaml | head -1

# 4. Mon endpoint public
echo "=== Mon endpoint public ==="
# Remplacez par votre DNS dynamique
echo "alice.duckdns.org:51820"
```

**Alice envoie ces 4 informations à Bob** (par email chiffré, Signal, etc.)

### Sur le serveur de Bob

Bob fait exactement la même chose et envoie ses informations à Alice.

---

## ÉTAPE 2 : Configuration sur le serveur d'Alice

Alice ajoute Bob comme pair :

```bash
cd ~/anemone

# 1. Éditer la configuration
nano config/config.yaml
```

Dans la section `peers:`, ajouter :

```yaml
peers:
  - name: "bob"
    endpoint: "bob.duckdns.org:51820"      # ← Endpoint de Bob
    public_key: "CLE_PUBLIQUE_WIREGUARD_DE_BOB"  # ← Clé WireGuard de Bob
    allowed_ips: "10.8.0.2/32"             # ← IP VPN de Bob
    persistent_keepalive: 25
    description: "Serveur de Bob"
```

Dans la section `backup.targets:`, ajouter :

```yaml
backup:
  targets:
    - name: "bob-backup"
      enabled: true
      type: "sftp"
      host: "10.8.0.2"                     # ← IP VPN de Bob
      port: 2222
      user: "restic"
      path: "/backups/alice"               # ← Chemin chez Bob
```

Dans la section `restic_server.authorized_keys:`, ajouter :

```yaml
restic_server:
  enabled: true
  port: 2222
  username: "restic"
  authorized_keys:
    - "CLE_PUBLIQUE_SSH_DE_BOB"            # ← Clé SSH de Bob
```

**Ou utiliser le script** :

```bash
# Alternative plus simple
./scripts/add-peer.sh
# Suivre les instructions interactives
```

---

## ÉTAPE 3 : Configuration sur le serveur de Bob

Bob fait la même chose mais avec les informations d'Alice :

```bash
cd ~/anemone
nano config/config.yaml
```

```yaml
peers:
  - name: "alice"
    endpoint: "alice.duckdns.org:51820"
    public_key: "CLE_PUBLIQUE_WIREGUARD_ALICE"
    allowed_ips: "10.8.0.1/32"             # ← IP VPN d'Alice
    persistent_keepalive: 25

backup:
  targets:
    - name: "alice-backup"
      enabled: true
      type: "sftp"
      host: "10.8.0.1"                     # ← IP VPN d'Alice
      port: 2222
      user: "restic"
      path: "/backups/bob"

restic_server:
  enabled: true
  authorized_keys:
    - "CLE_PUBLIQUE_SSH_ALICE"
```

---

## ÉTAPE 4 : Activer le service SFTP

Les deux serveurs doivent activer le profil SFTP :

```bash
cd ~/anemone

# Éditer docker-compose.yml
nano docker-compose.yml
```

**Vérifier que le service SFTP n'a PAS** de section `profiles:` (ou la commenter) :

```yaml
sftp:
  image: atmoz/sftp:latest
  container_name: anemone-sftp
  volumes:
    - ${BACKUP_PATH:-./backups}:/home/restic/backups
    - ./config/ssh/authorized_keys:/home/restic/.ssh/keys/authorized_keys:ro
  ports:
    - "${SFTP_PORT:-2222}:22"
  command: restic:restic:1000:1000:backups
  restart: unless-stopped
  networks:
    - anemone-net
  # profiles:              # ← Commenter ou supprimer ces 2 lignes
  #   - sftp-enabled
```

---

## ÉTAPE 5 : Redémarrer les services

### Sur les deux serveurs :

```bash
cd ~/anemone

# Arrêter les services
docker compose down

# Redémarrer (inclut maintenant SFTP)
docker compose up -d

# Attendre 30 secondes que tout démarre
sleep 30
```

---

## ÉTAPE 6 : Vérification de la connexion

### Sur le serveur d'Alice

```bash
# 1. Vérifier que WireGuard est actif
docker exec anemone-wireguard wg show

# Vous devriez voir Bob dans la liste des peers

# 2. Tester la connexion VPN avec Bob
docker exec anemone-restic ping -c 3 10.8.0.2

# Doit répondre "64 bytes from 10.8.0.2..."

# 3. Tester la connexion SFTP vers Bob
docker exec anemone-restic sftp -P 2222 restic@10.8.0.2 <<EOF
ls
quit
EOF

# Doit lister les dossiers sans erreur
```

### Sur le serveur de Bob

Faire la même chose mais avec l'IP d'Alice (10.8.0.1).

---

## ÉTAPE 7 : Test de backup

### Sur le serveur d'Alice

```bash
# Déclencher un backup manuel
docker exec anemone-restic /scripts/backup-now.sh

# Vérifier les logs
docker logs anemone-restic -f

# Vérifier que le backup est arrivé chez Bob
docker exec anemone-restic restic -r sftp:restic@10.8.0.2:/backups/alice snapshots
```

Si tout fonctionne, vous verrez la liste des snapshots créés chez Bob !

---

## 🔧 Dépannage

### WireGuard : Pas de connexion

```bash
# Vérifier les logs WireGuard
docker logs anemone-wireguard

# Vérifier que le port 51820 est ouvert
sudo netstat -tulpn | grep 51820

# Vérifier le port-forwarding sur la box
# Allez dans l'interface de votre routeur
# NAT/PAT → Ajouter : Port 51820/UDP → IP locale du serveur
```

### SFTP : Permission denied

```bash
# Vérifier que la clé SSH publique du pair est bien dans authorized_keys
cat config/ssh/authorized_keys

# Vérifier les permissions
ls -la config/ssh/

# Le fichier authorized_keys doit avoir les permissions 600 ou 644
chmod 644 config/ssh/authorized_keys
docker compose restart sftp
```

### Backup échoue : "repository does not exist"

```bash
# Initialiser le repository Restic chez le pair
docker exec anemone-restic restic -r sftp:restic@10.8.0.2:/backups/alice init

# Entrer le mot de passe Restic quand demandé
```

### Ping fonctionne mais pas SFTP

```bash
# Vérifier que le conteneur SFTP tourne
docker ps | grep sftp

# Si absent, vérifier le docker-compose.yml (section profiles commentée)

# Redémarrer avec le profil SFTP explicite
docker compose --profile sftp-enabled up -d
```

---

## 📊 Vérification de la santé du système

Checklist complète :

```bash
# 1. WireGuard actif
docker exec anemone-wireguard wg show | grep -q "peer:" && echo "✅ WireGuard OK"

# 2. Connexion VPN établie
docker exec anemone-restic ping -c 1 10.8.0.2 &>/dev/null && echo "✅ VPN OK"

# 3. SFTP accessible
docker exec anemone-restic echo "quit" | sftp -P 2222 restic@10.8.0.2 &>/dev/null && echo "✅ SFTP OK"

# 4. Repository Restic existe
docker exec anemone-restic restic -r sftp:restic@10.8.0.2:/backups/alice snapshots &>/dev/null && echo "✅ Restic OK"

# 5. Backup automatique configuré
docker logs anemone-restic | grep -q "Backup" && echo "✅ Backup schedulé"
```

---

## 🔐 Sécurité

**Points importants** :

✅ **Clés SSH** : Chaque serveur a sa propre paire de clés SSH
✅ **Clés WireGuard** : Jamais partagées, seules les clés publiques sont échangées
✅ **Clé Restic** : Reste privée sur chaque serveur, jamais transmise
✅ **VPN chiffré** : Toutes les communications passent par WireGuard (cryptographie moderne)
✅ **Backups chiffrés** : Les données stockées chez le pair sont chiffrées par Restic

**Le pair ne peut PAS** :
- ❌ Lire vos backups (ils sont chiffrés)
- ❌ Se connecter à votre serveur (pas de clé privée)
- ❌ Accéder à vos fichiers locaux (seul SFTP est exposé)

**Le pair PEUT** :
- ✅ Stocker vos backups chiffrés
- ✅ Se connecter via SFTP pour déposer ses backups chez vous

---

## 🎯 Résumé des commandes essentielles

```bash
# Afficher mes informations à partager
cat config/wireguard/public.key     # Clé publique WireGuard
cat config/ssh/id_rsa.pub           # Clé publique SSH
grep "address:" config/config.yaml  # Mon IP VPN

# Ajouter un pair
./scripts/add-peer.sh

# Redémarrer après config
docker compose down && docker compose up -d

# Tester la connexion
docker exec anemone-restic ping 10.8.0.2
docker exec anemone-wireguard wg show

# Backup manuel
docker exec anemone-restic /scripts/backup-now.sh

# Voir les snapshots chez le pair
docker exec anemone-restic restic -r sftp:restic@10.8.0.2:/backups/moi snapshots
```

---

## 📚 Ressources

- **WireGuard** : https://www.wireguard.com/
- **Restic** : https://restic.readthedocs.io/
- **DuckDNS** (DNS dynamique gratuit) : https://www.duckdns.org/
- **Port-forwarding** : Consultez le manuel de votre box Internet

---

**Besoin d'aide ?** Consultez TROUBLESHOOTING.md ou ouvrez une issue sur GitHub.
