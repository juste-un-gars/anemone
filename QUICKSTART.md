# 🚀 Guide de démarrage rapide

Ce guide vous aide à installer et démarrer Anemone v2.

## Prérequis

- Go 1.21+ - [Installation](https://go.dev/doc/install)
- Samba (pour partages SMB)
- Accès sudo (pour configuration système)

## Installation automatique (recommandé)

```bash
# Cloner le dépôt
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Lancer l'installateur
sudo ./install.sh

# L'installateur va :
# - Compiler le binaire
# - Créer /srv/anemone
# - Installer Samba
# - Configurer SELinux (Fedora/RHEL)
# - Configurer le firewall
# - Créer le service systemd
# - Générer les certificats TLS
```

## Installation manuelle

```bash
# Cloner le dépôt
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Compiler
CGO_ENABLED=1 go build -o anemone ./cmd/anemone

# Créer répertoire données
sudo mkdir -p /srv/anemone
sudo chown $USER:$USER /srv/anemone

# Démarrer
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

## Premier démarrage

1. **Accédez à l'interface web** :
   ```
   https://localhost:8443
   ```
   - Acceptez l'avertissement du certificat auto-signé (normal en local)
   - Vous serez automatiquement redirigé vers `/setup`

2. **Page de configuration initiale** :
   - Sélectionnez la langue (FR/EN)
   - Donnez un nom au NAS (ex: `nas-maison`)
   - Choisissez le fuseau horaire
   - Créez le premier compte administrateur :
     - Nom d'utilisateur
     - Mot de passe (min 8 caractères)
     - Email (optionnel)

3. **Clé de chiffrement** :
   - ⚠️ **CRITIQUE** : La clé s'affiche une seule fois !
   - Copiez-la ou téléchargez-la
   - Cochez les cases de confirmation
   - Cliquez sur "Accéder au tableau de bord"

4. **Tableau de bord** :
   - Dashboard admin avec gestion utilisateurs, pairs P2P, partages SMB

## Vérification de l'installation

### Santé de l'application
```bash
curl -k https://localhost:8443/health
# Retour attendu: OK
```

### Base de données
```bash
# Vérifier que la base existe
ls -la /srv/anemone/db/anemone.db

# Inspecter le contenu (après setup)
sqlite3 /srv/anemone/db/anemone.db "SELECT * FROM system_config;"
sqlite3 /srv/anemone/db/anemone.db "SELECT id, username, is_admin FROM users;"
```

### Logs
```bash
# Avec systemd
journalctl -u anemone -f

# Si démarré manuellement
# Les logs s'affichent dans le terminal
```

### Partages SMB
```bash
# Vérifier service Samba
sudo systemctl status smb    # Fedora
sudo systemctl status smbd   # Debian/Ubuntu

# Tester depuis Windows
\\<ip-serveur>\backup_utilisateur
```

## Structure des données créées

Après le setup initial :

```
/srv/anemone/
├── db/
│   └── anemone.db          # Base SQLite
├── shares/                 # Partages utilisateurs
│   └── username/
│       ├── backup/         # Synchronisé vers pairs
│       └── data/           # Local uniquement
├── certs/                  # Certificats TLS
└── smb/                    # Configuration Samba
    └── smb.conf
```

## Réinitialiser le setup

Si vous voulez recommencer :

```bash
# ATTENTION : Supprime toutes les données !
sudo rm -rf /srv/anemone/*

# Redémarrer l'application
systemctl restart anemone
```

## Tests fonctionnels

### Test 1 : Redirection setup
```bash
# Avant setup : doit rediriger vers /setup
curl -I -k https://localhost:8443/
# Attendu: HTTP 303 See Other, Location: /setup

# Après setup : doit afficher le dashboard
curl -I -k https://localhost:8443/
# Attendu: HTTP 200 OK
```

### Test 2 : Protection du setup
```bash
# Après setup : /setup doit rediriger vers /
curl -I -k https://localhost:8443/setup
# Attendu: HTTP 303 See Other, Location: /
```

### Test 3 : Création utilisateur
```bash
# Vérifier que l'admin a bien été créé
sqlite3 /srv/anemone/db/anemone.db <<EOF
SELECT
    username,
    email,
    is_admin,
    datetime(created_at) as created,
    datetime(activated_at) as activated
FROM users;
EOF
```

### Test 4 : Partages SMB
```bash
# Vérifier qu'un utilisateur activé a bien ses partages
sqlite3 /srv/anemone/db/anemone.db "SELECT name, path, protocol FROM shares;"

# Vérifier config Samba
sudo testparm -s | grep -A 5 backup_

# Tester accès
smbclient -L localhost -U utilisateur
```

## Prochaines étapes

Fonctionnalités actuellement implémentées :

1. ✅ Configuration initiale (setup)
2. ✅ Système d'authentification
3. ✅ Gestion multi-utilisateurs
4. ✅ Activation utilisateurs (avec liens temporaires)
5. ✅ Gestion pairs P2P
6. ✅ Partages SMB automatiques (backup + data)
7. ✅ Configuration Samba dynamique
8. ✅ Support HTTPS avec TLS auto-signé
9. ✅ Multilingue (FR/EN)

À venir :

1. ⏭️ Synchronisation P2P réelle
2. ⏭️ Chiffrement des partages backup
3. ⏭️ Quotas utilisateur
4. ⏭️ Corbeille avec rétention
5. ⏭️ Monitoring et statistiques

## Dépannage

### Erreur : "bind: address already in use"
Le port est déjà utilisé. Changez le port :

```bash
# Variable d'environnement
export HTTPS_PORT=8444
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

### Erreur : "no such table: system_config"
Les migrations n'ont pas été exécutées. Vérifiez les logs au démarrage.

### Erreur : Partages SMB inaccessibles

```bash
# Vérifier SELinux (Fedora/RHEL)
ls -laZ /srv/anemone/shares/
# Le contexte doit être samba_share_t

# Corriger si nécessaire
sudo semanage fcontext -a -t samba_share_t "/srv/anemone/shares(/.*)?"
sudo restorecon -Rv /srv/anemone/shares/
sudo setsebool -P samba_export_all_rw on
```

### Erreur : Certificat TLS invalide

C'est normal ! Le certificat est auto-signé pour un usage local.

- **Navigateur** : Cliquez sur "Avancé" → "Continuer vers le site"
- **curl** : Utilisez l'option `-k` ou `--insecure`

## Fichiers importants

- `cmd/anemone/main.go` - Point d'entrée
- `internal/web/` - Routeur HTTP et handlers
- `internal/i18n/` - Traductions FR/EN
- `internal/users/` - Gestion utilisateurs
- `internal/shares/` - Gestion partages SMB
- `internal/peers/` - Gestion pairs P2P
- `internal/smb/` - Configuration Samba
- `internal/crypto/` - Chiffrement
- `internal/database/` - Schéma et migrations
- `web/templates/` - Templates HTML
- `scripts/` - Scripts d'installation
- `install.sh` - Installateur automatique

## Support

- 📚 Documentation complète : voir `README.md`
- 🐛 Problèmes : https://github.com/juste-un-gars/anemone/issues
