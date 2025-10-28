// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package i18n

import (
	"sync"
)

// Translator handles internationalization
type Translator struct {
	translations map[string]map[string]string
	defaultLang  string
	mu           sync.RWMutex
}

var defaultTranslator *Translator
var once sync.Once

// Init initializes the default translator
func Init(defaultLang string) error {
	var err error
	once.Do(func() {
		defaultTranslator = &Translator{
			translations: make(map[string]map[string]string),
			defaultLang:  defaultLang,
		}
		err = defaultTranslator.loadTranslations()
	})
	return err
}

// T translates a key using the default translator
func T(lang, key string) string {
	if defaultTranslator == nil {
		return key
	}
	return defaultTranslator.Translate(lang, key)
}

// Translate returns the translation for a key in the specified language
func (t *Translator) Translate(lang, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if translations, ok := t.translations[lang]; ok {
		if translation, ok := translations[key]; ok {
			return translation
		}
	}

	// Fallback to default language
	if lang != t.defaultLang {
		if translations, ok := t.translations[t.defaultLang]; ok {
			if translation, ok := translations[key]; ok {
				return translation
			}
		}
	}

	// Return key if no translation found
	return key
}

// loadTranslations loads translation files from web/locales directory
func (t *Translator) loadTranslations() error {
	// For now, we'll embed translations directly
	// In production, these would be loaded from JSON files

	t.translations["fr"] = map[string]string{
		// Setup page
		"setup.title":              "Configuration initiale d'Anemone",
		"setup.welcome":            "Bienvenue sur Anemone",
		"setup.description":        "Configurons votre NAS multi-utilisateurs avec synchronisation P2P chiffrée",
		"setup.language":           "Langue",
		"setup.language.fr":        "Français",
		"setup.language.en":        "English",
		"setup.nas_name":           "Nom du NAS",
		"setup.nas_name.help":      "Un nom pour identifier ce serveur (ex: nas-maison)",
		"setup.timezone":           "Fuseau horaire",
		"setup.timezone.help":      "Utilisé pour les logs et la planification",
		"setup.admin.title":        "Créer le compte administrateur",
		"setup.admin.username":     "Nom d'utilisateur",
		"setup.admin.password":     "Mot de passe",
		"setup.admin.password.help": "Minimum 8 caractères",
		"setup.admin.password_confirm": "Confirmer le mot de passe",
		"setup.admin.email":        "Email (optionnel)",
		"setup.admin.email.help":   "Pour les notifications (optionnel)",
		"setup.button":             "Créer et démarrer",
		"setup.button.submit":      "Créer et démarrer",
		"setup.info":               "Cette configuration ne sera effectuée qu'une seule fois. Assurez-vous de bien sauvegarder vos informations.",
		"setup.errors.required":    "Ce champ est requis",
		"setup.errors.password_mismatch": "Les mots de passe ne correspondent pas",
		"setup.errors.password_length": "Le mot de passe doit contenir au moins 8 caractères",
		"setup.success.title":      "Configuration terminée !",
		"setup.success.key":        "Votre clé de chiffrement",
		"setup.success.key_warning": "⚠️ Attention : Cette clé ne sera affichée qu'une seule fois !",
		"setup.success.key_help":   "Sauvegardez cette clé dans un endroit sûr. Elle est nécessaire pour déchiffrer vos sauvegardes.",
		"setup.success.download":   "Télécharger la clé",
		"setup.success.checkbox1":  "J'ai sauvegardé ma clé de chiffrement",
		"setup.success.checkbox2":  "Je comprends que je ne peux pas récupérer mes données sans cette clé",
		"setup.success.button":     "Accéder au tableau de bord",

		// Login page
		"login.title":              "Connexion à Anemone",
		"login.welcome":            "Bienvenue",
		"login.username":           "Nom d'utilisateur",
		"login.password":           "Mot de passe",
		"login.button":             "Se connecter",
		"login.error":              "Nom d'utilisateur ou mot de passe incorrect",

		// Dashboard
		"dashboard.title":          "Tableau de bord",
		"dashboard.welcome":        "Bienvenue, {{username}} !",
		"dashboard.admin.title":    "Tableau de bord administrateur",
		"dashboard.user.title":     "Tableau de bord utilisateur",
		"dashboard.logout":         "Se déconnecter",
		"dashboard.users":          "Utilisateurs",
		"dashboard.peers":          "Pairs connectés",
		"dashboard.storage":        "Stockage",
		"dashboard.backup":         "Sauvegardes",
		"dashboard.settings":       "Paramètres",
		"dashboard.stats.users":    "Utilisateurs",
		"dashboard.stats.storage":  "Stockage utilisé",
		"dashboard.stats.backups":  "Dernière sauvegarde",
		"dashboard.stats.peers":    "Pairs actifs",

		// Users management
		"users.title":              "Gestion des utilisateurs",
		"users.add":                "Ajouter un utilisateur",
		"users.add.title":          "Créer un nouvel utilisateur",
		"users.list":               "Liste des utilisateurs",
		"users.username":           "Nom d'utilisateur",
		"users.email":              "Email",
		"users.role":               "Rôle",
		"users.role.admin":         "Administrateur",
		"users.role.user":          "Utilisateur",
		"users.created":            "Créé le",
		"users.last_login":         "Dernière connexion",
		"users.status":             "Statut",
		"users.status.active":      "Actif",
		"users.status.pending":     "En attente",
		"users.actions":            "Actions",
		"users.action.delete":      "Supprimer",
		"users.action.copy_link":   "Copier le lien",
		"users.quota_total":        "Quota total (GB)",
		"users.quota_backup":       "Quota sauvegarde (GB)",
		"users.is_admin":           "Administrateur",
		"users.token.title":        "Lien d'activation créé",
		"users.token.info":         "Envoyez ce lien au nouvel utilisateur",
		"users.token.warning":      "⚠️ Ce lien expire dans 24 heures",
		"users.token.copy":         "Copier le lien",
		"users.token.done":         "Terminé",
		"users.delete.confirm":     "Êtes-vous sûr de vouloir supprimer cet utilisateur ?",
		"users.errors.username_required": "Le nom d'utilisateur est requis",
		"users.errors.username_exists": "Ce nom d'utilisateur existe déjà",

		// Activation page
		"activate.title":           "Activer votre compte Anemone",
		"activate.welcome":         "Bienvenue sur Anemone",
		"activate.username":        "Nom d'utilisateur",
		"activate.set_password":    "Choisissez votre mot de passe",
		"activate.password":        "Mot de passe",
		"activate.password_confirm": "Confirmer le mot de passe",
		"activate.password.help":   "Minimum 8 caractères",
		"activate.button":          "Activer mon compte",
		"activate.errors.invalid_token": "Ce lien d'activation est invalide ou a expiré",
		"activate.errors.token_used": "Ce lien d'activation a déjà été utilisé",
		"activate.errors.password_mismatch": "Les mots de passe ne correspondent pas",
		"activate.errors.password_length": "Le mot de passe doit contenir au moins 8 caractères",
		"activate.success.title":   "Compte activé avec succès !",
		"activate.success.key":     "Votre clé de chiffrement",
		"activate.success.key_warning": "⚠️ IMPORTANT : Cette clé ne sera affichée qu'une seule fois !",
		"activate.success.key_help": "Cette clé est nécessaire pour déchiffrer vos sauvegardes. Sauvegardez-la dans un endroit sûr.",
		"activate.success.checkbox1": "J'ai sauvegardé ma clé de chiffrement en lieu sûr",
		"activate.success.checkbox2": "Je comprends que cette clé ne peut pas être récupérée",
		"activate.success.button":  "Se connecter",

		// Common
		"common.submit":            "Envoyer",
		"common.cancel":            "Annuler",
		"common.save":              "Enregistrer",
		"common.delete":            "Supprimer",
		"common.edit":              "Modifier",
		"common.back":              "Retour",
		"common.loading":           "Chargement...",
		"common.error":             "Erreur",
		"common.success":           "Succès",
	}

	t.translations["en"] = map[string]string{
		// Setup page
		"setup.title":              "Anemone Initial Setup",
		"setup.welcome":            "Welcome to Anemone",
		"setup.description":        "Let's configure your multi-user NAS with P2P encrypted synchronization",
		"setup.language":           "Language",
		"setup.language.fr":        "Français",
		"setup.language.en":        "English",
		"setup.nas_name":           "NAS Name",
		"setup.nas_name.help":      "A name to identify this server (e.g., home-nas)",
		"setup.timezone":           "Timezone",
		"setup.timezone.help":      "Used for logs and scheduling",
		"setup.admin.title":        "Create Administrator Account",
		"setup.admin.username":     "Username",
		"setup.admin.password":     "Password",
		"setup.admin.password.help": "Minimum 8 characters",
		"setup.admin.password_confirm": "Confirm Password",
		"setup.admin.email":        "Email (optional)",
		"setup.admin.email.help":   "For notifications (optional)",
		"setup.button":             "Create and Start",
		"setup.button.submit":      "Create and Start",
		"setup.info":               "This setup will only be performed once. Make sure to save your information carefully.",
		"setup.errors.required":    "This field is required",
		"setup.errors.password_mismatch": "Passwords do not match",
		"setup.errors.password_length": "Password must be at least 8 characters",
		"setup.success.title":      "Setup Complete!",
		"setup.success.key":        "Your Encryption Key",
		"setup.success.key_warning": "⚠️ Warning: This key will only be displayed once!",
		"setup.success.key_help":   "Save this key in a safe place. It is required to decrypt your backups.",
		"setup.success.download":   "Download Key",
		"setup.success.checkbox1":  "I have saved my encryption key",
		"setup.success.checkbox2":  "I understand that I cannot recover my data without this key",
		"setup.success.button":     "Go to Dashboard",

		// Login page
		"login.title":              "Anemone Login",
		"login.welcome":            "Welcome",
		"login.username":           "Username",
		"login.password":           "Password",
		"login.button":             "Sign In",
		"login.error":              "Invalid username or password",

		// Dashboard
		"dashboard.title":          "Dashboard",
		"dashboard.welcome":        "Welcome, {{username}}!",
		"dashboard.admin.title":    "Admin Dashboard",
		"dashboard.user.title":     "User Dashboard",
		"dashboard.logout":         "Logout",
		"dashboard.users":          "Users",
		"dashboard.peers":          "Connected Peers",
		"dashboard.storage":        "Storage",
		"dashboard.backup":         "Backups",
		"dashboard.settings":       "Settings",
		"dashboard.stats.users":    "Users",
		"dashboard.stats.storage":  "Storage Used",
		"dashboard.stats.backups":  "Last Backup",
		"dashboard.stats.peers":    "Active Peers",

		// Users management
		"users.title":              "User Management",
		"users.add":                "Add User",
		"users.add.title":          "Create New User",
		"users.list":               "User List",
		"users.username":           "Username",
		"users.email":              "Email",
		"users.role":               "Role",
		"users.role.admin":         "Administrator",
		"users.role.user":          "User",
		"users.created":            "Created",
		"users.last_login":         "Last Login",
		"users.status":             "Status",
		"users.status.active":      "Active",
		"users.status.pending":     "Pending",
		"users.actions":            "Actions",
		"users.action.delete":      "Delete",
		"users.action.copy_link":   "Copy Link",
		"users.quota_total":        "Total Quota (GB)",
		"users.quota_backup":       "Backup Quota (GB)",
		"users.is_admin":           "Administrator",
		"users.token.title":        "Activation Link Created",
		"users.token.info":         "Send this link to the new user",
		"users.token.warning":      "⚠️ This link expires in 24 hours",
		"users.token.copy":         "Copy Link",
		"users.token.done":         "Done",
		"users.delete.confirm":     "Are you sure you want to delete this user?",
		"users.errors.username_required": "Username is required",
		"users.errors.username_exists": "This username already exists",

		// Activation page
		"activate.title":           "Activate Your Anemone Account",
		"activate.welcome":         "Welcome to Anemone",
		"activate.username":        "Username",
		"activate.set_password":    "Choose Your Password",
		"activate.password":        "Password",
		"activate.password_confirm": "Confirm Password",
		"activate.password.help":   "Minimum 8 characters",
		"activate.button":          "Activate My Account",
		"activate.errors.invalid_token": "This activation link is invalid or has expired",
		"activate.errors.token_used": "This activation link has already been used",
		"activate.errors.password_mismatch": "Passwords do not match",
		"activate.errors.password_length": "Password must be at least 8 characters",
		"activate.success.title":   "Account Successfully Activated!",
		"activate.success.key":     "Your Encryption Key",
		"activate.success.key_warning": "⚠️ IMPORTANT: This key will only be displayed once!",
		"activate.success.key_help": "This key is required to decrypt your backups. Save it in a safe place.",
		"activate.success.checkbox1": "I have saved my encryption key in a safe place",
		"activate.success.checkbox2": "I understand that this key cannot be recovered",
		"activate.success.button":  "Sign In",

		// Common
		"common.submit":            "Submit",
		"common.cancel":            "Cancel",
		"common.save":              "Save",
		"common.delete":            "Delete",
		"common.edit":              "Edit",
		"common.back":              "Back",
		"common.loading":           "Loading...",
		"common.error":             "Error",
		"common.success":           "Success",
	}

	return nil
}

// GetAvailableLanguages returns list of available languages
func GetAvailableLanguages() []Language {
	return []Language{
		{Code: "fr", Name: "Français"},
		{Code: "en", Name: "English"},
	}
}

// Language represents an available language
type Language struct {
	Code string
	Name string
}
