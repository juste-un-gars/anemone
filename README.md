# 🪸 Anemone

**Serveur de fichiers distribué, simple et chiffré, avec redondance entre proches**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![WireGuard](https://img.shields.io/badge/wireguard-%2388171A.svg?style=flat&logo=wireguard&logoColor=white)](https://www.wireguard.com/)

## 🎯 Qu'est-ce qu'Anemone ?

Anemone est un système de stockage distribué qui permet de :

- 📂 **Servir vos fichiers localement** via SMB, WebDAV ou SFTP
- 🔐 **Sauvegarder automatiquement** vos données de manière chiffrée
- 🤝 **Échanger des backups** avec vos proches via un VPN sécurisé
- 🚀 **Déployer en 5 minutes** avec Docker

### Cas d'usage typique

Alice, Bob et Charlie sont amis. Chacun héberge Anemone chez lui :
- Alice a 2 To de données → sauvegardées chez Bob et Charlie
- Bob a 1 To de données → sauvegardées chez Alice et Charlie  
- Charlie a 500 Go de données → sauvegardées chez Alice et Bob

**Tout est chiffré côté client. Personne ne peut lire les backups des autres.**

## ✨ Fonctionnalités

### Stockage local
- ✅ Partage réseau SMB (Windows/macOS/Linux)
- ✅ Accès WebDAV (navigateur, mobile, rclone)
- ✅ SFTP optionnel (accès technique)

### Backup distribué
- ✅ Chiffrement bout-à-bout (Restic)
- ✅ Déduplication automatique
- ✅ Backup incrémental
- ✅ Choix du mode : live, périodique ou planifié
- ✅ Rétention configurable

### Sécurité
- ✅ VPN WireGuard entre pairs
- ✅ Authentification par clés publiques
- ✅ Isolation Docker complète
- ✅ Aucun accès en clair aux données distantes

## 🚀 Installation rapide

### Prérequis

- Docker & Docker Compose
- 1 Go RAM minimum
- Port UDP 51820 ouvert (port-forwarding sur votre box)
- Un nom de domaine DynDNS (gratuit : [DuckDNS](https://www.duckdns.org), [No-IP](https://www.noip.com))

### Installation

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
./scripts/init.sh
# Éditez config/config.yaml et .env
docker-compose up -d
```

## 📖 Documentation complète

Consultez le [wiki](https://github.com/juste-un-gars/anemone/wiki) pour :
- Guide d'installation détaillé
- Configuration avancée
- Dépannage
- FAQ

## 🤝 Contribuer

Les contributions sont les bienvenues ! Consultez [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 Licence

MIT License - voir [LICENSE](LICENSE)
