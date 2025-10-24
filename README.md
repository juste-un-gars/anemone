# 🪸 Anemone

**Serveur de fichiers distribué, simple et chiffré, avec redondance entre proches**

## ✨ Fonctionnalités Principales

- 🔐 **Chiffrement AES-256** - Données ET noms de fichiers chiffrés avant synchronisation (rclone crypt)
- 🌐 **VPN WireGuard** - Connexion sécurisée entre tous vos serveurs
- 📦 **Miroir chiffré automatique** - Synchronisation continue de vos données vers vos pairs (totalement illisible chez eux)
- 🔄 **Disaster Recovery complet** - Interface web pour gérer et restaurer vos backups
- 📱 **Configuration par QR Code** - Ajoutez des serveurs en scannant un QR code
- 💾 **Partage SMB/WebDAV** - Accédez à vos fichiers depuis n'importe quel appareil
- 🎨 **Interface web moderne** - Gestion complète via navigateur
- 🔔 **Notifications optionnelles** - Alertes email/webhook en cas de problème (optionnel)

## 🎯 Cas d'Usage

- **Famille** : Sauvegardez les photos/vidéos de famille entre plusieurs maisons
- **Amis** : Partagez et sauvegardez mutuellement vos données importantes
- **Multi-sites** : Redondance automatique entre plusieurs localisations
- **Disaster Recovery** : Récupérez votre configuration complète depuis n'importe quel serveur pair

[Contenu identique jusqu'à la section Installation...]

## 🚀 Installation rapide

### Prérequis

- **Docker & Docker Compose** - [Guide d'installation officiel](https://docs.docker.com/engine/install/)
- 1 Go RAM minimum
- Port UDP 51820 ouvert (port-forwarding sur votre box)
- Un nom de domaine DynDNS (gratuit : [DuckDNS](https://www.duckdns.org), [No-IP](https://www.noip.com))

### Installation

#### Méthode recommandée (script tout-en-un)

```bash
# 1. Cloner le dépôt
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# 2. Lancer le script de démarrage (initialise et démarre automatiquement)
./fr_start.sh   # Interface en français
# ou
./en_start.sh   # Interface en anglais

# 3. Suivre les instructions affichées
# Le script vérifie l'initialisation et démarre Docker
```

#### Méthode manuelle (contrôle total)

```bash
# 1. Cloner le dépôt
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# 2. Initialiser (génère clés WireGuard et SSH)
./scripts/init.sh

# 3. Éditer la configuration
nano .env                    # Mots de passe SMB/WebDAV
nano config/config.yaml      # Configuration générale

# 4. Démarrer Anemone
docker compose up -d
```

**Dans les deux cas**, après le démarrage :

- Ouvrez http://localhost:3000/setup dans votre navigateur
- Suivez l'assistant de configuration
- **⚠️ SAUVEGARDEZ VOTRE CLÉ DANS BITWARDEN !**

## 🔐 Configuration initiale sécurisée

### Première fois (nouveau serveur)

1. Accédez à `http://localhost:3000/setup`
2. Choisissez **"Nouveau serveur"**
3. Une clé de chiffrement est générée automatiquement
4. **⚠️ SAUVEGARDEZ CETTE CLÉ IMMÉDIATEMENT**
   - Dans Bitwarden / 1Password / KeePass
   - Sur une clé USB dans un coffre
   - Sur papier dans un lieu sûr
5. Cochez la case de confirmation
6. Validez

### Restauration après incident

Anemone dispose d'un système complet de disaster recovery (3 méthodes) :

**Méthode 1 : Restauration interactive**
```bash
./fr_restore.sh   # Interface en français
# ou
./en_restore.sh   # Interface en anglais
# Le script vous guide pour restaurer depuis un backup local ou distant
```

**Méthode 2 : Interface web de recovery** (recommandé)
```
http://localhost:3000/recovery
# Interface graphique pour restaurer et gérer tous vos backups de configuration
```

Consultez le [Guide de Disaster Recovery](DISASTER_RECOVERY.md) pour plus de détails.

## 🔒 Sécurité de la clé de chiffrement

### Comment la clé est protégée

✅ **Jamais stockée en clair** : La clé est immédiatement chiffrée après la configuration  
✅ **Chiffrement fort** : AES-256-CBC avec 100 000 itérations PBKDF2  
✅ **Clé dérivée du système** : Utilise l'UUID unique de la machine  
✅ **Inaccessible via l'interface** : Impossible de consulter la clé après setup  
✅ **Logs sécurisés** : La clé n'apparaît jamais dans les logs  

### Que se passe-t-il si...

**❓ Je perds ma clé ?**  
→ ❌ Vos backups sont **irrécupérables**. C'est pourquoi il faut la sauvegarder !

**❓ Mon serveur est volé ?**  
→ ⚠️ Le voleur ne peut pas lire vos backups distants (ils sont chiffrés)  
→ ⚠️ Il peut potentiellement déchiffrer la clé si le serveur est démarré  
→ ✅ Solution : Coupez l'accès réseau du serveur volé immédiatement

**❓ Un pair est compromis ?**  
→ ✅ Le pirate a vos backups chiffrés mais **pas la clé**  
→ ✅ Vos données restent protégées

**❓ Je veux changer de serveur ?**  
→ ✅ Copiez votre clé depuis Bitwarden  
→ ✅ Utilisez "Restauration" lors du setup  
→ ✅ Récupérez vos données depuis n'importe quel pair

## 🛡️ Meilleures pratiques de sécurité

### ✅ À faire ABSOLUMENT

1. **Sauvegarder la clé** dans au moins 2 endroits différents :
   - Gestionnaire de mots de passe (Bitwarden, 1Password)
   - Clé USB chiffrée dans un coffre
   - Papier dans un lieu sûr physique

2. **Changer les mots de passe par défaut** dans `.env` :
   ```bash
   SMB_PASSWORD=MotDePasseFort123!
   WEBDAV_PASSWORD=AutreMotDePasseFort456!
   ```

3. **Configurer le firewall** :
   ```bash
   # Bloquer SMB/WebDAV depuis Internet
   sudo ufw deny 445
   sudo ufw deny 8080
   # Autoriser uniquement WireGuard
   sudo ufw allow 51820/udp
   ```

4. **Tester la restauration** régulièrement (1x par an minimum)

### ⚠️ À NE PAS faire

❌ Commiter la clé dans Git  
❌ Envoyer la clé par email non chiffré  
❌ Stocker la clé en clair sur le cloud  
❌ Partager la clé avec vos pairs (ils n'en ont pas besoin)  
❌ Oublier de sauvegarder la clé après génération  

## 🔧 Maintenance

### Sauvegarder votre clé après setup

Si vous avez oublié de sauvegarder votre clé lors du setup initial, vous pouvez la récupérer **une seule fois** avec cette commande :

```bash
# ⚠️ À utiliser UNIQUEMENT en urgence
docker exec anemone-core python3 /scripts/decrypt_key.py
```

**Ensuite sauvegardez-la IMMÉDIATEMENT dans Bitwarden !**

### Vérifier l'intégrité des backups

**Via l'interface web** (recommandé) :
```
http://localhost:3000/recovery
→ Cliquer sur "Vérifier" sur un backup
```

**Via ligne de commande** :
```bash
# Vérifier l'intégrité d'un backup de configuration
curl -X POST http://localhost:3000/api/recovery/verify \
  -H "Content-Type: application/json" \
  -d '{"backup_path": "/config-backups/local/backup.enc"}'

# Vérifier les backups Restic (données)
docker exec anemone-core restic -r sftp:user@host:/path check
```

### Tester une restauration

```bash
# Via l'interface web (recommandé)
http://localhost:3000/recovery

# Via script interactif
./fr_restore.sh   # ou ./en_restore.sh
# Suivez les instructions pour restaurer depuis un backup local ou distant
```

## 📋 Checklist de sécurité

Avant de mettre en production :

- [ ] Clé de chiffrement sauvegardée dans Bitwarden
- [ ] Clé de chiffrement sauvegardée sur clé USB
- [ ] Mots de passe SMB/WebDAV changés
- [ ] Firewall configuré (bloquer SMB/WebDAV depuis Internet)
- [ ] Port-forwarding WireGuard (51820/UDP) configuré
- [ ] DNS dynamique configuré et testé
- [ ] Premier backup réussi
- [ ] Restauration testée depuis un pair
- [ ] Clés publiques échangées avec les pairs
- [ ] VPN WireGuard fonctionnel : `docker exec anemone-wireguard wg show`

## ❓ FAQ Sécurité

**Q : Mes pairs peuvent-ils lire mes données ?**  
R : Non. Les backups sont chiffrés avec votre clé. Les pairs ne stockent que des données chiffrées.

**Q : La clé est-elle visible quelque part ?**  
R : Non, après le setup initial, elle est chiffrée et inaccessible via l'interface web ou les logs.

**Q : Que faire si je soupçonne une compromission ?**  
R : 
1. Arrêtez immédiatement Anemone : `docker-compose down`
2. Changez tous vos mots de passe
3. Générez une nouvelle clé et refaites les backups
4. Informez vos pairs

**Q : Puis-je changer de clé de chiffrement ?**  
R : Oui, mais il faudra refaire tous les backups. Procédure :
1. Sauvegarder vos données locales
2. Supprimer `config/.setup-completed`
3. Relancer `docker-compose restart api`
4. Refaire le setup avec une nouvelle clé
5. Les nouveaux backups utiliseront la nouvelle clé

**Q : Comment partager l'accès aux fichiers sans partager la clé ?**
R : Utilisez SMB/WebDAV avec des comptes séparés. La clé Restic reste privée et sert uniquement aux backups.

**Q : Comment connecter plusieurs serveurs Anemone ensemble ?**
R : Consultez le guide complet [INTERCONNEXION_GUIDE.md](INTERCONNEXION_GUIDE.md) ou utilisez le script `./scripts/add-peer.sh` pour un ajout interactif.

## 🤝 Contribuer

Les contributions sont les bienvenues ! Consultez [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 Licence

Copyright (C) 2025 juste-un-gars

Ce programme est un logiciel libre ; vous pouvez le redistribuer et/ou le modifier selon les termes de la **GNU Affero General Public License** telle que publiée par la Free Software Foundation ; soit la version 3 de la Licence, soit (à votre choix) toute version ultérieure.

Ce programme est distribué dans l'espoir qu'il sera utile, mais SANS AUCUNE GARANTIE ; sans même la garantie implicite de COMMERCIALISATION ou D'ADÉQUATION À UN USAGE PARTICULIER. Voir la GNU Affero General Public License pour plus de détails.

Vous devriez avoir reçu une copie de la GNU Affero General Public License avec ce programme. Si ce n'est pas le cas, consultez <https://www.gnu.org/licenses/>.

### Pourquoi AGPLv3 ?

L'AGPLv3 garantit que :
- ✅ Le code reste **libre et open source**
- ✅ Toute modification doit être **partagée avec la communauté**
- ✅ Même un service web utilisant Anemone doit **publier son code source**
- ✅ Vous pouvez **utiliser, modifier et distribuer** librement
- ✅ Les **prestations de service** sont autorisées (installation, maintenance, support)

Voir le fichier [LICENSE](LICENSE) pour le texte complet

## 🙏 Remerciements

Construit avec :
- [WireGuard](https://www.wireguard.com/) - VPN moderne
- [Restic](https://restic.net/) - Backup incrémental chiffré
- [Samba](https://www.samba.org/) - Partage SMB
- [Docker](https://www.docker.com/) - Conteneurisation
- [FastAPI](https://fastapi.tiangolo.com/) - API web

---

Fait avec ❤️ pour partager des fichiers entre proches, sans dépendre du cloud.

**⚠️ RAPPEL IMPORTANT** : Sauvegardez votre clé de chiffrement dans Bitwarden dès la première configuration !
