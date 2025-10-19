# Guide de dépannage VPN - Anemone

## 🔴 Symptômes

- Impossible de ping entre serveurs via les IPs VPN (ex: 10.8.0.1 ↔ 10.8.0.2)
- Le ping local fonctionne (ex: 192.168.x.x)
- Erreur typique : `ping: connect: Network is unreachable` ou timeout

## 🔍 Diagnostic

### Étape 1 : Exécuter le script de diagnostic

Sur **CHAQUE serveur**, exécutez :

```bash
cd /chemin/vers/anemone
./scripts/diagnose-vpn.sh
```

Ce script va vérifier :
- ✅ Conteneur WireGuard en cours d'exécution
- ✅ Interface wg0 configurée avec une IP
- ✅ Fichier `config/wg_confs/wg0.conf` existe
- ✅ Clés publiques valides (pas de placeholders)
- ✅ Peers configurés dans wg0.conf
- ✅ Cohérence entre clé privée dans wg0.conf et private.key
- ✅ Logs WireGuard pour erreurs

### Étape 2 : Identifier le problème

Le script de diagnostic vous indiquera le problème exact. Voici les cas les plus fréquents :

---

## 🛠️ Solutions par problème

### Problème 1 : `wg0.conf` manquant

**Symptôme** :
```
✗ config/wg_confs/wg0.conf MANQUANT !
```

**Cause** : L'initialisation a été faite avec une ancienne version du script `init.sh` qui ne créait pas `wg0.conf`.

**Solution** :

**Option A - Régénérer depuis config.yaml** (recommandé si vous avez déjà ajouté des peers dans config.yaml) :
```bash
./scripts/regenerate-wg-config.sh
docker-compose restart wireguard
```

**Option B - Réinitialiser complètement** (si pas de peers configurés) :
```bash
# Backup de votre config actuelle
cp config/config.yaml config/config.yaml.backup

# Régénérer wg0.conf
./scripts/init.sh

# Restaurer votre config personnalisée
mv config/config.yaml.backup config/config.yaml

# Redémarrer
docker-compose restart wireguard
```

---

### Problème 2 : Aucun peer dans wg0.conf

**Symptôme** :
```
❌ PROBLÈME : Aucun peer dans wg0.conf
Nombre de peers configurés : 0
```

**Cause** : Les peers ont été ajoutés dans `config.yaml` mais pas dans `wg0.conf` (si vous avez utilisé une ancienne version de `add-peer.sh`).

**Solution** :

1. **Régénérer wg0.conf depuis config.yaml** :
   ```bash
   ./scripts/regenerate-wg-config.sh
   ```

2. **Redémarrer WireGuard** (OBLIGATOIRE) :
   ```bash
   docker-compose restart wireguard
   ```

3. **Vérifier que les peers sont bien là** :
   ```bash
   docker exec anemone-wireguard wg show
   ```

   Vous devriez voir vos peers listés avec leur clé publique.

---

### Problème 3 : Clé publique est un placeholder

**Symptôme** :
```
⚠ ATTENTION : Clé publique est un placeholder !
# Clé publique sera générée au démarrage du conteneur
```

**Cause** : La génération de clé publique via Docker a échoué lors de `init.sh`.

**Solution** :
```bash
./scripts/extract-wireguard-pubkey.sh
```

Ensuite, **IMPORTANT** : Vous devez partager cette nouvelle clé publique avec vos pairs et leur demander de mettre à jour leur configuration.

---

### Problème 4 : Clé privée dans wg0.conf ne correspond pas

**Symptôme** :
```
✗ ERREUR : Clé privée dans wg0.conf ne correspond pas !
```

**Cause** : Désynchronisation entre `config/wireguard/private.key` et `config/wg_confs/wg0.conf`.

**Solution** :
```bash
./scripts/regenerate-wg-config.sh
docker-compose restart wireguard
```

---

### Problème 5 : Configuration correcte mais ping ne fonctionne toujours pas

**Vérifications à faire sur les DEUX serveurs** :

#### A. Vérifier que les deux serveurs se connaissent mutuellement

**Sur Serveur 1 (10.8.0.1)** :
```bash
# Vérifier que Serveur 2 est dans les peers
docker exec anemone-wireguard wg show | grep -A5 "peer:"
```

Vous devriez voir un bloc comme :
```
peer: ABC123...  (clé publique du Serveur 2)
  endpoint: serveur2.duckdns.org:51820
  allowed ips: 10.8.0.2/32
```

**Sur Serveur 2 (10.8.0.2)** :
```bash
# Vérifier que Serveur 1 est dans les peers
docker exec anemone-wireguard wg show | grep -A5 "peer:"
```

#### B. Vérifier que les clés publiques correspondent

**Sur Serveur 1** :
```bash
# Afficher VOTRE clé publique
cat config/wireguard/public.key
```

**Sur Serveur 2** :
```bash
# Vérifier que la clé publique du Serveur 1 est bien celle-ci
grep "PublicKey" config/wg_confs/wg0.conf
```

**Les clés doivent correspondre exactement !**

#### C. Vérifier les endpoints

Si vos serveurs sont derrière des NAT, au moins l'un des deux doit avoir :
- Une IP publique statique OU
- Un nom de domaine DynDNS (ex: serveur1.duckdns.org)
- Le port 51820/UDP ouvert (port forwarding sur le routeur)

**Test de connectivité** :
```bash
# Depuis l'extérieur, tester si le port est ouvert
nc -u -v votre-endpoint.duckdns.org 51820
```

#### D. Forcer un handshake

Parfois il faut "réveiller" la connexion :

```bash
# Sur Serveur 1, ping vers Serveur 2
docker exec anemone-restic ping -c 5 10.8.0.2

# Vérifier si handshake établi
docker exec anemone-wireguard wg show
```

Si vous voyez `latest handshake: X seconds ago`, c'est bon signe !

---

## 📋 Checklist complète pour connecter deux serveurs

### Sur Serveur 1 (10.8.0.1)

- [ ] `./scripts/init.sh` exécuté
- [ ] `config/wg_confs/wg0.conf` existe
- [ ] Clé publique valide dans `config/wireguard/public.key`
- [ ] Serveur 2 ajouté avec `./scripts/add-peer.sh`
- [ ] WireGuard redémarré : `docker-compose restart wireguard`
- [ ] Interface wg0 a l'IP 10.8.0.1 : `docker exec anemone-wireguard ip addr show wg0`
- [ ] Peer visible dans `wg show`

### Sur Serveur 2 (10.8.0.2)

- [ ] `./scripts/init.sh` exécuté
- [ ] `config/wg_confs/wg0.conf` existe
- [ ] Clé publique valide dans `config/wireguard/public.key`
- [ ] Serveur 1 ajouté avec `./scripts/add-peer.sh`
- [ ] WireGuard redémarré : `docker-compose restart wireguard`
- [ ] Interface wg0 a l'IP 10.8.0.2 : `docker exec anemone-wireguard ip addr show wg0`
- [ ] Peer visible dans `wg show`

### Test final

```bash
# Depuis Serveur 1
docker exec anemone-restic ping 10.8.0.2

# Depuis Serveur 2
docker exec anemone-restic ping 10.8.0.1
```

**Si ça fonctionne** : ✅ VPN opérationnel !

---

## 🔧 Commandes utiles

### Voir la configuration WireGuard actuelle
```bash
docker exec anemone-wireguard wg show
```

### Voir l'IP de l'interface wg0
```bash
docker exec anemone-wireguard ip addr show wg0
```

### Voir les logs WireGuard
```bash
docker logs anemone-wireguard --tail 50
```

### Régénérer wg0.conf depuis config.yaml
```bash
./scripts/regenerate-wg-config.sh
```

### Redémarrer WireGuard sans tout arrêter
```bash
docker-compose restart wireguard
```

### Redémarrer tout Anemone
```bash
docker-compose down
docker-compose up -d
```

### Afficher vos clés publiques à partager
```bash
./scripts/show-keys.sh
```

---

## 🆘 Si rien ne fonctionne

1. **Récupérez les diagnostics des deux serveurs** :
   ```bash
   # Sur chaque serveur
   ./scripts/diagnose-vpn.sh > diagnostic-serveur-X.txt
   ```

2. **Vérifiez que Docker a bien les bonnes capacités** :
   ```bash
   docker inspect anemone-wireguard | grep -A10 CapAdd
   ```

   Doit contenir `NET_ADMIN` et `SYS_MODULE`.

3. **Testez manuellement avec wg-quick** :
   ```bash
   # Entrer dans le conteneur
   docker exec -it anemone-wireguard sh

   # Vérifier la config
   wg show

   # Relancer wg
   wg-quick down wg0
   wg-quick up wg0
   ```

4. **Vérifiez les règles iptables** :
   ```bash
   docker exec anemone-wireguard iptables -L -n -v
   ```

---

## 📚 Ressources

- [Documentation WireGuard officielle](https://www.wireguard.com/quickstart/)
- [INTERCONNEXION_GUIDE.md](./INTERCONNEXION_GUIDE.md) - Guide complet d'interconnexion
- [WIREGUARD_SETUP.md](./WIREGUARD_SETUP.md) - Configuration détaillée de WireGuard

---

## 🐛 Cas particuliers

### Serveurs derrière le même NAT

Si vos deux serveurs sont sur le même réseau local :
- Utilisez les IPs locales comme endpoint (ex: `192.168.1.100:51820`)
- Pas besoin de port forwarding
- Le VPN fonctionnera en local

### Serveurs tous deux derrière des NATs différents

Il faut au moins un serveur avec :
- IP publique statique OU
- DynDNS + port forwarding

L'autre serveur peut rester derrière NAT.

### Connexion intermittente

Si la connexion se perd régulièrement :
- Vérifiez `PersistentKeepalive = 25` dans wg0.conf
- Augmentez-le si nécessaire (ex: `PersistentKeepalive = 60`)
- Vérifiez que votre routeur ne bloque pas les connexions UDP longues

---

**Dernière mise à jour** : 2025-10-19
