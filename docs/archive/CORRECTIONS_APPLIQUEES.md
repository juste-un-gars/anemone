# ✅ Corrections appliquées suite à la discussion

Ce document résume toutes les corrections importantes apportées au projet Anemone suite aux problèmes identifiés pendant la phase de test.

## 🔴 Problème #1 : Volume `/config` en lecture seule

### Symptôme
```
Encryption error: Can't open "/config/.restic.encrypted" for writing, Read-only file system
```

### Cause
Le volume `/config` était monté en lecture seule (`:ro`) pour le service API dans `docker-compose.yml`.

### Correction
**Fichier** : `docker-compose.yml`

```yaml
# AVANT (❌ incorrect)
api:
  volumes:
    - ./config:/config:ro    # Read-only

# APRÈS (✅ correct)
api:
  volumes:
    - ./config:/config       # Read-write
```

---

## 🔴 Problème #2 : Migration OpenSSL → Python Cryptography

### Symptôme
```
Error encrypting key: [Errno 2] No such file or directory: 'openssl'
```

### Cause
Le projet utilisait des appels subprocess à `openssl` qui n'était pas installé dans les conteneurs.

### Correction

**Fichiers modifiés** :
- `services/api/requirements.txt` : Ajout de `cryptography==41.0.7`
- `services/api/main.py` : Remplacement des subprocess openssl par cryptography
- `services/api/setup.py` : Remplacement des subprocess openssl par cryptography
- `services/restic/decrypt_key.py` : Nouveau script Python pour déchiffrement
- `services/restic/entrypoint.sh` : Utilise le script Python au lieu de openssl
- `services/restic/Dockerfile` : Ajout de cryptography dans pip install
- `scripts/init_script.sh` : Remplacement de `openssl rand` par Python secrets

**Algorithme utilisé** :
- Dérivation de clé : PBKDF2-HMAC-SHA256 (100,000 itérations)
- Chiffrement : AES-256-CBC
- Padding : PKCS7
- Format : IV (16 bytes) + données chiffrées

---

## 🔴 Problème #3 : UUID vs HOSTNAME (CRITIQUE)

### Symptôme
Le setup fonctionne initialement, mais après redémarrage du conteneur :
```
❌ Failed to decrypt key
```

### Cause
La clé système utilisait `/proc/sys/kernel/random/uuid` qui **change à chaque redémarrage du conteneur Docker**.

### Explication technique
```python
# ❌ MAUVAIS - UUID change à chaque redémarrage
with open('/proc/sys/kernel/random/uuid') as f:
    system_key = f.read().strip()

# Le chiffrement utilise : PBKDF2(password=UUID + salt)
# Au redémarrage : nouveau UUID → impossible de dériver la même clé → échec déchiffrement
```

### Correction
**Fichiers modifiés** :
- `services/api/main.py:get_system_key()`
- `services/api/setup.py:encrypt_restic_key()`
- `services/restic/decrypt_key.py:get_system_key()`

```python
# ✅ CORRECT - HOSTNAME persiste à travers les redémarrages
def get_system_key() -> str:
    # IMPORTANT : Utiliser le HOSTNAME (fixe et persistant) au lieu de UUID
    # L'UUID change à chaque redémarrage du conteneur
    return os.getenv('HOSTNAME', 'anemone')
```

**Pourquoi HOSTNAME fonctionne** :
- Docker définit la variable d'environnement `HOSTNAME` basée sur le nom du conteneur
- Le nom du conteneur est défini dans `docker-compose.yml` : `container_name: anemone-api`
- Cette valeur reste identique même après `docker-compose restart`

---

## 🔴 Problème #4 : Conflit réseau Docker

### Symptôme
```
Error response from daemon: failed to set up container networking: Address already in use
```

### Cause
Plusieurs projets Docker sur la même machine utilisaient le même subnet (172.20.0.0/16).

### Correction
**Fichier** : `docker-compose.yml`

```yaml
# Subnet changé pour éviter les conflits
networks:
  anemone-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.30.0.0/16  # Au lieu de 172.20.0.0/16

wireguard:
  networks:
    anemone-net:
      ipv4_address: 172.30.0.2  # Au lieu de 172.20.0.2
```

---

## 🟢 Améliorations #1 : Logging amélioré

### Ajout
Messages de debug détaillés dans `encrypt_restic_key()` pour faciliter le diagnostic :

```python
print(f"DEBUG: System key obtained (length: {len(system_key)})", flush=True)
print(f"DEBUG: Salt generated", flush=True)
print(f"DEBUG: Key derived", flush=True)
print(f"DEBUG: Cipher initialized", flush=True)
print(f"DEBUG: Key padded (length: {len(padded_key)})", flush=True)
print(f"DEBUG: Encryption complete", flush=True)
print(f"DEBUG: Encrypted key saved to {RESTIC_ENCRYPTED}", flush=True)
```

Avec traceback complet en cas d'erreur :
```python
except Exception as e:
    import traceback
    print(f"ERROR encrypting key: {e}", flush=True)
    print(f"Traceback: {traceback.format_exc()}", flush=True)
```

---

## 🟢 Améliorations #2 : Vérifications de permissions

### Ajout
Vérification explicite des permissions d'écriture avant chiffrement :

```python
# Vérifier que le dossier config existe
if not config_dir.exists():
    print(f"ERROR: Config directory does not exist", flush=True)
    return False

# Test d'écriture
try:
    test_file = config_dir / '.test_write'
    test_file.touch()
    test_file.unlink()
except Exception as e:
    print(f"ERROR: Cannot write to config directory: {e}", flush=True)
    return False
```

---

## 🟢 Améliorations #3 : Documentation

### Fichiers créés/mis à jour
1. **TROUBLESHOOTING.md** : Guide complet de dépannage
2. **CLAUDE.md** : Documentation pour Claude Code avec section "Common Pitfalls"
3. **CORRECTIONS_APPLIQUEES.md** : Ce fichier (historique des corrections)

---

## 📋 Checklist de migration pour installations existantes

Si vous avez une installation existante qui utilise encore l'ancienne version :

```bash
# 1. Sauvegarder la clé existante (si elle est en clair)
cp config/restic-password ~/restic-key-backup.txt

# 2. Arrêter les services
docker-compose down

# 3. Mettre à jour le code (git pull ou téléchargement)
git pull origin main

# 4. Supprimer les anciennes données de setup
rm config/.setup-completed config/.restic.* config/restic-password

# 5. Vérifier docker-compose.yml
# - Volume /config sans :ro pour l'API
# - Subnet réseau différent si conflit

# 6. Rebuild complet
docker-compose build --no-cache

# 7. Redémarrer
docker-compose up -d

# 8. Refaire le setup via http://localhost:3000/setup
# - Mode "Nouveau" : génère une nouvelle clé
# - Mode "Restauration" : coller l'ancienne clé depuis ~/restic-key-backup.txt

# 9. Vérifier que tout fonctionne
docker-compose ps
docker logs anemone-restic | grep "decrypted"
docker logs anemone-api | tail -20

# 10. Tester un redémarrage
docker-compose restart
sleep 10
docker logs anemone-restic | grep "decrypted"  # Doit afficher "✅ Restic key decrypted"
```

---

## ⚠️ Points d'attention pour le futur

1. **Ne JAMAIS utiliser UUID comme clé système** dans un conteneur Docker
2. **Toujours tester après un redémarrage** : `docker-compose restart`
3. **Vérifier les permissions** des volumes montés (éviter `:ro` quand écriture nécessaire)
4. **Utiliser Python cryptography** plutôt que des subprocess openssl
5. **Logger avec `flush=True`** pour voir les messages immédiatement dans Docker logs

---

## 🎯 État actuel du projet

✅ OpenSSL remplacé par Python cryptography
✅ Permissions d'écriture corrigées
✅ UUID remplacé par HOSTNAME
✅ Logging amélioré avec debug et tracebacks
✅ Documentation complète (TROUBLESHOOTING.md, CLAUDE.md)
✅ Setup web fonctionnel (mode nouveau + restauration)
✅ Déchiffrement persistant à travers les redémarrages

---

**Dernière mise à jour** : 2025-10-17
**Version** : Post-migration Python cryptography + corrections UUID/HOSTNAME
