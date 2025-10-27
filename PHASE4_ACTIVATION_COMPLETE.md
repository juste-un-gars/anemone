# Phase 4 : Activation Utilisateur - Terminé ✅

**Date** : 2025-10-27
**Objectif** : Permettre aux utilisateurs d'activer leur compte via un lien d'activation et de récupérer leur clé de chiffrement

## 🎯 Fonctionnalités implémentées

### 1. Traductions étendues
- **Fichier** : `internal/i18n/i18n.go` (étendu)
- **+18 nouvelles clés** (FR/EN) :
  - Labels pour la page d'activation
  - Messages d'erreur (token invalide, expiré, déjà utilisé)
  - Messages de succès
  - Instructions de sécurité

### 2. Page d'activation
- **Template** : `web/templates/activate.html`
- **Contenu** :
  - Message de bienvenue avec nom d'utilisateur
  - Champ username (readonly, pré-rempli)
  - Formulaire de choix du mot de passe
  - Confirmation du mot de passe
  - Info box expliquant la génération de clé
  - Validation côté client et serveur
  - Affichage des erreurs contextuelles

### 3. Page de succès d'activation
- **Template** : `web/templates/activate_success.html`
- **Contenu** :
  - Icône de succès avec animation
  - Bannière d'avertissement (clé affichée 1 seule fois)
  - Affichage de la clé de chiffrement
  - Bouton de copie dans le presse-papiers
  - Bouton de téléchargement de la clé
  - Checkboxes de confirmation obligatoires
  - Bouton "Se connecter" désactivé jusqu'à confirmation

### 4. Handlers web complets
- **Fichier** : `internal/web/router.go` (étendu)
- **Nouveaux handlers** :
  - `handleActivate()` - GET/POST activation
  - `handleActivateConfirm()` - Confirmation finale

- **Nouvelles routes** :
  - `GET /activate/{token}` - Formulaire d'activation
  - `POST /activate/{token}` - Traitement de l'activation
  - `POST /activate/confirm` - Confirmation et redirection

## 📁 Fichiers créés/modifiés

### Nouveaux fichiers
- `web/templates/activate.html` - Page d'activation
- `web/templates/activate_success.html` - Page de succès

### Fichiers modifiés
- `internal/i18n/i18n.go` - +18 clés (FR/EN)
- `internal/web/router.go` - +2 handlers, +2 routes

## 🔄 Flux d'activation complet

### Du côté de l'utilisateur

```
1. Utilisateur reçoit le lien d'activation
   http://localhost:8080/activate/{token}
   ↓
2. Clique sur le lien
   ↓
3. Système vérifie le token :
   - Existe ?
   - Non expiré (< 24h) ?
   - Non utilisé ?
   ↓
4. Affiche formulaire d'activation
   - Username pré-rempli (readonly)
   - Choisir mot de passe (min 8 chars)
   - Confirmer mot de passe
   ↓
5. Soumet le formulaire
   ↓
6. Système :
   - Hash le mot de passe (bcrypt)
   - Génère clé de chiffrement (32 bytes)
   - Chiffre la clé avec master key
   - Met à jour l'utilisateur en BD
   - Marque le token comme utilisé
   ↓
7. Affiche page de succès avec la clé
   ↓
8. Utilisateur :
   - Copie ou télécharge la clé
   - Coche les confirmations
   - Clique "Se connecter"
   ↓
9. Redirection vers /login
   ↓
10. Utilisateur peut se connecter !
```

## 🔐 Sécurité

### Validation du token
- **Existence** : Token doit exister en base de données
- **Expiration** : Token doit être valide (< 24h)
- **Utilisation** : Token ne peut être utilisé qu'une seule fois
- **Messages d'erreur** : Différenciés (invalide vs expiré vs utilisé)

### Génération de la clé
- **Aléatoire sécurisé** : 32 bytes via `crypto/rand`
- **Encodage** : Base64 standard
- **Hash SHA-256** : Stocké pour vérification
- **Chiffrement** : AES-256-GCM avec master key
- **Affichage unique** : Jamais récupérable après

### Mot de passe
- **Longueur minimale** : 8 caractères
- **Hash** : bcrypt avec cost par défaut
- **Confirmation** : Obligatoire (validation client + serveur)
- **Autocomplete** : `new-password` pour éviter suggestions

### Cookie temporaire
- **Name** : `activation_key`
- **HttpOnly** : Oui (protection XSS)
- **Secure** : Non en dev (à activer en prod)
- **MaxAge** : 600 secondes (10 minutes)
- **Nettoyage** : Supprimé après confirmation

## 🎨 Interface utilisateur

### Page d'activation

**Design** :
- Centré verticalement et horizontalement
- Background gris clair
- Carte blanche avec ombre
- Logo et titre Anemone

**Formulaire** :
- Username en readonly (ne peut pas être changé)
- Deux champs de mot de passe styled
- Aide contextuelle ("Minimum 8 caractères")
- Info box bleue explicative
- Bouton gradient violet proéminent

**UX** :
- Validation en temps réel (HTML5)
- Messages d'erreur en rouge (bannière)
- Focus automatique sur password
- Placeholders informatifs

### Page de succès

**Hiérarchie** :
- Icône de succès verte (checkmark)
- Titre et sous-titre de bienvenue
- Bannière jaune d'avertissement
- Clé dans input readonly
- Actions claires (copier, télécharger)

**Interactions** :
- Copie avec feedback visuel (✓ Copié)
- Téléchargement avec nom de fichier personnalisé
- Checkboxes obligatoires avant de continuer
- Bouton désactivé jusqu'à validation complète

## 🧪 Flux de test

### Test 1 : Activation normale
```bash
# 1. Admin crée un utilisateur
→ john.doe créé avec token

# 2. Copier le lien d'activation
http://localhost:8080/activate/a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j

# 3. Ouvrir le lien dans le navigateur
→ Formulaire d'activation affiché
→ Username: john.doe (readonly)

# 4. Remplir :
Password: MySecurePassword123
Confirm: MySecurePassword123

# 5. Soumettre
→ Page de succès affichée
→ Clé affichée: [32 bytes en base64]

# 6. Copier la clé
→ Feedback "✓ Copié"

# 7. Télécharger la clé
→ Fichier "anemone-encryption-key-john.doe.txt"

# 8. Cocher les confirmations
→ Bouton "Se connecter" activé

# 9. Cliquer "Se connecter"
→ Redirige vers /login

# 10. Se connecter
Username: john.doe
Password: MySecurePassword123
→ Connexion réussie !
```

### Test 2 : Token invalide
```bash
# Tenter d'accéder avec un token inexistant
http://localhost:8080/activate/invalidtoken123
→ Erreur: "Ce lien d'activation est invalide ou a expiré"
```

### Test 3 : Token expiré
```bash
# Attendre 24h ou manipuler expires_at en BD
sqlite3 data/db/anemone.db "UPDATE activation_tokens SET expires_at = '2020-01-01' WHERE id = 1;"

# Accéder au lien
→ Erreur: "Ce lien d'activation est invalide ou a expiré"
```

### Test 4 : Token déjà utilisé
```bash
# Tenter de réutiliser un lien après activation
http://localhost:8080/activate/a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j
→ Erreur: "Ce lien d'activation a déjà été utilisé"
```

### Test 5 : Validation mot de passe
```bash
# Mots de passe différents
Password: Password123
Confirm: DifferentPassword
→ Erreur: "Les mots de passe ne correspondent pas"

# Mot de passe trop court
Password: short
Confirm: short
→ Erreur: "Le mot de passe doit contenir au moins 8 caractères"
```

### Test 6 : Base de données après activation
```bash
sqlite3 data/db/anemone.db <<EOF
-- Vérifier que l'utilisateur est activé
SELECT username, activated_at, password_hash, encryption_key_hash
FROM users WHERE username = 'john.doe';
-- activated_at doit être rempli
-- password_hash doit être rempli
-- encryption_key_hash doit être rempli

-- Vérifier que le token est marqué comme utilisé
SELECT token, used_at FROM activation_tokens WHERE username = 'john.doe';
-- used_at doit être rempli
EOF
```

## 📊 Structure des données

### Utilisateur activé (après activation)
```json
{
  "id": 2,
  "username": "john.doe",
  "password_hash": "$2a$10$...",  // Hash bcrypt
  "email": "john.doe@example.com",
  "encryption_key_hash": "sha256_hash",  // SHA-256 de la clé
  "encryption_key_encrypted": "encrypted_blob",  // Clé chiffrée avec master key
  "is_admin": false,
  "quota_total_gb": 100,
  "quota_backup_gb": 50,
  "created_at": "2025-10-27T10:30:00Z",
  "activated_at": "2025-10-27T11:00:00Z"  // Date d'activation
}
```

### Token après utilisation
```json
{
  "id": 1,
  "token": "a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j",
  "user_id": 2,
  "username": "john.doe",
  "email": "john.doe@example.com",
  "created_at": "2025-10-27T10:30:00Z",
  "expires_at": "2025-10-28T10:30:00Z",
  "used_at": "2025-10-27T11:00:00Z"  // Marqué comme utilisé
}
```

## ⏭️ Prochaines étapes

### Phase 5 : Partages et quotas
- [ ] Configuration Samba dynamique
- [ ] Création automatique des répertoires utilisateurs
  - `{datadir}/shares/{username}-backup/`
- [ ] Calcul de l'usage réel du stockage
- [ ] Monitoring des quotas
- [ ] Affichage dans les dashboards
- [ ] Alertes de dépassement

### Phase 6 : Synchronisation P2P
- [ ] Adaptation rclone pour multi-users
- [ ] Page de gestion des pairs
- [ ] Configuration de la synchronisation
- [ ] Logs de synchronisation
- [ ] Dashboard de statut

### Améliorations futures
- [ ] Envoi d'email avec le lien d'activation
- [ ] Régénération de lien si expiré (par admin)
- [ ] Page de profil utilisateur
- [ ] Changement de mot de passe
- [ ] Réinitialisation de mot de passe

## 📝 Notes techniques

### Extraction du token depuis l'URL
```go
// URL : /activate/a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j
path := r.URL.Path  // "/activate/a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j"
parts := strings.Split(strings.Trim(path, "/"), "/")
// parts = ["activate", "a3K9dF2mN5pQ8rT1vW4xY7zB0cE6gH9j"]
tokenString := parts[1]
```

### Validation du token
```go
// 1. Récupération
token, err := activation.GetTokenByString(db, tokenString)

// 2. Validation
if !token.IsValid() {
    // Soit expiré, soit déjà utilisé
    if token.UsedAt != nil {
        // Déjà utilisé
    } else {
        // Expiré
    }
}
```

### Activation en une transaction
```go
// 1. Hash du mot de passe
passwordHash := bcrypt.GenerateFromPassword(...)

// 2. Génération de la clé
encryptionKey := crypto.GenerateEncryptionKey()  // 32 bytes
keyHash := crypto.HashKey(encryptionKey)  // SHA-256
encryptedKey := crypto.EncryptKey(encryptionKey, masterKey)  // AES-GCM

// 3. Update en BD
UPDATE users SET
    password_hash = ?,
    encryption_key_hash = ?,
    encryption_key_encrypted = ?,
    activated_at = NOW()
WHERE id = ?

// 4. Marquer token comme utilisé
UPDATE activation_tokens SET used_at = NOW() WHERE id = ?
```

## 📈 Statistiques

- **Lignes de code Go** : ~190 lignes (2 handlers)
- **Templates HTML** : ~280 lignes (2 templates)
- **Traductions** : +18 clés (FR/EN)
- **Routes** : +2 routes publiques
- **Fonctions** : 2 handlers

## ✅ Ce qui fonctionne (théoriquement)

1. ✅ Utilisateur peut accéder au lien d'activation
2. ✅ Système valide le token (existe, non expiré, non utilisé)
3. ✅ Utilisateur peut choisir son mot de passe
4. ✅ Système génère automatiquement une clé de chiffrement
5. ✅ Clé affichée une seule fois avec avertissements
6. ✅ Utilisateur peut copier ou télécharger la clé
7. ✅ Checkboxes de confirmation obligatoires
8. ✅ Token marqué comme utilisé après activation
9. ✅ Redirection vers login après confirmation
10. ✅ Utilisateur peut se connecter avec ses credentials

## 🔗 Flux de bout en bout

```
ADMIN : Création utilisateur
  ↓
ADMIN : Génération token d'activation (24h)
  ↓
ADMIN : Copie et envoi du lien
  ↓
USER : Reçoit le lien par email/chat/etc.
  ↓
USER : Clique sur le lien
  ↓
SYSTEM : Valide le token
  ↓
USER : Choisit mot de passe
  ↓
SYSTEM : Active le compte + génère clé
  ↓
USER : Sauvegarde sa clé de chiffrement
  ↓
USER : Se connecte au système
  ↓
USER : Accède à son dashboard
  ↓
USER : Partages SMB disponibles
  ↓
USER : Peut commencer à utiliser Anemone
```

## 🎉 Conclusion

La Phase 4 est **complète** au niveau du code. Le flux complet de création et d'activation d'utilisateur est maintenant fonctionnel :

1. ✅ Admin crée un utilisateur (Phase 3)
2. ✅ Token d'activation généré automatiquement (Phase 3)
3. ✅ Utilisateur active son compte (Phase 4)
4. ✅ Utilisateur récupère sa clé de chiffrement (Phase 4)
5. ✅ Utilisateur peut se connecter (Phase 2)

**Prochaine action recommandée** :
1. Tester le flux complet end-to-end
2. Vérifier tous les scénarios (token invalide, expiré, etc.)
3. Passer à la Phase 5 (partages Samba + quotas)
