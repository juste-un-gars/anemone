# Correction du problème WireGuard - 2025-10-18

## 🐛 Problème identifié

Les utilisateurs devaient exécuter manuellement des commandes comme :
```bash
docker exec anemone-wireguard sh -c "echo 'jQugK9t3BDm29Bc/f9rnToASpXPTCAPAXDvheyjNUBE=' | wg pubkey" > config/wireguard/public.key
```

**Cause** : Le script `init.sh` générait des clés dans `config/wireguard/` mais le conteneur WireGuard les ignorait et générait ses propres clés ailleurs.

## ✅ Modifications apportées

### 1. `scripts/init.sh` - Génération de wg0.conf

**Avant** :
- Générait uniquement `config/wireguard/private.key` et `public.key`
- Ces fichiers n'étaient jamais utilisés par WireGuard

**Après** :
- Génère `config/wireguard/private.key` et `public.key` (pour sauvegarde/partage)
- **Crée `config/wg_confs/wg0.conf`** avec la clé privée générée
- Le conteneur WireGuard utilise directement ce fichier

**Lignes modifiées** : init.sh:31, init.sh:56-100

### 2. `scripts/add-peer.sh` - Ajout de peers

**Avant** :
- Modifiait uniquement `config/config.yaml`
- Ne touchait pas à la configuration WireGuard
- **Résultat** : Les peers n'étaient jamais ajoutés au VPN !

**Après** :
- Modifie `config/config.yaml` (pour Anemone)
- **Modifie `config/wg_confs/wg0.conf`** (pour WireGuard)
- Crée des backups automatiques de wg0.conf
- Affiche un rappel pour redémarrer WireGuard

**Lignes ajoutées** : add-peer.sh:94-113

### 3. `scripts/show-keys.sh` - Nouveau script

Script utilitaire pour afficher rapidement les clés publiques à partager avec les pairs.

Usage :
```bash
./scripts/show-keys.sh
```

### 4. Documentation

**Nouveau fichier** : `WIREGUARD_SETUP.md`

Guide complet expliquant :
- La structure des fichiers WireGuard
- Comment initialiser un nouveau serveur
- Comment ajouter des pairs (méthode automatique et manuelle)
- Comment vérifier que tout fonctionne
- Dépannage des problèmes courants
- Flux de travail pour connecter deux serveurs

## 🧪 Test de la correction

Pour vérifier que la correction fonctionne sur un nouveau serveur :

```bash
# 1. Clean install
rm -rf config/
./scripts/init.sh

# 2. Vérifier que wg0.conf existe et contient la clé
cat config/wg_confs/wg0.conf

# 3. Démarrer WireGuard
docker-compose up -d wireguard

# 4. Vérifier que WireGuard utilise la bonne clé
docker exec anemone-wireguard wg show
cat config/wireguard/public.key

# Les deux commandes ci-dessus doivent afficher la MÊME clé publique
```

## 📝 Migration pour installations existantes

Si vous avez déjà un serveur Anemone qui fonctionne avec des clés WireGuard custom :

### Option A : Réutiliser vos clés existantes

```bash
# 1. Sauvegarder vos clés actuelles
docker exec anemone-wireguard wg show dump > /tmp/wg_backup.txt

# 2. Extraire la clé privée (si possible)
# Sinon, garder les clés actuelles et recréer wg0.conf manuellement

# 3. Mettre à jour init.sh (déjà fait)

# 4. Créer wg0.conf avec vos clés existantes
# Éditer config/wg_confs/wg0.conf

# 5. Redémarrer
docker-compose restart wireguard
```

### Option B : Générer de nouvelles clés (recommandé si possible)

```bash
# 1. Sauvegarder la config actuelle
cp -r config config.backup

# 2. Régénérer
rm config/wireguard/*.key
./scripts/init.sh

# 3. Partager la nouvelle clé publique avec vos pairs
./scripts/show-keys.sh

# 4. Demander à vos pairs de mettre à jour leur configuration
```

## 🎯 Impact utilisateur

**Avant** :
- Configuration manuelle complexe
- Risque d'erreur élevé
- Clés désynchronisées
- Documentation confuse

**Après** :
- Un seul script : `./scripts/init.sh`
- Ajout de peer simplifié : `./scripts/add-peer.sh`
- Clés toujours synchronisées
- Documentation claire dans WIREGUARD_SETUP.md

## 🔒 Sécurité

Aucun impact négatif sur la sécurité :
- Les clés privées restent privées
- Le fichier `wg0.conf` est déjà dans `.gitignore`
- Permissions correctes (600 pour private.key et wg0.conf)

## 📌 Fichiers modifiés

```
scripts/init.sh                  # Modifié
scripts/add-peer.sh              # Modifié
scripts/show-keys.sh             # Créé
WIREGUARD_SETUP.md               # Créé
CHANGELOG_WIREGUARD_FIX.md       # Créé (ce fichier)
```

## ✨ Prochaines étapes recommandées

1. Tester la procédure complète sur un serveur de test
2. Mettre à jour `INTERCONNEXION_GUIDE.md` pour référencer `WIREGUARD_SETUP.md`
3. Ajouter un test automatisé qui vérifie la correspondance des clés
4. Envisager un script de migration pour les installations existantes
