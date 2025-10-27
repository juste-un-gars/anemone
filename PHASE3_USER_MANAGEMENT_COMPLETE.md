# Phase 3 : Gestion des Utilisateurs (Admin) - Terminé ✅

**Date** : 2025-10-27
**Objectif** : Permettre aux administrateurs de créer des utilisateurs et de générer des liens d'activation

## 🎯 Fonctionnalités implémentées

### 1. Module de tokens d'activation
- **Fichier** : `internal/activation/tokens.go`
- **Fonctionnalités** :
  - Génération de tokens aléatoires sécurisés (32 bytes, base64 URL-safe)
  - Création de tokens avec expiration automatique (24h)
  - Récupération de tokens par chaîne
  - Validation de tokens (non expiré, non utilisé)
  - Marquage de tokens comme utilisés
  - Suppression automatique des tokens expirés
  - Récupération de tous les tokens en attente

### 2. Fonctions utilisateur étendues
- **Fichier** : `internal/users/users.go` (étendu)
- **Nouvelles fonctions** :
  - `CreatePendingUser()` - Crée un utilisateur en attente d'activation
  - `ActivateUser()` - Active un utilisateur (mot de passe + clé)
  - `GetByID()` - Récupère un utilisateur par ID
  - `GetAllUsers()` - Liste tous les utilisateurs
  - `DeleteUser()` - Supprime un utilisateur
  - `IsActivated()` - Vérifie si un utilisateur est activé

### 3. Traductions étendues
- **Fichier** : `internal/i18n/i18n.go` (étendu)
- **+18 nouvelles clés** (FR/EN) :
  - Labels et messages pour la gestion des utilisateurs
  - Messages d'erreur (username requis, username existe)
  - Messages pour les tokens d'activation
  - Labels de statut (actif, en attente)

### 4. Page liste des utilisateurs
- **Template** : `web/templates/admin_users.html`
- **Contenu** :
  - Tableau avec tous les utilisateurs
  - Colonnes : Username, Email, Rôle, Statut, Date création
  - Badge visuel pour rôle (Admin/User)
  - Badge visuel pour statut (Actif/En attente)
  - Actions : Copier le lien d'activation, Supprimer
  - Bouton "Ajouter un utilisateur"
  - Message si aucun utilisateur

### 5. Page d'ajout d'utilisateur
- **Template** : `web/templates/admin_users_add.html`
- **Contenu** :
  - Formulaire complet : Username, Email, Rôle, Quotas
  - Choix radio : Utilisateur ou Administrateur
  - Deux champs de quotas : Total et Backup (en GB)
  - Valeurs par défaut : 100 GB total, 50 GB backup
  - Info box expliquant le processus d'activation
  - Validation côté client et serveur
  - Affichage des erreurs

### 6. Page d'affichage du lien d'activation
- **Template** : `web/templates/admin_users_token.html`
- **Contenu** :
  - Message de succès avec animation
  - Bannière d'avertissement (expire dans 24h)
  - Affichage du lien d'activation complet
  - Bouton de copie dans le presse-papiers
  - Information sur l'expiration
  - Informations de l'utilisateur créé
  - Bouton "Terminé" pour retour liste

### 7. Handlers web complets
- **Fichier** : `internal/web/router.go` (étendu)
- **Nouveaux handlers** :
  - `handleAdminUsers()` - Liste des utilisateurs
  - `handleAdminUsersAdd()` - GET/POST création utilisateur
  - `handleAdminUsersActions()` - Actions (token, delete)

- **Nouvelles routes** :
  - `GET /admin/users` - Liste des utilisateurs
  - `GET /admin/users/add` - Formulaire d'ajout
  - `POST /admin/users/add` - Traitement du formulaire
  - `GET /admin/users/{id}/token` - Affichage du lien
  - `POST /admin/users/{id}/delete` - Suppression

## 📁 Fichiers créés/modifiés

### Nouveaux fichiers
- `internal/activation/tokens.go` - Module de tokens
- `web/templates/admin_users.html` - Liste des utilisateurs
- `web/templates/admin_users_add.html` - Formulaire d'ajout
- `web/templates/admin_users_token.html` - Affichage du lien

### Fichiers modifiés
- `internal/users/users.go` - +6 fonctions
- `internal/i18n/i18n.go` - +18 clés (FR/EN)
- `internal/web/router.go` - +3 handlers, +5 routes

## 🔄 Flux d'utilisation

### Création d'un utilisateur par l'admin

```
1. Admin se connecte au dashboard
   ↓
2. Clique sur "Gestion des utilisateurs"
   ↓
3. Clique sur "Ajouter un utilisateur"
   ↓
4. Remplit le formulaire :
   - Username
   - Email (optionnel)
   - Rôle (User/Admin)
   - Quotas (Total/Backup)
   ↓
5. Soumet le formulaire
   ↓
6. Système crée l'utilisateur (statut: pending)
   ↓
7. Système génère un token d'activation (24h)
   ↓
8. Redirige vers page d'affichage du lien
   ↓
9. Admin copie le lien
   ↓
10. Admin envoie le lien à l'utilisateur
    (par email, chat, etc.)
```

### Actions disponibles sur la liste

- **Copier le lien** (si utilisateur non activé) :
  - Génère un nouveau lien si l'ancien a expiré
  - Affiche le lien à copier

- **Supprimer** :
  - Confirmation JavaScript
  - Suppression immédiate
  - Rechargement automatique de la page

## 🔐 Sécurité

### Tokens d'activation
- **Génération** : 32 bytes aléatoires sécurisés
- **Encodage** : Base64 URL-safe (évite les problèmes dans URLs)
- **Expiration** : 24 heures exactement
- **Unicité** : 1 token par utilisateur à la fois
- **Validation** : Vérifie expiration et statut

### Création d'utilisateurs
- **Username unique** : Vérification en base de données
- **Statut pending** : Ne peut pas se connecter avant activation
- **Clé et mot de passe** : Définis lors de l'activation (Phase 4)
- **Quotas** : Configurables avec valeurs par défaut saines

### Protection des routes
- **Toutes les routes admin** : Protégées par `RequireAdmin` middleware
- **Vérification rôle** : Admin uniquement
- **Redirection** : Vers /login si non authentifié
- **403 Forbidden** : Si authentifié mais pas admin

## 🎨 Interface utilisateur

### Liste des utilisateurs

**Tableau responsive avec** :
- Badges de rôle colorés (violet pour admin, gris pour user)
- Badges de statut colorés (vert pour actif, jaune pour pending)
- Format de date lisible (JJ/MM/AAAA)
- Actions contextuelles selon le statut
- Empty state si aucun utilisateur

**Design moderne** :
- Tailwind CSS pour styling
- Hover effects sur les actions
- Bouton d'ajout proéminent
- Navigation breadcrumb

### Formulaire d'ajout

**Champs organisés** :
- Inputs styled uniformément
- Radio buttons pour le rôle
- Grid 2 colonnes pour les quotas
- Info box explicatif
- Boutons Cancel/Submit

**UX** :
- Labels clairs
- Placeholders informatifs
- Validation en temps réel (HTML5)
- Messages d'erreur contextuels

### Page du lien d'activation

**Hiérarchie visuelle** :
- Icône de succès proéminente
- Titre et sous-titre clairs
- Bannière d'avertissement jaune
- Lien dans un input readonly
- Bouton de copie intégré

**Interactions** :
- Copie dans le presse-papiers
- Feedback visuel (✓ Copié !)
- Informations supplémentaires en bas

## 🧪 Flux de test

### Test 1 : Création d'utilisateur simple
```bash
# 1. Login en tant qu'admin
http://localhost:8080/login
# Credentials: admin / password123

# 2. Accéder à la gestion
http://localhost:8080/admin/users
→ Liste vide ou avec admin

# 3. Cliquer "Ajouter un utilisateur"
→ Formulaire affiché

# 4. Remplir :
Username: john.doe
Email: john.doe@example.com
Role: User
Quota Total: 100 GB
Quota Backup: 50 GB

# 5. Soumettre
→ Redirige vers /admin/users/{id}/token

# 6. Vérifier le lien
→ Affiche: http://localhost:8080/activate/{token}

# 7. Copier le lien
→ Feedback "✓ Copié !"

# 8. Retourner à la liste
→ john.doe apparaît avec statut "En attente"
```

### Test 2 : Validation des erreurs
```bash
# Tenter de créer sans username
→ Erreur : "Le nom d'utilisateur est requis"

# Tenter de créer avec username existant
→ Erreur : "Ce nom d'utilisateur existe déjà"
```

### Test 3 : Base de données
```bash
# Vérifier l'utilisateur créé
sqlite3 data/db/anemone.db <<EOF
SELECT username, email, is_admin, activated_at
FROM users;
EOF
# john.doe doit avoir activated_at = NULL

# Vérifier le token
sqlite3 data/db/anemone.db <<EOF
SELECT token, username, expires_at, used_at
FROM activation_tokens;
EOF
# Token doit exister, used_at = NULL
```

### Test 4 : Expiration de token
```bash
# Manipuler l'horloge système ou attendre 24h
# Tenter d'accéder à /activate/{expired_token}
→ Devrait rejeter le token (Phase 4)
```

### Test 5 : Suppression d'utilisateur
```bash
# Depuis la liste, cliquer "Supprimer"
→ Confirmation JavaScript

# Confirmer
→ Utilisateur supprimé
→ Page rechargée automatiquement
→ Utilisateur n'apparaît plus

# Vérifier en BD
sqlite3 data/db/anemone.db "SELECT COUNT(*) FROM users WHERE username = 'john.doe';"
# Doit retourner 0
```

## 📊 Structure des données

### Table activation_tokens
```sql
CREATE TABLE activation_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT UNIQUE NOT NULL,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    email TEXT,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)
```

### Utilisateur pending (exemple)
```json
{
  "id": 2,
  "username": "john.doe",
  "password_hash": "",  // Vide jusqu'à activation
  "email": "john.doe@example.com",
  "encryption_key_hash": "",  // Vide jusqu'à activation
  "encryption_key_encrypted": "",  // Vide jusqu'à activation
  "is_admin": false,
  "quota_total_gb": 100,
  "quota_backup_gb": 50,
  "created_at": "2025-10-27T10:30:00Z",
  "activated_at": null  // NULL = pending
}
```

### Token d'activation (exemple)
```json
{
  "id": 1,
  "token": "a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j",
  "user_id": 2,
  "username": "john.doe",
  "email": "john.doe@example.com",
  "created_at": "2025-10-27T10:30:00Z",
  "expires_at": "2025-10-28T10:30:00Z",  // +24h
  "used_at": null  // NULL jusqu'à activation
}
```

## ⏭️ Prochaines étapes

### Phase 4 : Activation utilisateur (Onboarding)
- [ ] Route `/activate/{token}` publique
- [ ] Validation du token (existe, non expiré, non utilisé)
- [ ] Page de choix du mot de passe
- [ ] Génération de la clé de chiffrement
- [ ] Affichage unique de la clé avec avertissements
- [ ] Confirmation et activation finale
- [ ] Redirection vers login

### Phase 5 : Partages et quotas
- [ ] Configuration Samba dynamique
- [ ] Création automatique des répertoires utilisateurs
- [ ] Calcul de l'usage réel du stockage
- [ ] Monitoring des quotas
- [ ] Alertes de dépassement
- [ ] Affichage dans les dashboards

### Phase 6 : Synchronisation P2P
- [ ] Adaptation rclone pour multi-users
- [ ] Configuration de la synchronisation
- [ ] Gestion des pairs
- [ ] Logs de synchronisation
- [ ] Dashboard de statut sync

## 📝 Notes techniques

### Génération de tokens
```go
b := make([]byte, 32)
rand.Read(b)
token := base64.URLEncoding.EncodeToString(b)
// Résultat : ~43 caractères, URL-safe
```

### Parsing des URLs dynamiques
```go
// URL : /admin/users/2/token
parts := strings.Split(strings.Trim(path, "/"), "/")
// parts = ["admin", "users", "2", "token"]
userID, _ := strconv.Atoi(parts[2])  // 2
action := parts[3]  // "token"
```

### Copie dans le presse-papiers (JavaScript)
```javascript
const input = document.getElementById('activation-link');
input.select();
document.execCommand('copy');
// Feedback visuel pendant 2 secondes
```

## 📈 Statistiques

- **Lignes de code Go** : ~450 lignes (activation + users + router)
- **Templates HTML** : ~450 lignes (3 nouveaux templates)
- **Traductions** : +18 clés (FR/EN)
- **Routes** : +5 routes
- **Fonctions** : +10 fonctions

## ✅ Ce qui fonctionne (théoriquement)

1. ✅ Admin peut voir la liste de tous les utilisateurs
2. ✅ Admin peut créer un nouvel utilisateur
3. ✅ Système génère automatiquement un token d'activation
4. ✅ Admin peut copier le lien d'activation
5. ✅ Lien expire automatiquement après 24h
6. ✅ Admin peut supprimer un utilisateur
7. ✅ Validation des champs (username unique, requis)
8. ✅ Quotas configurables avec valeurs par défaut
9. ✅ Rôles configurables (User/Admin)
10. ✅ Interface responsive et moderne

## 🎉 Conclusion

La Phase 3 est **complète** au niveau du code. Les administrateurs peuvent maintenant :
- Créer des utilisateurs avec configuration complète
- Générer des liens d'activation sécurisés
- Gérer les utilisateurs (liste, suppression)
- Voir le statut de chaque utilisateur

**Prochaine action recommandée** :
1. Tester le flux complet de création
2. Vérifier la génération de tokens
3. Tester la copie du lien d'activation
4. Passer à la Phase 4 (activation par l'utilisateur)
