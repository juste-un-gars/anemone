# Correction : Génération des clés WireGuard - 2025-10-18

## 🐛 Problème rencontré

Lors de l'exécution de `start.sh` sur un serveur de test sans l'outil `wg` installé, l'erreur suivante se produisait :

```
RTNETLINK answers: Operation not permitted
```

**Cause** : Le conteneur Docker `linuxserver/wireguard` essayait de configurer des interfaces réseau au démarrage, ce qui nécessite des privilèges élevés et échouait lors de la simple génération de clés.

## ✅ Solution implémentée

### 1. Modification de `scripts/init.sh`

Ajout d'un mécanisme de fallback robuste pour la génération des clés WireGuard :

**Ordre de priorité** :
1. **wg (natif)** : Si `wg` est installé sur le host → utilisation directe
2. **Docker avec entrypoint** : Contournement de l'entrypoint par défaut avec `--entrypoint wg`
3. **Python fallback** : Si Docker échoue → génération avec Python (toujours disponible)

**Code modifié** (lignes 44-63 de `init.sh`) :
```bash
docker run --rm --entrypoint wg linuxserver/wireguard:latest genkey > ... || {
    # Fallback Python
    python3 -c "import base64, os; print(base64.b64encode(os.urandom(32)).decode())" > ...
}
```

### 2. Script d'extraction de clé publique

**Nouveau fichier** : `scripts/extract-wireguard-pubkey.sh`

Permet d'extraire la clé publique WireGuard depuis le conteneur en cours d'exécution.

**Usage** :
```bash
./scripts/extract-wireguard-pubkey.sh
```

### 3. Extraction automatique dans `start.sh`

Après le démarrage du conteneur WireGuard, le script `start.sh` vérifie automatiquement si la clé publique est un placeholder et l'extrait du conteneur.

**Code ajouté** (lignes 289-307 de `start.sh`).

### 4. Amélioration de `scripts/show-keys.sh`

Le script détecte maintenant si la clé publique est un placeholder et propose de l'extraire automatiquement depuis le conteneur en cours d'exécution.

## 🧪 Tests

### Test 1 : Installation propre (sans `wg` sur le host)

```bash
# 1. Clone sur un serveur test
git clone https://github.com/votre-repo/anemone.git
cd anemone

# 2. Lancer start.sh
bash start.sh

# Résultat attendu :
# - Clé privée générée avec Python
# - Clé publique extraite automatiquement après démarrage WireGuard
# - Pas d'erreur RTNETLINK
```

### Test 2 : Extraction manuelle de la clé publique

```bash
# Si la clé publique n'a pas été extraite automatiquement
./scripts/extract-wireguard-pubkey.sh

# Résultat attendu :
# ✅ Clé publique extraite et sauvegardée
# Clé publique WireGuard : AbCdEfG...
```

### Test 3 : Affichage des clés

```bash
./scripts/show-keys.sh

# Résultat attendu :
# - Si placeholder détecté → extraction automatique
# - Affichage de la clé publique WireGuard
# - Affichage de la clé publique SSH
```

## 📁 Fichiers modifiés

```
scripts/init.sh                        ✏️ Fallback Python pour génération clés
scripts/extract-wireguard-pubkey.sh    ✨ Nouveau
scripts/show-keys.sh                   ✏️ Détection placeholder + extraction auto
start.sh                               ✏️ Extraction auto après démarrage
WIREGUARD_KEY_FIX.md                   ✨ Nouveau (ce fichier)
```

## 🔍 Diagnostic rapide

### Vérifier que la clé privée est générée

```bash
cat config/wireguard/private.key
# Doit afficher une chaîne base64 de ~44 caractères
```

### Vérifier que la clé publique est valide

```bash
cat config/wireguard/public.key
# Ne doit PAS contenir "# Clé publique sera générée"
# Doit afficher une chaîne base64 de ~44 caractères
```

### Extraire manuellement la clé publique

```bash
docker exec anemone-wireguard sh -c "cat /config/wireguard/private.key | wg pubkey"
```

## 🎯 Compatibilité

La correction garantit que les clés WireGuard peuvent être générées sur **tous les environnements** :

- ✅ Serveurs avec `wg` installé
- ✅ Serveurs sans `wg` mais avec Docker
- ✅ Serveurs avec Docker en mode rootless (restrictions réseau)
- ✅ Conteneurs Docker (CI/CD)
- ✅ WSL/WSL2 (Windows)

## 🔐 Sécurité

- Les clés privées restent dans `config/wireguard/private.key` (permissions 600)
- Les clés publiques dans `config/wireguard/public.key` (permissions 644)
- Aucune clé n'est loggée ou affichée sauf via `show-keys.sh`
- Le fallback Python utilise `os.urandom()` (cryptographiquement sécurisé)

## 📝 Notes importantes

1. **La clé publique peut être régénérée à tout moment** depuis la clé privée sans impact
2. **Ne partagez JAMAIS la clé privée** (`private.key`)
3. **La clé publique est nécessaire** pour ajouter ce serveur comme peer chez d'autres
4. **Si vous régénérez les clés**, tous vos pairs devront mettre à jour leur configuration

## ✨ Améliorations futures possibles

- [ ] Ajouter un test automatisé qui vérifie la génération de clés
- [ ] Créer un script de validation qui teste tous les fallbacks
- [ ] Ajouter une option `--force-regenerate` pour regénérer les clés
- [ ] Implémenter la génération de clés Curve25519 pure en Python (sans dépendance `wg`)
