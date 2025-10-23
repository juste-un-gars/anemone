# 🔧 Guide de dépannage Anemone

## Erreur : "Address already in use" au démarrage de WireGuard

### Symptôme
```
Error response from daemon: failed to set up container networking: Address already in use
```

### Cause
Cette erreur apparaissait dans les anciennes versions d'Anemone qui spécifiaient un subnet fixe (172.20.0.0/16).

### Solution

**Anemone utilise maintenant l'allocation automatique de subnet par Docker**. Ce problème ne devrait plus se produire.

Si vous rencontrez toujours cette erreur :

```bash
# 1. Nettoyer complètement les réseaux Docker
docker compose down
docker network prune -f

# 2. Redémarrer
docker compose up -d
```

Si le problème persiste, c'est qu'un réseau Docker du même nom existe déjà :

```bash
# Lister tous les réseaux
docker network ls

# Si vous voyez "anemone_anemone-net" ou similaire, supprimez-le
docker network rm anemone_anemone-net

# Puis redémarrer
docker compose up -d
```

**Note pour les anciennes installations** : Si vous migrez depuis une version antérieure avec un subnet fixe, le docker-compose.yml a été simplifié. Docker choisit automatiquement un subnet libre.

---

## Erreur : "Erreur lors du chiffrement" lors du setup

### Symptôme
Après avoir cliqué sur "Continuer" lors du setup (nouveau serveur ou restauration), vous obtenez une erreur HTTP 500 avec le message "Erreur lors du chiffrement".

### Cause
Le volume `/config` est monté en lecture seule pour le service API, ce qui l'empêche de créer les fichiers nécessaires.

### Solution

1. **Arrêter les services** :
   ```bash
   docker-compose down
   ```

2. **Vérifier le docker-compose.yml** :

   Ouvrir le fichier `docker-compose.yml` et vérifier la section du service `api` :

   ```yaml
   api:
     volumes:
       - ./config:/config        # ✅ CORRECT (lecture/écriture)
       # PAS
       - ./config:/config:ro     # ❌ INCORRECT (lecture seule)
   ```

3. **Vérifier les permissions du dossier config** :
   ```bash
   ls -ld config
   # Doit afficher quelque chose comme : drwxr-xr-x ... user group ... config

   # Si nécessaire, corriger les permissions :
   chmod 755 config
   ```

4. **Redémarrer les services** :
   ```bash
   docker-compose up -d
   ```

5. **Consulter les logs pour diagnostic détaillé** :
   ```bash
   docker logs anemone-api -f
   ```

   Vous devriez voir des messages de debug comme :
   - `DEBUG: System key obtained`
   - `DEBUG: Salt generated`
   - `DEBUG: Key derived`
   - etc.

6. **Refaire le setup** :
   - Accédez à `http://localhost:3000/setup`
   - Suivez la procédure normalement

### Diagnostic approfondi

Si le problème persiste, vérifiez :

**1. Permissions d'écriture dans le conteneur** :
```bash
docker exec -it anemone-api sh -c "touch /config/.test && rm /config/.test && echo 'OK' || echo 'ERREUR'"
```

**2. Espace disque disponible** :
```bash
df -h
```

**3. Logs détaillés** :
```bash
docker logs anemone-api 2>&1 | grep -E "ERROR|DEBUG"
```

Les messages d'erreur possibles :
- `ERROR: Config directory does not exist` → Le dossier `/config` n'est pas monté
- `ERROR: Cannot write to config directory` → Problème de permissions
- `ERROR encrypting key: ...` → Erreur de chiffrement, voir le traceback

## Erreur : Le service Restic ne démarre pas

### Symptôme
```
❌ Setup not completed
   Please access http://localhost:3000/setup
```

### Solution
Le setup n'a pas été complété. Accédez à l'interface web et complétez le setup.

---

### Symptôme
```
❌ Failed to decrypt key
```

### ⚠️ Cause n°1 : UUID vs HOSTNAME (TRÈS IMPORTANT)

**Problème critique** : Si votre `get_system_key()` utilise `/proc/sys/kernel/random/uuid`, la clé système **change à chaque redémarrage du conteneur**, rendant le déchiffrement impossible !

**Solution** : Vérifier que le code utilise bien `HOSTNAME` :

```bash
# Vérifier main.py
grep -A3 "def get_system_key" services/api/main.py

# Doit afficher :
# def get_system_key() -> str:
#     # IMPORTANT : Utiliser le HOSTNAME...
#     return os.getenv('HOSTNAME', 'anemone')
```

Si le code utilise encore UUID, corrigez :

```python
# services/api/main.py
def get_system_key() -> str:
    # IMPORTANT : Utiliser le HOSTNAME (fixe et persistant)
    return os.getenv('HOSTNAME', 'anemone')
```

Puis reconstruisez :
```bash
docker-compose build --no-cache api
docker-compose down
rm config/.setup-completed config/.restic.*
docker-compose up -d
# Refaire le setup
```

### Causes possibles (autres)

1. **Setup incomplet ou fichiers manquants** :
   ```bash
   ls -la config/.restic*
   # Doit afficher :
   # config/.restic.encrypted
   # config/.restic.salt
   # config/.setup-completed
   ```

2. **Clé utilisée différente** :
   Si vous avez restauré avec une mauvaise clé, refaites le setup :
   ```bash
   rm config/.setup-completed config/.restic.*
   docker-compose restart api
   # Refaire le setup avec la bonne clé
   ```

3. **Migration depuis ancienne version** :
   Si vous aviez `config/restic-password` en clair :
   ```bash
   # Sauvegarder l'ancienne clé
   cp config/restic-password ~/restic-key-backup.txt

   # Refaire le setup en mode "Restauration"
   rm config/.setup-completed
   docker-compose restart api
   # Accéder au setup et coller l'ancienne clé
   ```

## Erreur : Page de setup inaccessible

### Symptôme
La page `/setup` redirige vers `/` ou vice-versa

### Diagnostic

```bash
# Vérifier l'état du setup
ls -la config/.setup-completed

# Si le fichier existe mais vous voulez refaire le setup :
rm config/.setup-completed
docker-compose restart api
```

## Erreur : "cryptography" module not found

### Symptôme
```
ModuleNotFoundError: No module named 'cryptography'
```

### Solution

Le module n'a pas été installé. Reconstruire l'image :

```bash
docker-compose build --no-cache api
docker-compose up -d api
```

Vérifier que `requirements.txt` contient bien :
```
cryptography==41.0.7
```

## Erreur : Permission denied sur /proc/sys/kernel/random/uuid

### Symptôme
La fonction `get_system_key()` échoue

### Solution
Le système utilise automatiquement un fallback (HOSTNAME). C'est normal sur certains systèmes. Aucune action requise.

Si vous voulez forcer un système spécifique, définissez la variable d'environnement :
```yaml
# docker-compose.yml
api:
  environment:
    - HOSTNAME=mon-serveur-unique
```

## Problèmes avec la corbeille

### La corbeille ne se vide pas automatiquement

**Vérifier le cron** :
```bash
docker exec anemone-shares crontab -l
# Doit afficher : 0 * * * * /scripts/trash-cleanup.sh
```

**Vérifier crond** :
```bash
docker exec anemone-shares ps aux | grep crond
```

**Tester manuellement** :
```bash
docker exec anemone-shares /scripts/trash-cleanup.sh
```

### Les fichiers supprimés ne vont pas dans la corbeille

**Vérifier la configuration Samba** :
```bash
docker exec anemone-shares cat /etc/samba/smb.conf | grep -A 5 "vfs objects"
# Doit afficher : vfs objects = recycle
```

**Vérifier que le répertoire existe** :
```bash
docker exec anemone-shares ls -la /mnt/backup/.trash
```

### La corbeille est pleine mais rien ne se supprime

**Augmenter la taille limite** :

Éditez `services/shares/scripts/trash-cleanup.sh` :
```bash
MAX_SIZE_GB=${TRASH_MAX_SIZE_GB:-20}  # au lieu de 10
```

Puis redémarrez :
```bash
docker-compose restart shares
```

---

## Problèmes de restauration depuis peer

### Aucun peer disponible pour restauration

**Vérifier la connectivité VPN** :
```bash
docker exec anemone-core wg show
docker exec anemone-core ping 10.8.0.2  # IP du peer
```

**Vérifier SSH** :
```bash
docker exec anemone-core ssh -o ConnectTimeout=5 restic@10.8.0.2 echo "OK"
```

### La restauration échoue avec "Permission denied"

**Vérifier les clés SSH** :
```bash
# Sur le serveur source
docker exec anemone-core cat /root/.ssh/id_rsa.pub

# Sur le serveur destination (doit contenir la clé ci-dessus)
cat config/ssh/authorized_keys
```

### La restauration est très lente

C'est normal si vous avez beaucoup de données. Utilisez d'abord le **mode simulation** pour estimer la durée.

---

## Erreur : Permission denied lors de la synchronisation rsync/restic

### Symptôme
```
rsync: mkdir "/backups/SERVER" failed: No such file or directory (2)
# ou
Fatal: unable to open repository: MkdirAll /backups/SERVER: permission denied
```

### Cause
L'utilisateur `restic` n'a accès qu'à son répertoire home (`/home/restic/`). Les chemins de destination doivent être **relatifs** et non absolus.

### Solution

**Vérifier votre configuration** dans `config/config.yaml` :

❌ **INCORRECT** (chemin absolu) :
```yaml
backup:
  targets:
    - name: "peer-backup"
      host: "10.8.0.2"
      port: 22222
      user: "restic"
      path: "/backups/myserver"   # ❌ Absolu - ne fonctionne pas
```

✅ **CORRECT** (chemin relatif) :
```yaml
backup:
  targets:
    - name: "peer-backup"
      host: "10.8.0.2"
      port: 22222
      user: "restic"
      path: "backups/myserver"    # ✅ Relatif - sera /home/restic/backups/myserver
```

**Corriger la configuration** :
```bash
# 1. Éditer config.yaml
nano config/config.yaml

# 2. Enlever le / initial du path (dans la section backup.targets)

# 3. Redémarrer le conteneur core
docker compose restart core

# 4. Tester
docker exec anemone-core /scripts/sync-now.sh
```

**Note** : Si vous ajoutez des pairs via l'interface web, les chemins sont automatiquement créés en relatif (depuis la version récente).

---

## Problèmes avec les configurations de pairs

### Aucune configuration de pair visible

**Vérifier que les backups sont reçus** :
```bash
ls -la config-backups/peer-configs/
```

Si vide, vérifier sur un pair :
```bash
# Sur l'autre serveur
docker logs anemone-core 2>&1 | grep "backup-config-auto"
```

### Impossible de télécharger un backup de pair

**Vérifier les permissions** :
```bash
docker exec anemone-api ls -la /config-backups/peer-configs/
```

**Vérifier le volume mount** :
```bash
docker inspect anemone-api | grep -A 5 "Mounts"
# Doit afficher /config-backups
```

---

## Problèmes avec les scripts de démarrage

### ./fr_start.sh : Permission denied

```bash
chmod +x fr_start.sh en_start.sh fr_restore.sh en_restore.sh
```

### Script de restauration : "Échec du déchiffrement"

**Causes possibles** :

1. **Mauvaise clé** : Vérifiez dans Bitwarden
2. **Fichier corrompu** : Retéléchargez depuis le peer
3. **Mauvais format** : Le fichier doit être .enc

**Test de validation** :
```bash
file backup-SERVER-DATE.enc
# Doit afficher : data (fichier binaire)
```

### Docker Compose non trouvé

Le script cherche `docker-compose` ou `docker compose`.

**Installation** :
```bash
# Sur Ubuntu/Debian
sudo apt install docker-compose

# Ou utiliser le plugin Docker
docker compose version
```

---

## Besoin d'aide supplémentaire

1. **Logs complets** :
   ```bash
   docker-compose logs > anemone-logs.txt
   ```

2. **État des services** :
   ```bash
   docker-compose ps
   ```

3. **Informations système** :
   ```bash
   docker version
   docker-compose version
   uname -a
   df -h
   ```

4. **Ouvrir une issue** sur GitHub avec :
   - Description du problème
   - Logs (en masquant les informations sensibles)
   - Configuration (sans les secrets)
   - Commandes exécutées

## Commandes utiles de dépannage

```bash
# Tout redémarrer proprement
docker-compose down
docker-compose up -d

# Reconstruire tout depuis zéro
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Logs en temps réel
docker-compose logs -f api

# Shell dans le conteneur API
docker exec -it anemone-api sh

# Vérifier les fichiers de config
ls -la config/

# Test de chiffrement/déchiffrement
docker exec anemone-api python3 -c "
from cryptography.hazmat.primitives import hashes
print('Cryptography OK')
"
```
