"""
Système de traductions pour Anemone
Supporte: fr (français), en (anglais)
"""

TRANSLATIONS = {
    "fr": {
        # Menu & Navigation
        "home": "Accueil",
        "peers": "Pairs",
        "api_status": "API Status",

        # Dashboard
        "dashboard_title": "Serveur de fichiers distribué et chiffré",
        "vpn_status": "Statut VPN",
        "storage": "Stockage",
        "disk": "Disque",
        "loading": "Chargement...",
        "last_backup": "Dernier backup",
        "status": "État",
        "active": "Actif",
        "inactive": "Inactif",
        "connected_peers": "Pairs connectés",
        "data_local": "Data (local)",
        "backup_saved": "Backup (sauvegardé)",
        "backups_received": "Backups (reçus)",
        "total": "Total",
        "used": "Utilisé",
        "free": "Libre",
        "loading_error": "Erreur de chargement",

        # Peers page
        "peers_management": "Gestion des Pairs",
        "wireguard_vpn_status": "Statut du VPN WireGuard",
        "my_invitation_code": "Mon Code d'Invitation",
        "share_code_desc": "Partagez ce code pour permettre à d'autres serveurs de se connecter",
        "protect_with_pin": "Protéger avec un PIN",
        "pin_4_8_digits": "PIN (4-8 chiffres)",
        "leave_empty_auto": "Laissez vide pour générer automatiquement",
        "generate_code": "Générer le Code",
        "json_code_to_share": "Code JSON à partager",
        "pin_to_communicate": "PIN à communiquer :",
        "communicate_pin_secure": "Communiquez ce PIN par téléphone, SMS ou autre canal sécurisé",
        "copy_code": "Copier le Code",
        "tip_send_json": "Astuce :",
        "tip_send_json_desc": "Envoyez le code JSON par email/WhatsApp, puis communiquez le PIN par téléphone pour une sécurité maximale !",
        "add_peer": "Ajouter un Pair",
        "paste_json_code": "Collez le code JSON reçu",
        "json_code": "Code JSON",
        "pin_if_protected": "PIN (si protégé)",
        "enter_pin": "Saisissez le PIN",
        "add_peer_button": "Ajouter le Pair",
        "connected_peers_list": "Pairs Connectés",
        "no_peers": "Aucun pair configuré",
        "connected": "Connecté",
        "disconnected": "Déconnecté",
        "remove": "Supprimer",
        "active_tunnels": "Tunnels actifs:",
        "endpoint": "Endpoint",
        "last_handshake": "Dernier handshake",
        "transfer": "Transfert",
        "never": "Jamais",
        "none": "Aucun",
        "configured_tunnels": "Tunnels configurés",
        "vpn_inactive": "Le VPN WireGuard n'est pas actif. Vérifiez les logs du conteneur wireguard.",
        "vpn_status_error": "Erreur de chargement du statut VPN",
        "restart_vpn": "Redémarrer VPN",

        # Alerts & Messages
        "pin_must_be_4_8": "Le PIN doit contenir entre 4 et 8 chiffres",
        "error": "Erreur",
        "code_copied": "Code copié dans le presse-papier",
        "copy_manual": "Sélectionnez le texte et utilisez Ctrl+C (ou Cmd+C sur Mac)",
        "enter_json_code": "Veuillez saisir le code JSON",
        "invalid_code": "Code invalide",
        "peer_added_success": "Pair ajouté avec succès !",
        "confirm_remove_peer": "Supprimer le pair",
        "remove_error": "Suppression impossible",

        # Setup pages
        "anemone": "Anemone",
        "initial_setup": "Configuration initiale",
        "new_server": "Nouveau serveur",
        "new_server_desc": "Générer une clé",
        "restore": "Restauration",
        "restore_desc": "J'ai déjà une clé",
        "continue": "Continuer",
        "key_generated": "Clé générée",
        "save_key_now": "SAUVEGARDEZ CETTE CLÉ MAINTENANT",
        "copy": "Copier",
        "download": "Télécharger",
        "i_saved_key": "J'ai sauvegardé ma clé",
        "restore_title": "Restauration",
        "paste_restic_key": "Collez votre clé Restic :",
        "from_bitwarden": "Depuis Bitwarden...",
        "validate": "Valider",
        "setup_complete": "Configuration terminée",
        "key_saved_securely": "La clé a été enregistrée de manière sécurisée",
        "page_wont_show_again": "Cette page ne s'affichera plus jamais",
        "dashboard": "Dashboard",
        "redirect_in": "Redirection dans 5s...",
    },

    "en": {
        # Menu & Navigation
        "home": "Home",
        "peers": "Peers",
        "api_status": "API Status",

        # Dashboard
        "dashboard_title": "Distributed and encrypted file server",
        "vpn_status": "VPN Status",
        "storage": "Storage",
        "disk": "Disk",
        "loading": "Loading...",
        "last_backup": "Last Backup",
        "status": "Status",
        "active": "Active",
        "inactive": "Inactive",
        "connected_peers": "Connected peers",
        "data_local": "Data (local)",
        "backup_saved": "Backup (saved)",
        "backups_received": "Backups (received)",
        "total": "Total",
        "used": "Used",
        "free": "Free",
        "loading_error": "Loading error",

        # Peers page
        "peers_management": "Peers Management",
        "wireguard_vpn_status": "WireGuard VPN Status",
        "my_invitation_code": "My Invitation Code",
        "share_code_desc": "Share this code to allow other servers to connect",
        "protect_with_pin": "Protect with PIN",
        "pin_4_8_digits": "PIN (4-8 digits)",
        "leave_empty_auto": "Leave empty to auto-generate",
        "generate_code": "Generate Code",
        "json_code_to_share": "JSON code to share",
        "pin_to_communicate": "PIN to communicate:",
        "communicate_pin_secure": "Communicate this PIN via phone, SMS or other secure channel",
        "copy_code": "Copy Code",
        "tip_send_json": "Tip:",
        "tip_send_json_desc": "Send the JSON code via email/WhatsApp, then communicate the PIN by phone for maximum security!",
        "add_peer": "Add Peer",
        "paste_json_code": "Paste the received JSON code",
        "json_code": "JSON Code",
        "pin_if_protected": "PIN (if protected)",
        "enter_pin": "Enter PIN",
        "add_peer_button": "Add Peer",
        "connected_peers_list": "Connected Peers",
        "no_peers": "No configured peers",
        "connected": "Connected",
        "disconnected": "Disconnected",
        "remove": "Remove",
        "active_tunnels": "Active tunnels:",
        "endpoint": "Endpoint",
        "last_handshake": "Last handshake",
        "transfer": "Transfer",
        "never": "Never",
        "none": "None",
        "configured_tunnels": "Configured tunnels",
        "vpn_inactive": "WireGuard VPN is not active. Check wireguard container logs.",
        "vpn_status_error": "VPN status loading error",
        "restart_vpn": "Restart VPN",

        # Alerts & Messages
        "pin_must_be_4_8": "PIN must contain between 4 and 8 digits",
        "error": "Error",
        "code_copied": "Code copied to clipboard",
        "copy_manual": "Select text and use Ctrl+C (or Cmd+C on Mac)",
        "enter_json_code": "Please enter JSON code",
        "invalid_code": "Invalid code",
        "peer_added_success": "Peer added successfully!",
        "confirm_remove_peer": "Remove peer",
        "remove_error": "Cannot remove",

        # Setup pages
        "anemone": "Anemone",
        "initial_setup": "Initial Setup",
        "new_server": "New server",
        "new_server_desc": "Generate a key",
        "restore": "Restore",
        "restore_desc": "I already have a key",
        "continue": "Continue",
        "key_generated": "Key generated",
        "save_key_now": "SAVE THIS KEY NOW",
        "copy": "Copy",
        "download": "Download",
        "i_saved_key": "I saved my key",
        "restore_title": "Restore",
        "paste_restic_key": "Paste your Restic key:",
        "from_bitwarden": "From Bitwarden...",
        "validate": "Validate",
        "setup_complete": "Setup Complete",
        "key_saved_securely": "The key has been saved securely",
        "page_wont_show_again": "This page will never show again",
        "dashboard": "Dashboard",
        "redirect_in": "Redirecting in 5s...",
    }
}

def get_text(key: str, lang: str = "fr") -> str:
    """
    Récupère une traduction

    Args:
        key: Clé de traduction
        lang: Code langue (fr ou en)

    Returns:
        Texte traduit ou la clé si non trouvé
    """
    if lang not in TRANSLATIONS:
        lang = "fr"  # Fallback to French

    return TRANSLATIONS[lang].get(key, key)

def get_all_texts(lang: str = "fr") -> dict:
    """
    Récupère toutes les traductions pour une langue

    Args:
        lang: Code langue (fr ou en)

    Returns:
        Dictionnaire de traductions
    """
    if lang not in TRANSLATIONS:
        lang = "fr"

    return TRANSLATIONS[lang]
