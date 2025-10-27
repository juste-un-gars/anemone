# 🚀 Guide de démarrage rapide

Ce guide vous aide à tester Anemone v2 (refonte Go).

## Prérequis

Choisissez l'une des deux options :

### Option A : Docker (recommandé)
```bash
# Installation Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Démarrer Anemone
docker compose up --build
```

### Option B : Go en local
```bash
# Installation Go 1.21+
# https://go.dev/doc/install

# Télécharger les dépendances
go mod download

# Démarrer Anemone
go run cmd/anemone/main.go
```

## Premier démarrage

1. **Accédez à l'interface web** :
   ```
   http://localhost:8080
   ```
   → Vous serez automatiquement redirigé vers `/setup`

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
   - Actuellement : message "Dashboard coming soon"
   - À implémenter dans les prochaines phases

## Vérification de l'installation

### Santé de l'application
```bash
curl http://localhost:8080/health
# Retour attendu: OK
```

### Base de données
```bash
# Vérifier que la base existe
ls -la data/db/anemone.db

# Inspecter le contenu (après setup)
sqlite3 data/db/anemone.db "SELECT * FROM system_config;"
sqlite3 data/db/anemone.db "SELECT id, username, is_admin FROM users;"
```

### Logs
```bash
# Avec Docker
docker compose logs -f anemone

# En local
# Les logs s'affichent directement dans le terminal
```

## Structure des données créées

Après le setup initial :

```
data/
├── db/
│   └── anemone.db          # Base SQLite
├── shares/                 # Partages utilisateurs (à créer)
└── config/                 # Configs générées (à créer)
```

## Réinitialiser le setup

Si vous voulez recommencer :

```bash
# ATTENTION : Supprime toutes les données !
rm -rf data/db/anemone.db

# Redémarrer l'application
docker compose restart anemone
# OU
go run cmd/anemone/main.go
```

## Tests fonctionnels

### Test 1 : Redirection setup
```bash
# Avant setup : doit rediriger vers /setup
curl -I http://localhost:8080/
# Attendu: HTTP 303 See Other, Location: /setup

# Après setup : doit afficher le dashboard
curl -I http://localhost:8080/
# Attendu: HTTP 200 OK
```

### Test 2 : Protection du setup
```bash
# Après setup : /setup doit rediriger vers /
curl -I http://localhost:8080/setup
# Attendu: HTTP 303 See Other, Location: /
```

### Test 3 : Création utilisateur
```bash
# Vérifier que l'admin a bien été créé
sqlite3 data/db/anemone.db <<EOF
SELECT
    username,
    email,
    is_admin,
    datetime(created_at) as created,
    datetime(activated_at) as activated
FROM users;
EOF
```

## Prochaines étapes

Une fois le setup fonctionnel, vous pouvez :

1. ✅ Tester le changement de langue (bouton dans le formulaire)
2. ✅ Vérifier que la clé est bien générée (32 bytes en base64)
3. ✅ Confirmer que la clé est chiffrée en base de données
4. ⏭️ Implémenter le système d'authentification
5. ⏭️ Créer le dashboard admin

## Dépannage

### Erreur : "bind: address already in use"
Le port 8080 est déjà utilisé. Changez le port :

```bash
# Option 1 : Variable d'environnement
export PORT=8081
go run cmd/anemone/main.go

# Option 2 : Modifier docker-compose.yml
ports:
  - "8081:8080"
```

### Erreur : "no such table: system_config"
Les migrations n'ont pas été exécutées. Vérifiez les logs au démarrage.

### Erreur : Templates introuvables
Assurez-vous que le dossier `web/templates/` existe et contient les fichiers HTML.

```bash
ls -la web/templates/
# Attendu: base.html, setup.html, setup_success.html
```

## Fichiers importants

- `cmd/anemone/main.go` - Point d'entrée
- `internal/web/router.go` - Routeur HTTP et handlers
- `internal/i18n/i18n.go` - Traductions FR/EN
- `internal/users/users.go` - Gestion utilisateurs
- `internal/crypto/crypto.go` - Chiffrement
- `internal/database/migrations.go` - Schéma de base de données
- `web/templates/*.html` - Templates HTML

## Support

- 📚 Documentation complète : voir `README.md`
- 🐛 Problèmes : https://github.com/juste-un-gars/anemone/issues
