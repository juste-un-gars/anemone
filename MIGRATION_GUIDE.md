# 🔄 Guide de migration - Configuration sécurisée

Ce guide explique comment mettre à jour votre projet Anemone avec le nouveau système de configuration sécurisée.

## 📋 Changements récents

### Version actuelle : Allocation automatique de subnet

**Changement important** : Anemone n'utilise plus de subnet Docker fixe pour éviter les conflits "Address already in use".

**Fichiers modifiés** :
1. **docker-compose.yml**
   - Suppression de l'attribut obsolète `version: '3.8'`
   - Suppression de l'IP statique WireGuard (`ipv4_address: 172.XX.0.2`)
   - Suppression du subnet fixe (`subnet: 172.XX.0.0/16`)
   - Docker choisit automatiquement un subnet libre

2. **services/api/main.py**
   - Interface de setup avec génération/restauration de clé
   - Middleware de redirection automatique
   - Chiffrement via Python cryptography (plus d'openssl)

3. **services/restic/decrypt_key.py**
   - Script Python pour déchiffrement (utilise HOSTNAME au lieu d'UUID)

4. **TROUBLESHOOTING.md**
   - Documentation du problème réseau et solution

## 🚀 Procédure de migration

### Étape 1 : Sauvegarde

```bash
# Sauvegarder votre configuration actuelle
cp -r config config.backup
cp docker-compose.yml docker-compose.yml.backup

# Si vous avez déjà une clé Restic, sauvegardez-la !
if [ -f config/restic-password ]; then
    cp config/restic-password ~/restic-key-backup.txt
    echo "⚠️  CLÉ SAUVEGARDÉE DANS ~/restic-key-backup.txt"
fi
```

### Étape 2 : Arrêter les services

```bash
docker-compose down
```

### Étape 3 : Mettre à jour les fichiers

```bash
# Remplacer les fichiers modifiés
# (copiez-collez depuis les artefacts fournis)

# services/api/main.py
# services/api/requirements.txt
# services/restic/entrypoint.sh
# scripts/init.sh
# scripts/restore.sh
# .gitignore
```

### Étape 4 : Reconstruire les images

```bash
# Forcer la reconstruction des images Docker
docker-compose build --no-cache
```

### Étape 5 : Démarrer et configurer

```bash
# Démarrer les services
docker-compose up -d

# Attendre que les services démarrent (30 secondes)
sleep 30

# Ouvrir l'interface de setup
echo "Ouvrez http://localhost:3000/setup dans votre navigateur"
```

### Étape 6 : Configuration selon votre cas

#### CAS A : Nouvelle installation (pas de backups existants)

1. Accédez à `http://localhost:3000/setup`
2. Choisissez **"Nouveau serveur"**
3. **SAUVEGARDEZ LA CLÉ** dans Bitwarden immédiatement
4. Cochez la confirmation
5. Validez

#### CAS B : Migration (vous avez déjà des backups)

1. Récupérez votre ancienne clé :
   ```bash
   cat ~/restic-key-backup.txt
   # Ou depuis Bitwarden si vous l'aviez déjà sauvegardée
   ```

2. Accédez à `http://localhost:3000/setup`
3. Choisissez **"Restauration"**
4. Collez votre clé
5. Validez

### Étape 7 : Vérification

```bash
# Vérifier que tout fonctionne
docker-compose ps

# Vérifier les logs
docker-compose logs -f restic

# Tester un backup manuel
docker exec anemone-restic /scripts/backup-now.sh

# Vérifier le dashboard
curl http://localhost:3000/health
```

## ⚠️ Points d'attention

### Si vous perdez la clé pendant la migration

Si vous aviez des backups mais avez perdu la clé :

```bash
# Vérifier si l'ancienne clé existe encore
cat config.backup/restic-password

# Si oui, utilisez-la pour la restauration dans le setup web
```

### Si le setup ne s'affiche pas

```bash
# Vérifier les logs de l'API
docker logs anemone-api

# Vérifier que le port 3000 est accessible
curl http://localhost:3000/setup

# Redémarrer l'API si nécessaire
docker-compose restart api
```

### Si vous ne pouvez pas accéder au setup web

```bash
# Alternative : configurer manuellement (DÉCONSEILLÉ)
# Générer une clé
openssl rand -base64 32 > /tmp/temp-key.txt

# Lancer un conteneur temporaire pour chiffrer
docker run --rm -v $(pwd)/config:/config -v /tmp:/tmp alpine sh -c '
  apk add --no-cache openssl
  SYSTEM_KEY=$(cat /proc/sys/kernel/random/uuid)
  SALT=$(openssl rand -hex 32)
  echo "$SALT" > /config/.restic.salt
  cat /tmp/temp-key.txt | openssl enc -aes-256-cbc \
    -pbkdf2 -iter 100000 \
    -pass pass:"$SYSTEM_KEY:$SALT" \
    -out /config/.restic.encrypted
  touch /config/.setup-completed
'

# IMPORTANT : Sauvegarder la clé
cat /tmp/temp-key.txt
# Copier dans Bitwarden MAINTENANT

# Nettoyer
rm /tmp/temp-key.txt

# Redémarrer
docker-compose restart
```

## 🔍 Diagnostic des problèmes

### Le service Restic ne démarre pas

```bash
# Voir les logs
docker logs anemone-restic

# Erreur probable : "Failed to decrypt Restic key"
# Solution : Refaire le setup via l'interface web
docker-compose restart api
# Puis supprimer le marqueur et refaire le setup
docker exec anemone-api rm /config/.setup-completed
```

### La clé ne se déchiffre pas

```bash
# Vérifier la présence des fichiers
ls -la config/.restic*

# Vous devriez voir :
# config/.restic.encrypted
# config/.restic.salt
# config/.setup-completed

# Si un fichier manque, refaire le setup
```

### Impossible de restaurer les anciens backups

```bash
# Vérifier que vous utilisez la bonne clé
# La clé doit être EXACTEMENT la même que celle utilisée pour créer les backups

# Tester manuellement
docker exec -it anemone-restic sh
export RESTIC_PASSWORD="votre-ancienne-clé"
restic -r sftp:user@host:/path snapshots

# Si ça fonctionne, c'est que la clé est bonne
# Refaites le setup web avec cette clé
```

## ✅ Validation post-migration

Checklist pour vérifier que tout fonctionne :

```bash
# 1. Setup complété
[ -f config/.setup-completed ] && echo "✅ Setup OK" || echo "❌ Setup manquant"

# 2. Clé chiffrée présente
[ -f config/.restic.encrypted ] && echo "✅ Clé chiffrée OK" || echo "❌ Clé manquante"

# 3. Ancienne clé supprimée
[ ! -f config/restic-password ] && echo "✅ Ancienne clé supprimée" || echo "⚠️  Ancienne clé encore présente"

# 4. Services actifs
docker-compose ps | grep -q "Up" && echo "✅ Services actifs" || echo "❌ Services arrêtés"

# 5. API accessible
curl -s http://localhost:3000/health | grep -q "healthy" && echo "✅ API OK" || echo "❌ API KO"

# 6. Restic peut déchiffrer
docker exec anemone-restic sh -c 'echo $RESTIC_PASSWORD' | grep -q . && echo "✅ Restic OK" || echo "❌ Restic KO"

# 7. Dashboard accessible
curl -s http://localhost:3000/ | grep -q "Anemone" && echo "✅ Dashboard OK" || echo "❌ Dashboard KO"
```

## 🆘 Support

Si vous rencontrez des problèmes :

1. Consultez les logs : `docker-compose logs -f`
2. Vérifiez la checklist ci-dessus
3. Ouvrez une issue sur GitHub avec :
   - Les logs complets
   - Votre configuration (sans les secrets)
   - Les étapes que vous avez suivies

## 📝 Rollback (retour arrière)

Si la migration ne fonctionne pas et que vous voulez revenir en arrière :

```bash
# Arrêter tout
docker-compose down

# Restaurer les backups
cp -r config.backup/* config/
cp docker-compose.yml.backup docker-compose.yml

# Redémarrer avec l'ancienne version
docker-compose up -d
```

---

**Bonne migration ! N'oubliez pas de sauvegarder votre clé dans Bitwarden ! 🔐**
