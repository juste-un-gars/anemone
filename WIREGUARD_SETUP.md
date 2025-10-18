# Configuration WireGuard - Anemone

## 🔧 Problème résolu

**Avant** : Les clés générées par `init.sh` dans `config/wireguard/` n'étaient **jamais utilisées** par le conteneur WireGuard. Le conteneur générait ses propres clés, créant une désynchronisation totale.

**Maintenant** : `init.sh` génère les clés ET crée le fichier `config/wg_confs/wg0.conf` que le conteneur utilise directement.

## 📁 Structure des fichiers

```
config/
├── wireguard/
│   ├── private.key      # Clé privée (sauvegarde uniquement, NE PAS PARTAGER)
│   └── public.key       # Clé publique (à partager avec vos pairs)
└── wg_confs/
    └── wg0.conf         # Configuration WireGuard UTILISÉE par le conteneur
```

## 🚀 Initialisation (nouveau serveur)

```bash
# 1. Générer les clés et créer wg0.conf
./scripts/init.sh

# 2. Afficher vos clés publiques
./scripts/show-keys.sh

# 3. Démarrer les services
docker-compose up -d

# 4. Vérifier que WireGuard utilise les bonnes clés
docker exec anemone-wireguard wg show
```

## 🤝 Ajouter un pair

### Méthode automatique (recommandée)

```bash
./scripts/add-peer.sh
```

Le script vous demandera :
- Nom du pair (ex: alice)
- Clé publique WireGuard du pair
- Endpoint public (ex: alice.duckdns.org:51820)
- IP VPN du pair (ex: 10.8.0.2)
- Clé publique SSH du pair

**Important** : Le script modifie maintenant **DEUX fichiers** :
1. `config/config.yaml` (pour la configuration Anemone)
2. `config/wg_confs/wg0.conf` (pour WireGuard)

Puis **redémarrez WireGuard** :

```bash
docker-compose restart wireguard
```

### Méthode manuelle

Ajoutez cette section à la fin de `config/wg_confs/wg0.conf` :

```ini
# Peer: alice
[Peer]
PublicKey = CLE_PUBLIQUE_ALICE
AllowedIPs = 10.8.0.2/32
Endpoint = alice.duckdns.org:51820
PersistentKeepalive = 25
```

Puis redémarrez WireGuard :

```bash
docker-compose restart wireguard
```

## 🔍 Vérification

### Vérifier la clé publique utilisée par WireGuard

```bash
# Méthode 1 : Via wg show
docker exec anemone-wireguard wg show

# Méthode 2 : Via le fichier (doit correspondre à config/wireguard/public.key)
cat config/wireguard/public.key

# Les deux doivent afficher la MÊME clé !
```

### Vérifier les peers configurés

```bash
docker exec anemone-wireguard wg show
```

Vous devriez voir :
- Votre `public key`
- Liste des `peer:` avec leur clé publique
- `endpoint:` pour chaque peer
- `allowed ips:` pour chaque peer
- `latest handshake:` si la connexion est établie

### Tester la connexion VPN

```bash
# Depuis le conteneur Restic (qui utilise le réseau WireGuard)
docker exec anemone-restic ping 10.8.0.2
```

## ⚠️ Points critiques

### 1. Ne jamais éditer wg0.conf pendant que le conteneur tourne

```bash
# Arrêtez d'abord
docker-compose stop wireguard

# Modifiez wg0.conf
nano config/wg_confs/wg0.conf

# Redémarrez
docker-compose start wireguard
```

### 2. Backup automatique de wg0.conf

Le script `add-peer.sh` crée automatiquement un backup :
```
config/wg_confs/wg0.conf.backup.20251018_154530
```

### 3. Fichiers à NE JAMAIS committer dans git

✅ Déjà dans `.gitignore` :
- `config/wireguard/private.key`
- `config/wg_confs/wg0.conf`

## 🐛 Dépannage

### "Les clés ne correspondent pas"

Si `docker exec anemone-wireguard wg show` affiche une clé différente de `config/wireguard/public.key` :

```bash
# 1. Arrêter le conteneur
docker-compose stop wireguard

# 2. Supprimer le wg0.conf actuel
rm config/wg_confs/wg0.conf

# 3. Régénérer avec init.sh
./scripts/init.sh

# 4. Redémarrer
docker-compose start wireguard

# 5. Vérifier
docker exec anemone-wireguard wg show
cat config/wireguard/public.key
```

### "Le pair n'apparaît pas dans wg show"

1. Vérifiez que le pair est bien dans `wg0.conf` :
   ```bash
   cat config/wg_confs/wg0.conf
   ```

2. Redémarrez WireGuard :
   ```bash
   docker-compose restart wireguard
   ```

3. Vérifiez les logs :
   ```bash
   docker logs anemone-wireguard
   ```

### "Connection refused" lors du ping

Vérifiez les deux côtés :

**Côté local** :
```bash
docker exec anemone-wireguard wg show
# Vérifiez que le peer apparaît
```

**Côté distant** :
```bash
# Sur le serveur distant
docker exec anemone-wireguard wg show
# Vérifiez que VOTRE serveur apparaît comme peer
```

## 📝 Flux de travail typique

### Connecter deux serveurs Anemone (Alice et Bob)

**Sur le serveur d'Alice (10.8.0.1)** :

```bash
# 1. Afficher les clés publiques d'Alice
./scripts/show-keys.sh

# 2. Envoyer à Bob :
#    - Clé publique WireGuard d'Alice
#    - Clé publique SSH d'Alice
#    - Endpoint public d'Alice (ex: alice.duckdns.org:51820)
```

**Sur le serveur de Bob (10.8.0.2)** :

```bash
# 1. Afficher les clés publiques de Bob
./scripts/show-keys.sh

# 2. Ajouter Alice comme peer
./scripts/add-peer.sh
# Entrer : alice, <clé WG d'Alice>, alice.duckdns.org:51820, 10.8.0.1, <clé SSH d'Alice>

# 3. Redémarrer WireGuard
docker-compose restart wireguard

# 4. Envoyer à Alice :
#    - Clé publique WireGuard de Bob
#    - Clé publique SSH de Bob
#    - Endpoint public de Bob (ex: bob.duckdns.org:51820)
```

**Retour sur le serveur d'Alice** :

```bash
# 1. Ajouter Bob comme peer
./scripts/add-peer.sh
# Entrer : bob, <clé WG de Bob>, bob.duckdns.org:51820, 10.8.0.2, <clé SSH de Bob>

# 2. Redémarrer WireGuard
docker-compose restart wireguard

# 3. Tester la connexion
docker exec anemone-restic ping 10.8.0.2
```

**Les deux côtés devraient maintenant pouvoir communiquer via le VPN !** 🎉

## 🔐 Sécurité

- **Clé privée WireGuard** : Ne JAMAIS partager, reste sur le serveur
- **Clé publique WireGuard** : À partager avec vos pairs (non sensible)
- **Clé privée SSH** : Ne JAMAIS partager, reste sur le serveur
- **Clé publique SSH** : À partager avec vos pairs pour autoriser les backups (non sensible)
- **wg0.conf** : Contient votre clé privée, ne JAMAIS committer dans git

## 📚 Références

- [WireGuard Quickstart](https://www.wireguard.com/quickstart/)
- [linuxserver/wireguard Documentation](https://docs.linuxserver.io/images/docker-wireguard/)
- Fichier `INTERCONNEXION_GUIDE.md` pour le processus complet
