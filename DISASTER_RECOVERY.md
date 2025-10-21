# 🆘 Guide de Disaster Recovery - Anemone

Ce guide explique comment **sauvegarder et restaurer** complètement un serveur Anemone en cas de panne matérielle, réinstallation, ou migration.

## 📦 Export de Configuration

### Via l'interface web (recommandé)

1. Accédez à votre serveur : `http://localhost:3000`
2. Téléchargez votre configuration : `http://localhost:3000/api/config/export`
3. Un fichier `anemone-backup-NOMSERVEUR-TIMESTAMP.enc` sera téléchargé

**Exemple :**
```
anemone-backup-FR1-20251021-095520.enc
```

### Ce qui est sauvegardé

Le fichier de backup contient **TOUTE** votre configuration :

- ✅ `config.yaml` - Configuration complète (peers, targets, quotas, etc.)
- ✅ Clés WireGuard (VPN) - `private.key`, `public.key`
- ✅ Clés SSH - `id_rsa`, `id_rsa.pub`, `authorized_keys`
- ✅ Clé Restic chiffrée - `.restic.encrypted` + `.restic.salt`

### Sécurité

- 🔒 Le fichier est **chiffré avec votre clé Restic** (AES-256-CBC)
- 🔒 Dérivation de clé avec PBKDF2-HMAC-SHA256 (100 000 itérations)
- 🔒 Sans votre clé Restic, le fichier est **impossible à déchiffrer**

⚠️ **IMPORTANT** : Conservez ce fichier ET votre clé Restic en lieu sûr !

---

## 🔄 Restauration Complète

### Scénario 1 : Réinstallation complète d'un serveur

Vous avez perdu un serveur et voulez le restaurer depuis un backup.

**Prérequis :**
- Fichier de backup : `anemone-backup-FR1-20251021-095520.enc`
- Votre clé Restic

**Étapes :**

```bash
# 1. Cloner Anemone sur le nouveau serveur
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# 2. Copier votre fichier de backup dans le répertoire
cp /chemin/vers/anemone-backup-FR1-20251021-095520.enc .

# 3. Restaurer avec start.sh
./start.sh --restore-from=anemone-backup-FR1-20251021-095520.enc

# Le script vous demandera votre clé Restic
# Entrez-la (saisie masquée)

# 4. Lancer Docker
docker compose up -d

# 5. Vérifier que tout fonctionne
docker compose ps
docker logs anemone-core --tail 30
```

**Résultat :**
- ✅ Configuration VPN restaurée (même IP VPN qu'avant)
- ✅ Clés SSH restaurées (les peers peuvent se connecter)
- ✅ Configuration Restic restaurée
- ✅ Liste des peers restaurée
- ✅ Les backups se synchronisent automatiquement depuis les autres serveurs

### Scénario 2 : Migration vers un nouveau serveur

Même procédure que le Scénario 1.

**Note :** L'adresse IP publique et le nom d'hôte peuvent changer, mais :
- L'**IP VPN** reste la même (10.8.0.X)
- Les **clés WireGuard** sont les mêmes
- Les **autres serveurs** reconnaissent automatiquement le serveur

⚠️ **À faire après migration :**
```bash
# Sur les AUTRES serveurs, mettre à jour l'endpoint public
# (si l'IP publique a changé)
# Via l'interface web : http://localhost:3000/peers
# Modifier l'endpoint du peer concerné
```

### Scénario 3 : Test de disaster recovery

**Recommandation :** Testez régulièrement votre procédure de restauration !

```bash
# 1. Créer un répertoire de test
mkdir ~/test-recovery
cd ~/test-recovery

# 2. Cloner Anemone
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# 3. Restaurer depuis votre backup
./start.sh --restore-from=/chemin/vers/backup.enc

# 4. Vérifier le contenu
cat config/config.yaml
ls -la config/wireguard/
ls -la config/ssh/

# 5. Nettoyer
cd ~
rm -rf ~/test-recovery
```

---

## 📝 Restauration Manuelle (sans start.sh)

Si vous préférez restaurer manuellement :

```bash
# 1. Déchiffrer et extraire
python3 scripts/restore-config.py backup.enc "votre-clé-restic"

# 2. Vérifier
ls -la config/

# 3. Lancer Docker
docker compose up -d
```

---

## 🔄 Récupération des Données

Après avoir restauré la **configuration**, vous pouvez récupérer vos **données** depuis les autres serveurs.

### Option 1 : Restauration automatique (via Restic)

Les backups se synchronisent automatiquement si le mode est configuré :

```bash
# Vérifier que les backups se synchronisent
docker logs anemone-core -f
```

### Option 2 : Restauration manuelle

```bash
# Lister les snapshots disponibles sur un peer
docker exec anemone-core bash -c '
  export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py) && \
  restic -r sftp:restic@10.8.0.2:/backups/FR1 snapshots
'

# Restaurer le dernier snapshot
docker exec anemone-core bash -c '
  export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py) && \
  restic -r sftp:restic@10.8.0.2:/backups/FR1 restore latest --target /mnt/backup
'

# Restaurer un snapshot spécifique
docker exec anemone-core bash -c '
  export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py) && \
  restic -r sftp:restic@10.8.0.2:/backups/FR1 restore 5642e33b --target /mnt/backup
'
```

---

## 🎯 Bonnes Pratiques

### 1. **Backups réguliers de la configuration**

Exportez votre configuration :
- ✅ **Après chaque ajout de peer**
- ✅ **Après chaque modification importante de config.yaml**
- ✅ **Au moins une fois par mois**

```bash
# Via curl (automatisable dans un cron)
curl -o "backup-$(date +%Y%m%d).enc" http://localhost:3000/api/config/export
```

### 2. **Stockage sécurisé**

Conservez vos backups de configuration :
- 📁 Sur un NAS ou serveur de fichiers séparé
- ☁️ Dans un cloud chiffré (Google Drive, Nextcloud, etc.)
- 💾 Sur une clé USB dans un coffre
- 📧 Envoyé par email (à vous-même) si le fichier est petit

⚠️ **Le fichier est chiffré**, mais stockez-le quand même en lieu sûr !

### 3. **Sauvegarde de la clé Restic**

Votre **clé Restic** est LA clé de tout :
- Elle déchiffre la configuration
- Elle déchiffre les backups

**Stockez-la séparément** :
```bash
# Afficher votre clé Restic (sur un serveur opérationnel)
docker exec anemone-core python3 /scripts/decrypt_key.py

# Copiez-la dans un gestionnaire de mots de passe :
# - Bitwarden
# - 1Password
# - KeePass
# - Ou un fichier texte sur une clé USB hors ligne
```

### 4. **Test régulier**

**Testez votre procédure au moins 2 fois par an** :
```bash
# Test rapide (sans Docker)
./start.sh --restore-from=backup.enc
cat config/config.yaml  # Vérifier le contenu
rm -rf config/  # Nettoyer
```

---

## 🤖 Backup Automatique (Phase 2)

### Fonctionnement

Anemone sauvegarde **automatiquement** votre configuration chaque jour à 2h du matin et la distribue à tous vos peers.

**Ce qui est automatisé :**
- ✅ Export quotidien de la configuration chiffrée
- ✅ Upload vers tous les peers configurés via SFTP
- ✅ Stockage redondant (chaque serveur garde les backups des autres)
- ✅ Rotation automatique (7 jours localement, 30 jours sur les peers)

### Vérifier les backups automatiques

```bash
# Voir les backups locaux
ls -lh config-backups/local/

# Voir les backups reçus des peers
ls -lh config-backups/*/

# Vérifier les logs du backup automatique
docker exec anemone-core tail -f /logs/config-backup.log
```

### Forcer un backup manuel

```bash
# Déclencher un backup immédiat
docker exec anemone-core /scripts/core/backup-config-auto.sh
```

---

## 🔍 Auto-Restore : Restauration Automatique

### Scénario : Perte totale du serveur

Vous avez perdu votre serveur et n'avez **PAS** de backup local. Mais vos **peers ont vos backups** !

**Prérequis minimaux :**
- Une copie de `config.yaml` avec la liste des peers (ou la recréer manuellement)
- Les clés SSH pour se connecter aux peers
- Connectivité réseau vers les peers

**Étapes :**

```bash
# 1. Cloner Anemone
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# 2. Créer un config.yaml minimal avec la liste des peers
mkdir -p config
cat > config/config.yaml << 'EOF'
server:
  name: "FR1"

peers:
  - name: "FR2"
    vpn_ip: "10.8.0.2"
    enabled: true
  - name: "US1"
    vpn_ip: "10.8.0.3"
    enabled: true
EOF

# 3. Copier vos clés SSH
# (depuis un backup externe ou regénérées si vous aviez partagé les clés publiques)
cp /backup-externe/id_rsa config/ssh/
cp /backup-externe/id_rsa.pub config/ssh/

# 4. Lancer l'auto-restore
./start.sh --auto-restore
```

**Ce qui se passe :**
1. Le script se connecte à tous vos peers
2. Recherche les backups de configuration disponibles
3. Affiche la liste des backups trouvés avec dates et tailles
4. Vous demande de choisir lequel restaurer
5. Télécharge le backup depuis le peer
6. Vous demande votre clé Restic
7. Restaure complètement la configuration

**Exemple de sortie :**

```
🔍 Mode Auto-Restore - Découverte des Backups

Recherche des backups disponibles sur les peers...

✅ 3 backup(s) trouvé(s)

Backups disponibles:

  1. anemone-backup-FR1-20251021-020000.enc
     Peer: FR2 (10.8.0.2)
     Date: 2025-10-21 02:00:00
     Taille: 2.34 MB

  2. anemone-backup-FR1-20251020-020000.enc
     Peer: US1 (10.8.0.3)
     Date: 2025-10-20 02:00:00
     Taille: 2.32 MB

  3. anemone-backup-FR1-20251019-020000.enc
     Peer: FR2 (10.8.0.2)
     Date: 2025-10-19 02:00:00
     Taille: 2.31 MB

Choisissez un backup à restaurer (1-3) ou 'q' pour annuler: 1

Téléchargement du backup depuis 10.8.0.2...
✅ Backup téléchargé

Pour déchiffrer ce backup, entrez votre clé Restic : [saisie masquée]

🔓 Restauration en cours...
✅ Configuration restaurée avec succès !

Vous pouvez maintenant lancer Docker :
  docker compose up -d
```

### Découvrir les backups disponibles (sans restaurer)

```bash
# Lister tous les backups disponibles sur les peers
python3 scripts/discover-backups.py

# Format JSON pour automatisation
python3 scripts/discover-backups.py --json
```

---

## 📊 Fonctionnalités par Phase

### Phase 1 : Export/Import Manuel ✅

- ✅ Export manuel via web (`/api/config/export`)
- ✅ Import via start.sh (`--restore-from`)
- ✅ Chiffrement AES-256-CBC avec clé Restic
- ✅ Restauration complète de la configuration

### Phase 2 : Backup Automatique ✅

- ✅ Backup automatique quotidien (2h du matin)
- ✅ Distribution automatique vers tous les peers
- ✅ Détection automatique des backups disponibles
- ✅ Mode `./start.sh --auto-restore`
- ✅ Rotation automatique des anciens backups
- ✅ Stockage redondant sur plusieurs serveurs

### Phase 3 : Fonctionnalités Avancées (À venir)

- ⏳ Interface web de recovery avec sélection graphique
- ⏳ Notifications en cas d'échec de backup
- ⏳ Historique multi-versions avec restore point-in-time
- ⏳ Backup incrémentiel de la configuration
- ⏳ Vérification d'intégrité automatique

---

## 🆘 En Cas de Problème

### Le déchiffrement échoue

```
❌ Erreur lors de la restauration: ...
```

**Causes possibles :**
1. **Mauvaise clé Restic** → Vérifiez votre clé
2. **Fichier corrompu** → Essayez un backup plus ancien
3. **Mauvais algorithme** → Fichier créé avec une version incompatible

### Le VPN ne se connecte pas après restauration

```bash
# Vérifier les clés WireGuard
docker exec anemone-core wg show

# Vérifier la configuration
cat config/wireguard/wg0.conf

# Sur les AUTRES serveurs, vérifier l'endpoint public
# Interface web > Peers > Éditer le peer
```

### Les backups ne se synchronisent pas

```bash
# Vérifier la connectivité VPN
docker exec anemone-core ping 10.8.0.2

# Vérifier SFTP
docker exec anemone-core sftp -P 22 -i /root/.ssh/id_rsa restic@10.8.0.2

# Voir les logs Restic
docker exec anemone-core tail -f /logs/restic.log
```

---

## 📞 Support

Pour toute question ou problème :
- 📖 Consultez `README.md`
- 📖 Consultez `TROUBLESHOOTING.md`
- 🐛 Ouvrez une issue sur GitHub

---

**Créé le :** 2025-10-21
**Dernière mise à jour :** 2025-10-21
**Version :** Phase 2 - Backup automatique sur peers
**Prochaine version :** Phase 3 - Fonctionnalités avancées (interface web, notifications, etc.)
