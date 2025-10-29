# Anemone Scripts

## Configuration SMB automatique

### Problème
Lors de la création d'un utilisateur, Anemone régénère la configuration Samba et doit recharger le service `smbd`. Sans configuration, cela déclenche une popup sudo.

### Solution
Configurer sudoers pour autoriser le reload de smbd **sans mot de passe**.

### Installation

```bash
# En tant que root (ou avec sudo)
cd /path/to/anemone
sudo ./scripts/configure-smb-reload.sh

# Si anemone tourne sous un utilisateur spécifique (ex: anemone)
sudo ./scripts/configure-smb-reload.sh anemone
```

### Que fait le script ?

1. Crée `/etc/sudoers.d/anemone-smb`
2. Autorise l'utilisateur à exécuter `systemctl reload smbd` sans mot de passe
3. Vérifie la configuration avec `visudo`

### Vérification

```bash
# Test manuel (en tant qu'utilisateur anemone)
sudo systemctl reload smbd
# Ne doit pas demander de mot de passe
```

### Sécurité

✅ **Sécurisé** : Seule la commande `systemctl reload smbd` est autorisée
✅ **Limité** : Pas d'accès root complet
✅ **Standard** : Pratique courante pour les services système

### Désinstallation

```bash
sudo rm /etc/sudoers.d/anemone-smb
```
