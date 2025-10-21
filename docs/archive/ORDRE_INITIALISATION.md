# 🔧 Ordre d'initialisation d'Anemone

Ce document explique l'ordre correct des opérations lors de l'installation d'Anemone.

## 🎯 Vue d'ensemble

```
1. Cloner le dépôt
   ↓
2. Générer les clés (WireGuard + SSH)
   ↓
3. Créer config.yaml et .env
   ↓
4. Démarrer les conteneurs Docker
   ↓
5. Setup web (générer/restaurer clé Restic)
   ↓
6. Système prêt !
```

## 📋 Explication détaillée

### Étape 1 : Clonage

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
```

**État après** : Vous avez le code source mais aucune configuration personnalisée.

---

### Étape 2 : Génération des clés cryptographiques

**Option A - Script automatique (recommandé)** :
```bash
./start.sh
```

**Option B - Manuel** :
```bash
./scripts/init.sh
```

**Ce qui est créé** :
- `config/wireguard/private.key` - Clé privée WireGuard (SECRÈTE)
- `config/wireguard/public.key` - Clé publique WireGuard (à partager)
- `config/ssh/id_rsa` - Clé privée SSH (SECRÈTE)
- `config/ssh/id_rsa.pub` - Clé publique SSH (à partager)
- `config/ssh/authorized_keys` - Liste des pairs autorisés (vide au départ)
- `config/config.yaml` - Copie de config.yaml.example
- `.env` - Copie de .env.example

**⚠️ CRITIQUE** : Ces clés doivent exister AVANT le premier `docker compose up` car :
- WireGuard ne peut pas démarrer sans sa clé privée
- Le service SFTP a besoin des clés SSH
- Les pairs ne peuvent pas se connecter sans échanger les clés publiques

---

### Étape 3 : Configuration personnalisée

**Éditer `.env`** :
```bash
nano .env
```

Changez au minimum :
- `SMB_PASSWORD` - Mot de passe pour accès Samba
- `WEBDAV_PASSWORD` - Mot de passe pour accès WebDAV

**Éditer `config/config.yaml`** :
```bash
nano config/config.yaml
```

Personnalisez :
- `node.name` - Nom de votre serveur
- `wireguard.address` - IP VPN (ex: 10.8.0.1/24)
- `wireguard.public_endpoint` - Votre DNS dynamique (ex: alice.duckdns.org:51820)
- `peers` - Liste vide au départ, à remplir plus tard

**État après** : Configuration adaptée à votre environnement.

---

### Étape 4 : Démarrage des conteneurs

```bash
docker compose up -d
```

**Ce qui démarre** :
1. **WireGuard** - VPN (utilise config/wireguard/private.key)
2. **Samba** - Partage SMB (attend setup complet)
3. **WebDAV** - Partage web (attend setup complet)
4. **SFTP** - Réception backups (utilise config/ssh/authorized_keys)
5. **API** - Interface web (redirige vers /setup)
6. **Restic** - Backups (ATTEND le setup web, bloqué si pas fait)

**État après** : Conteneurs démarrés mais Restic en attente du setup.

---

### Étape 5 : Setup web (génération clé Restic)

Accédez à http://localhost:3000/setup

**Option "Nouveau serveur"** :
1. Génère une clé Restic aléatoire (32 caractères)
2. Affiche la clé (À SAUVEGARDER DANS BITWARDEN !)
3. Chiffre la clé avec AES-256-CBC + PBKDF2
4. Sauvegarde dans `/config/.restic.encrypted`
5. Crée le marqueur `/config/.setup-completed`

**Option "Restauration"** :
1. Vous collez votre clé Restic existante
2. Même processus de chiffrement
3. Permet de récupérer vos backups existants

**État après** :
- Clé Restic configurée et chiffrée
- Service Restic peut démarrer
- Dashboard accessible

---

### Étape 6 : Système opérationnel

**Vérification** :
```bash
# Tous les conteneurs tournent
docker compose ps

# Restic a déchiffré sa clé
docker logs anemone-restic | grep "Restic key decrypted"

# Dashboard accessible
curl http://localhost:3000/
```

**Prochaines étapes** :
- Ajouter des pairs (voir INTERCONNEXION_GUIDE.md)
- Configurer DNS dynamique
- Tester un backup manuel

---

## 🚨 Problèmes courants

### "WireGuard container failed to start"

**Cause** : Clés WireGuard manquantes

**Solution** :
```bash
docker compose down
./scripts/init.sh
docker compose up -d
```

---

### "Restic: Setup not completed"

**Cause** : Vous avez démarré Docker avant de faire le setup web

**Solution** :
1. Accédez à http://localhost:3000/setup
2. Complétez le setup
3. Restic redémarrera automatiquement

---

### "SFTP: Permission denied"

**Cause** : Clés SSH manquantes ou permissions incorrectes

**Solution** :
```bash
docker compose down
rm -rf config/ssh
./scripts/init.sh
docker compose up -d
```

---

### "Config file not found"

**Cause** : config.yaml n'a pas été créé

**Solution** :
```bash
cp config/config.yaml.example config/config.yaml
nano config/config.yaml  # Personnaliser
docker compose restart
```

---

## 🔄 Ordre de démarrage des conteneurs

Docker Compose respecte cet ordre (via `depends_on`) :

```
1. WireGuard (healthcheck: wg show)
   ↓
2. Samba, WebDAV (en parallèle)
   ↓
3. Restic (attend WireGuard healthy)
   ↓
4. API (attend tous les autres)
```

**Note** : Restic utilise `network_mode: "service:wireguard"` donc partage la stack réseau de WireGuard. Il ne peut pas démarrer avant que WireGuard soit opérationnel.

---

## 📊 Fichiers créés à chaque étape

| Étape | Fichiers créés | Obligatoire |
|-------|----------------|-------------|
| Clonage | Code source | ✅ |
| init.sh | `config/wireguard/*.key`, `config/ssh/*`, `config/config.yaml`, `.env` | ✅ |
| Édition manuelle | Modifications de config.yaml et .env | ⚠️ Recommandé |
| docker compose up | Volumes Docker, réseaux | ✅ |
| Setup web | `config/.restic.encrypted`, `config/.restic.salt`, `config/.setup-completed` | ✅ |

**Légende** :
- ✅ Obligatoire pour le fonctionnement
- ⚠️ Recommandé pour la sécurité

---

## 🎓 Résumé : Commande unique vs étape par étape

### Commande unique (débutant)

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./start.sh
# Suivre les instructions, éditer .env et config.yaml si demandé
# Accéder à http://localhost:3000/setup
```

### Étape par étape (contrôle total)

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./scripts/init.sh
nano .env
nano config/config.yaml
docker compose up -d
# Accéder à http://localhost:3000/setup
```

---

**Recommandation** : Utilisez `./start.sh` qui vérifie automatiquement que tout est en ordre avant de démarrer Docker.
