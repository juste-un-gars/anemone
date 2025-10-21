# 📚 Index de Documentation - Anemone

Bienvenue dans la documentation Anemone ! Ce guide vous aidera à trouver rapidement l'information dont vous avez besoin.

---

## 🚀 Pour Commencer

**Vous débutez avec Anemone ?** Commencez ici :

1. **[README.md](README.md)** - Guide de démarrage rapide
   - Installation en 5 minutes
   - Configuration initiale sécurisée
   - Fonctionnalités principales
   - FAQ sécurité

2. **[ARCHITECTURE.md](ARCHITECTURE.md)** - Comprendre le système
   - Architecture générale
   - Services et leurs rôles
   - Flux de données

---

## 🔧 Guides de Configuration

### Connexion entre Serveurs

**[INTERCONNEXION_GUIDE.md](INTERCONNEXION_GUIDE.md)** - Le guide complet
- ✅ Méthode recommandée : QR Code via interface web
- ✅ Méthode manuelle : Script interactif
- ✅ Échange de clés sécurisé
- ✅ Configuration WireGuard VPN
- ✅ Tests de connectivité

### Partages de Fichiers

**[EXTERNAL_SHARES.md](EXTERNAL_SHARES.md)**
- Configuration SMB (Windows/Mac/Linux)
- Configuration WebDAV
- Permissions et sécurité

---

## 🆘 Disaster Recovery

### Guide Complet

**[DISASTER_RECOVERY.md](DISASTER_RECOVERY.md)** - ⭐ ESSENTIEL
- **Phase 1** : Export/Import manuel
- **Phase 2** : Backup automatique vers peers
- **Phase 3** : Interface web + fonctionnalités avancées

**Ce guide couvre :**
- ✅ Export de configuration chiffrée
- ✅ Restauration complète d'un serveur
- ✅ Backup automatique quotidien
- ✅ Auto-restore depuis les peers
- ✅ Interface web de recovery
- ✅ Notifications optionnelles (email/webhook)
- ✅ Backup incrémentiel
- ✅ Vérification d'intégrité
- ✅ Historique multi-versions

### Accès Rapide

```bash
# Export de configuration
http://localhost:3000/api/config/export

# Interface web de recovery
http://localhost:3000/recovery

# Restauration automatique
./start.sh --auto-restore

# Restauration depuis fichier local
./start.sh --restore-from=backup.enc
```

---

## 🐛 Dépannage

### Guide Général

**[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**
- Problèmes courants et solutions
- Commandes de diagnostic
- Vérification de l'état des services
- Problèmes de performance

### Problèmes Spécifiques

**[VPN_TROUBLESHOOTING.md](VPN_TROUBLESHOOTING.md)**
- Diagnostic WireGuard
- Problèmes de connectivité VPN
- Tests de réseau entre pairs

### Scripts de Diagnostic

```bash
# Diagnostic VPN complet
./scripts/diagnose-vpn.sh

# Afficher les clés (publiques uniquement)
./scripts/show-keys.sh

# Redémarrer le VPN
./scripts/restart-vpn.sh
```

---

## 🛠️ Documentation Technique

### Architecture Réseau

**[WIREGUARD_ARCHITECTURE.md](WIREGUARD_ARCHITECTURE.md)**
- Architecture du VPN mesh
- Configuration réseau
- Sécurité WireGuard

**[WIREGUARD_SETUP.md](WIREGUARD_SETUP.md)**
- Setup détaillé WireGuard
- Génération de clés
- Configuration avancée

**[NETWORK_AUTO_ALLOCATION.md](NETWORK_AUTO_ALLOCATION.md)**
- Allocation automatique des IPs VPN
- Éviter les conflits réseau
- Configuration Docker network

### Migration

**[MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)**
- Migration depuis anciennes versions
- Mise à jour de configuration
- Compatibilité

---

## 🤝 Contribuer

**[CONTRIBUTING.md](CONTRIBUTING.md)**
- Comment contribuer au projet
- Standards de code
- Structure du projet à jour
- Processus de Pull Request
- Conventions de commits

**[CLAUDE.md](CLAUDE.md)** - Instructions pour Claude Code
- Directives pour l'IA Claude Code
- Architecture du projet
- Conventions spécifiques

---

## 📖 Documentation Historique

Les documents suivants sont archivés mais conservés pour référence :

### `docs/archive/`

- **CHANGELOG_WIREGUARD_FIX.md** - Historique des corrections WireGuard
- **CORRECTIONS_APPLIQUEES.md** - Corrections techniques appliquées
- **WIREGUARD_KEY_FIX.md** - Correction du système de clés
- **ORDRE_INITIALISATION.md** - Ordre d'initialisation des services
- **PEERS_GUIDE.md** - Ancien guide de peers (remplacé par INTERCONNEXION_GUIDE.md)
- **PHASE1_IMPLEMENTATION_SUMMARY.md** - Résumé technique Phase 1
- **PHASE2_IMPLEMENTATION_SUMMARY.md** - Résumé technique Phase 2
- **PHASE3_IMPLEMENTATION_SUMMARY.md** - Résumé technique Phase 3

---

## 🔎 Guide par Tâche

### Je veux... installer Anemone
→ [README.md](README.md) - Section Installation rapide

### Je veux... connecter deux serveurs
→ [INTERCONNEXION_GUIDE.md](INTERCONNEXION_GUIDE.md) - Méthode QR Code

### Je veux... sauvegarder ma configuration
→ [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md) - Section Export

### Je veux... restaurer un serveur perdu
→ [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md) - Section Auto-Restore
→ OU directement : `./start.sh --auto-restore`

### Je veux... accéder à mes fichiers via SMB
→ [EXTERNAL_SHARES.md](EXTERNAL_SHARES.md) - Section SMB

### Je veux... déboguer un problème de VPN
→ [VPN_TROUBLESHOOTING.md](VPN_TROUBLESHOOTING.md)
→ OU lancer : `./scripts/diagnose-vpn.sh`

### Je veux... configurer des notifications de backup
→ [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md) - Section Notifications

### Je veux... vérifier l'intégrité de mes backups
→ Interface web : `http://localhost:3000/recovery`
→ OU [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md) - Section Vérification

### Je veux... comprendre l'architecture
→ [ARCHITECTURE.md](ARCHITECTURE.md)

### Je veux... contribuer au projet
→ [CONTRIBUTING.md](CONTRIBUTING.md)

---

## 📱 Interfaces Web

| URL | Description |
|-----|-------------|
| `http://localhost:3000/` | Dashboard principal |
| `http://localhost:3000/setup` | Configuration initiale (première fois) |
| `http://localhost:3000/peers` | Gestion des pairs + QR Code |
| `http://localhost:3000/recovery` | ⭐ Interface de Disaster Recovery |
| `http://localhost:3000/api/config/export` | Export de configuration |

---

## 🎯 Commandes Essentielles

```bash
# Démarrage
./start.sh                              # Initialise et démarre tout

# Disaster Recovery
./start.sh --auto-restore               # Restauration automatique
./start.sh --restore-from=backup.enc    # Restauration depuis fichier

# Gestion des peers
./scripts/add-peer.sh                   # Ajouter un peer (interactif)

# Diagnostic
./scripts/diagnose-vpn.sh               # Diagnostic VPN complet
./scripts/show-keys.sh                  # Afficher les clés publiques

# Docker
docker compose up -d                    # Démarrer les services
docker compose down                     # Arrêter les services
docker compose logs -f                  # Voir les logs en temps réel
docker compose logs -f core             # Logs du service core
```

---

## 📊 Statistiques de Documentation

- **Guides principaux** : 6 fichiers
- **Guides techniques** : 5 fichiers
- **Documents archivés** : 9 fichiers
- **Lignes totales** : ~3500+
- **Dernière mise à jour** : 2025-10-21

---

**💡 Astuce** : Utilisez Ctrl+F dans votre navigateur pour rechercher rapidement dans ce document !

**🆘 Besoin d'aide ?** Consultez d'abord [TROUBLESHOOTING.md](TROUBLESHOOTING.md) ou ouvrez une [Issue GitHub](https://github.com/juste-un-gars/anemone/issues).
