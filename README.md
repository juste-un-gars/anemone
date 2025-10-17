# 🪸 Anemone

**Serveur de fichiers distribué, simple et chiffré, avec redondance entre proches**

[Contenu identique jusqu'à la section Installation...]

## 🚀 Installation rapide

### Prérequis

- Docker & Docker Compose
- 1 Go RAM minimum
- Port UDP 51820 ouvert (port-forwarding sur votre box)
- Un nom de domaine DynDNS (gratuit : [DuckDNS](https://www.duckdns.org), [No-IP](https://www.noip.com))

### Installation

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
docker-compose up -d

# 5. Configuration sécurisée (IMPORTANT !)
# Ouvrir http://localhost:3000/setup dans votre navigateur
# Suivre l'assistant de configuration
```

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

1. Accédez à `http://localhost:3000/setup`
2. Choisissez **"Restauration"**
3. Collez votre clé depuis Bitwarden
4. Validez
5. Utilisez `./scripts/restore.sh` pour récupérer vos données

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
docker exec anemone-restic sh -c '
  SYSTEM_KEY=$(cat /proc/sys/kernel/random/uuid)
  SALT=$(cat /config/.restic.salt)
  openssl enc -aes-256-cbc -d \
    -pbkdf2 -iter 100000 \
    -pass pass:"$SYSTEM_KEY:$SALT" \
    -in /config/.restic.encrypted
'
```

**Ensuite sauvegardez-la IMMÉDIATEMENT dans Bitwarden !**

### Vérifier l'intégrité des backups

```bash
# Vérifier tous les backups
docker exec anemone-restic restic -r sftp:user@host:/path check

# Vérifier un backup spécifique
docker exec anemone-restic restic -r sftp:user@host:/path snapshots
```

### Tester une restauration

```bash
# Utiliser le script de restauration
./scripts/restore.sh

# Suivre les instructions interactives
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

MIT License - voir [LICENSE](LICENSE)

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
