# 🔐 Architecture WireGuard dans Anemone

## Vue d'ensemble

WireGuard crée un **réseau VPN mesh** (maillé) entre tous les nœuds Anemone, permettant aux différentes instances (chez vous, Alice, Bob, etc.) de communiquer de manière sécurisée comme si elles étaient sur le même réseau local.

## 🏗️ Architecture à deux niveaux

### Niveau 1 : Réseau Docker local (`anemone-net`)

Chaque machine exécute un docker-compose qui crée un réseau bridge local :

```
┌─────────────────────────────────────────────────┐
│  Machine locale (votre PC/serveur)              │
│                                                  │
│  Réseau Docker: 172.20.0.0/16 (anemone-net)    │
│  ┌──────────────────────────────────────────┐  │
│  │  anemone-wireguard    (172.20.0.2)       │  │
│  │  anemone-api          (172.20.0.x)       │  │
│  │  anemone-samba        (172.20.0.x)       │  │
│  │  anemone-webdav       (172.20.0.x)       │  │
│  │  anemone-sftp         (172.20.0.x)       │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

**Tous les conteneurs communiquent entre eux** sur ce réseau local Docker.

### Niveau 2 : Réseau VPN WireGuard (`wg0`)

WireGuard crée un **tunnel chiffré** entre les machines distantes :

```
┌─────────────────────┐         Internet         ┌─────────────────────┐
│  Vous (Home)        │    (chiffré WireGuard)   │  Alice              │
│  IP publique:       │◄────────────────────────►│  IP publique:       │
│  home.duckdns.org   │      UDP port 51820      │  alice.duckdns.org  │
│                     │                           │                     │
│  IP VPN: 10.8.0.1   │                           │  IP VPN: 10.8.0.2   │
└─────────────────────┘                           └─────────────────────┘
          │                                                    │
          └────────────────────────────────────────────────────┘
                        Réseau VPN: 10.8.0.0/24
```

## 🔑 Composants clés

### 1. Conteneur WireGuard

```yaml
wireguard:
  image: linuxserver/wireguard:latest
  cap_add:
    - NET_ADMIN        # Permissions pour modifier les interfaces réseau
    - SYS_MODULE       # Permissions pour charger le module kernel WireGuard
  ports:
    - "51820:51820/udp"  # Port UDP exposé sur Internet
  sysctls:
    - net.ipv4.ip_forward=1  # Active le routage IP (pour faire transiter le trafic)
```

**Rôle** :
- Crée l'interface VPN `wg0`
- Écoute sur le port UDP 51820
- Gère les connexions entrantes/sortantes des pairs
- Route le trafic entre le réseau local Docker et le VPN

### 2. Network Mode: `service:wireguard` (CRUCIAL!)

```yaml
restic:
  network_mode: "service:wireguard"  # ← LA CLÉ DE TOUT !
```

Cette ligne **magique** fait que :
- Le conteneur `restic` **partage la stack réseau du conteneur wireguard**
- Restic voit directement l'interface `wg0` comme si elle était locale
- Restic peut accéder aux IP du VPN (10.8.0.x) **sans passer par le réseau Docker**

### 3. Configuration des pairs

Dans `config/config.yaml` :

```yaml
peers:
  - name: "alice"
    endpoint: "alice.duckdns.org:51820"      # Où la joindre sur Internet
    public_key: "CLE_PUBLIQUE_WIREGUARD"     # Sa clé publique WireGuard
    allowed_ips: "10.8.0.2/32"               # Son IP dans le VPN
    persistent_keepalive: 25                  # Ping toutes les 25s (NAT traversal)
```

## 🔄 Flux de communication complet

### Scénario : Vous voulez sauvegarder vos données chez Alice

```
┌────────────────────────────────────────────────────────────────────┐
│  CHEZ VOUS (10.8.0.1)                                              │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  1. Conteneur Restic                                         │ │
│  │     Commande: restic backup -r sftp:restic@10.8.0.2:/backup │ │
│  │                                                               │ │
│  │  2. Partage la stack réseau avec WireGuard                  │ │
│  │     network_mode: "service:wireguard"                        │ │
│  │                                                               │ │
│  │     ↓ Trafic envoyé vers 10.8.0.2                           │ │
│  │                                                               │ │
│  │  3. Conteneur WireGuard (interface wg0)                     │ │
│  │     - Regarde sa table de routage                            │ │
│  │     - 10.8.0.2 → doit passer par le tunnel vers Alice       │ │
│  │     - Chiffre le trafic avec les clés WireGuard             │ │
│  │     - Envoie via UDP vers alice.duckdns.org:51820           │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
│                          ↓ Internet (chiffré)                      │
└────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ Paquet UDP chiffré
                                   ↓
┌────────────────────────────────────────────────────────────────────┐
│  CHEZ ALICE (10.8.0.2)                                             │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  1. Conteneur WireGuard (interface wg0)                      │ │
│  │     - Reçoit le paquet UDP sur port 51820                    │ │
│  │     - Vérifie la signature avec votre clé publique           │ │
│  │     - Déchiffre le trafic                                    │ │
│  │     - Route vers le conteneur SFTP sur réseau Docker local   │ │
│  │                                                               │ │
│  │  2. Conteneur SFTP (anemone-sftp)                           │ │
│  │     - Reçoit la connexion SFTP sur port 2222                 │ │
│  │     - Vérifie votre clé SSH publique                         │ │
│  │     - Stocke les données chiffrées dans /backups             │ │
│  └──────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

## 🎯 Double chiffrement !

**Important** : Les données sont chiffrées **deux fois** :

1. **Chiffrement Restic** : Les fichiers sont chiffrés avec votre clé Restic AVANT d'être envoyés
   - Alice ne peut **jamais** voir vos fichiers en clair
   - Même si elle a accès physique à son serveur

2. **Chiffrement WireGuard** : Le tunnel VPN chiffre la transmission
   - Protège contre l'espionnage sur Internet
   - Authentifie les deux parties

```
Fichier original
    ↓
[Chiffrement Restic] ← Votre clé secrète
    ↓
Blob chiffré #1
    ↓
[Chiffrement WireGuard] ← Clés échangées au setup
    ↓
Transmission Internet
    ↓
[Déchiffrement WireGuard] ← Chez Alice
    ↓
Blob chiffré #1 (toujours chiffré Restic!)
    ↓
Stockage chez Alice (elle ne peut PAS le déchiffrer)
```

## 🌐 Traversée de NAT (important!)

La plupart des box Internet utilisent le NAT, ce qui complique les connexions peer-to-peer.

### Problème
```
Vous (NAT) ←→ Internet ←→ Alice (NAT)
     ?                           ?
```

Les deux machines sont derrière un NAT, elles ne peuvent pas s'initier mutuellement une connexion.

### Solution : `persistent_keepalive: 25`

```yaml
peers:
  - name: "alice"
    persistent_keepalive: 25  # ← Envoie un ping toutes les 25 secondes
```

**Comment ça marche** :
1. Votre WireGuard envoie un petit paquet vers Alice toutes les 25 secondes
2. Cela crée une "ouverture" dans votre NAT
3. Alice peut maintenant répondre (le NAT reconnaît le flux)
4. La connexion reste active en permanence

**Sans keepalive** : Le NAT fermerait la connexion après quelques minutes d'inactivité.

## 🔧 Configuration manuelle de WireGuard

Les fichiers de configuration générés automatiquement :

### `/config/wireguard/wg0.conf` (exemple)

```ini
[Interface]
# Votre configuration locale
Address = 10.8.0.1/24
ListenPort = 51820
PrivateKey = VOTRE_CLE_PRIVEE
MTU = 1420

# Pair Alice
[Peer]
PublicKey = CLE_PUBLIQUE_ALICE
Endpoint = alice.duckdns.org:51820
AllowedIPs = 10.8.0.2/32
PersistentKeepalive = 25

# Pair Bob (si configuré)
[Peer]
PublicKey = CLE_PUBLIQUE_BOB
Endpoint = bob.noip.com:51820
AllowedIPs = 10.8.0.3/32
PersistentKeepalive = 25
```

## 🛠️ Commandes utiles

### Vérifier l'état de WireGuard

```bash
# Voir l'interface WireGuard et les pairs connectés
docker exec anemone-wireguard wg show

# Sortie attendue :
# interface: wg0
#   public key: VOTRE_CLE_PUBLIQUE
#   private key: (hidden)
#   listening port: 51820
#
# peer: CLE_PUBLIQUE_ALICE
#   endpoint: 1.2.3.4:51820
#   allowed ips: 10.8.0.2/32
#   latest handshake: 23 seconds ago    ← Connexion active!
#   transfer: 1.25 MiB received, 892 KiB sent
#   persistent keepalive: every 25 seconds
```

### Tester la connectivité VPN

```bash
# Depuis le conteneur Restic (qui partage la stack réseau WireGuard)
docker exec anemone-restic ping 10.8.0.2

# Si ça marche → VPN opérationnel
# Si timeout → Problème de connexion ou configuration
```

### Débugger les problèmes

```bash
# Logs WireGuard
docker logs anemone-wireguard

# Vérifier le port UDP
sudo netstat -ulnp | grep 51820

# Tester l'endpoint public (depuis une autre machine)
nc -u -v votre-nom.duckdns.org 51820
```

## 🎭 Comparaison avec d'autres approches

### Approche classique (ce que Anemone NE fait PAS)

```yaml
# ❌ Mauvaise approche : expose directement les services
restic:
  ports:
    - "2222:22"  # SFTP exposé sur Internet → DANGEREUX
```

**Problèmes** :
- Services exposés directement sur Internet
- Attaques brute-force sur SSH
- Pas de chiffrement du transport (sauf SSH)
- Difficile à gérer avec plusieurs pairs

### Approche Anemone (ce qu'on fait)

```yaml
# ✅ Bonne approche : tout passe par le VPN
wireguard:
  ports:
    - "51820:51820/udp"  # Seulement WireGuard exposé

restic:
  network_mode: "service:wireguard"  # Partage la stack réseau

sftp:
  # Pas de ports exposés sur Internet
  # Accessible uniquement via VPN (10.8.0.x)
```

**Avantages** :
- Un seul port UDP exposé (51820)
- Authentification cryptographique forte
- Chiffrement de tout le trafic
- Facilement extensible (ajouter des pairs)
- Fonctionne derrière NAT

## 📊 Schéma récapitulatif

```
┌─────────────────────────────────────────────────────────────┐
│  Votre machine                                              │
│                                                             │
│  ┌────────────────────────────────────────────────────┐    │
│  │  Réseau Docker Local (172.20.0.0/16)              │    │
│  │                                                     │    │
│  │  ┌──────────────┐  ┌────────────┐  ┌───────────┐ │    │
│  │  │   Samba      │  │  WebDAV    │  │   API     │ │    │
│  │  │ (partage)    │  │  (HTTP)    │  │  (3000)   │ │    │
│  │  └──────────────┘  └────────────┘  └───────────┘ │    │
│  │         ↕                 ↕              ↕         │    │
│  │  ═══════════════════════════════════════════════  │    │
│  │         ↕                 ↕              ↕         │    │
│  │  ┌───────────────────────────────────────────┐   │    │
│  │  │  WireGuard (wg0)                          │   │    │
│  │  │  IP VPN: 10.8.0.1                         │   │    │
│  │  │  Port: 51820/udp ─────────────────────────┼───┼────┤
│  │  └───────────────────────────────────────────┘   │    │
│  │         ↕ (network_mode: "service:wireguard")    │    │
│  │  ┌───────────────────────────────────────────┐   │    │
│  │  │  Restic (backup)                          │   │    │
│  │  │  Voit directement wg0 et 10.8.0.x         │   │    │
│  │  └───────────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Internet (UDP 51820, chiffré)
                            ↓
              ┌──────────────────────────────┐
              │  Autres nœuds Anemone        │
              │  Alice: 10.8.0.2             │
              │  Bob:   10.8.0.3             │
              │  ...                         │
              └──────────────────────────────┘
```

## 🔐 Sécurité du modèle

1. **Authentification** : Clés publiques WireGuard (cryptographie à courbes elliptiques)
2. **Chiffrement** : ChaCha20-Poly1305 ou AES-256-GCM (selon le CPU)
3. **Perfect Forward Secrecy** : Nouvelles clés de session régulièrement
4. **Protection anti-replay** : Numéros de séquence
5. **Minimisation de la surface d'attaque** : Un seul port UDP exposé

## 🎓 Résumé pour comprendre

**En une phrase** : WireGuard crée un réseau privé virtuel chiffré (comme un LAN) entre tous les nœuds Anemone distants, et le conteneur Restic "emprunte" la connexion réseau du conteneur WireGuard pour accéder directement à ce réseau VPN.

**Analogie** :
- WireGuard = Un tunnel privé entre les maisons
- network_mode = Restic prend sa voiture dans le garage de WireGuard pour emprunter le tunnel
- Les autres conteneurs (Samba, API) restent dans votre maison locale

---

**Prochaines étapes** : Consultez `scripts/add-peer.sh` pour voir comment ajouter un nouveau pair au réseau.
