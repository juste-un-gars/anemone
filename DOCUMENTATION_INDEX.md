# 📚 Index de la documentation Anemone

Ce fichier vous guide vers la bonne documentation selon votre besoin.

## 🚀 Démarrage rapide

- **Nouveau utilisateur ?** → Lisez [README.md](README.md) pour une vue d'ensemble et l'installation
- **Premier setup ?** → Suivez la section "Configuration initiale sécurisée" du [README.md](README.md)

## 🔧 Problèmes et dépannage

- **Erreur pendant le setup ?** → [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Service qui ne démarre pas ?** → [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Erreur "Failed to decrypt key" ?** → [TROUBLESHOOTING.md](TROUBLESHOOTING.md#erreur--le-service-restic-ne-démarre-pas)

## 🤝 Contribuer au projet

- **Signaler un bug** → [CONTRIBUTING.md](CONTRIBUTING.md)
- **Proposer une fonctionnalité** → [CONTRIBUTING.md](CONTRIBUTING.md)
- **Comprendre la structure du code** → [CONTRIBUTING.md](CONTRIBUTING.md#structure-du-projet)
- **Standards de code** → [CONTRIBUTING.md](CONTRIBUTING.md#standards-de-code)

## 🔄 Migration et historique

- **Migration depuis ancienne version** → [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)
- **Historique des corrections** → [CORRECTIONS_APPLIQUEES.md](CORRECTIONS_APPLIQUEES.md)
- **Comprendre les problèmes résolus** → [CORRECTIONS_APPLIQUEES.md](CORRECTIONS_APPLIQUEES.md)

## 🤖 Pour Claude Code (développeurs IA)

- **Architecture du projet** → [CLAUDE.md](CLAUDE.md)
- **Commandes essentielles** → [CLAUDE.md](CLAUDE.md#essential-commands)
- **Problèmes courants** → [CLAUDE.md](CLAUDE.md#common-pitfalls)
- **Détails d'implémentation** → [CLAUDE.md](CLAUDE.md#important-implementation-details)

## 📖 Résumé des fichiers

| Fichier | Public cible | Contenu |
|---------|--------------|---------|
| [README.md](README.md) | Utilisateurs finaux | Vue d'ensemble, installation, utilisation, sécurité |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | Utilisateurs avec problèmes | Guide de dépannage complet, erreurs courantes |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributeurs | Comment contribuer, structure, standards de code |
| [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) | Utilisateurs existants | Migration depuis anciennes versions |
| [CORRECTIONS_APPLIQUEES.md](CORRECTIONS_APPLIQUEES.md) | Développeurs/curieux | Historique des bugs et corrections |
| [CLAUDE.md](CLAUDE.md) | IA/Développeurs avancés | Architecture technique détaillée |
| [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) | Tout le monde | Ce fichier - index de navigation |

## 🔍 Recherche rapide par sujet

### Sécurité
- Système de clés : [README.md](README.md#configuration-initiale-sécurisée)
- Meilleures pratiques : [README.md](README.md#meilleures-pratiques-de-sécurité)
- Checklist de sécurité : [README.md](README.md#checklist-de-sécurité)

### Installation
- Prérequis : [README.md](README.md#prérequis)
- Installation rapide : [README.md](README.md#installation)
- Configuration : [README.md](README.md#éditer-la-configuration)

### Développement
- Structure du projet : [CONTRIBUTING.md](CONTRIBUTING.md#structure-du-projet)
- Standards Python : [CONTRIBUTING.md](CONTRIBUTING.md#python-api)
- Standards Bash : [CONTRIBUTING.md](CONTRIBUTING.md#bash-scripts)
- Architecture Docker : [CLAUDE.md](CLAUDE.md#multi-service-docker-architecture)

### Erreurs spécifiques
- "Erreur lors du chiffrement" : [TROUBLESHOOTING.md](TROUBLESHOOTING.md#erreur--erreur-lors-du-chiffrement-lors-du-setup)
- "Failed to decrypt key" : [TROUBLESHOOTING.md](TROUBLESHOOTING.md#erreur--le-service-restic-ne-démarre-pas)
- Problème UUID/HOSTNAME : [CLAUDE.md](CLAUDE.md#critical-uuid-vs-hostname-container-restart-problem)
- Permission denied : [TROUBLESHOOTING.md](TROUBLESHOOTING.md#erreur--permission-denied-sur-procsyskernelrandomuuid)

### Concepts avancés
- Gestion des clés de chiffrement : [CLAUDE.md](CLAUDE.md#encryption-key-management-system)
- Modes de backup : [CLAUDE.md](CLAUDE.md#backup-modes)
- Migration cryptographie : [CLAUDE.md](CLAUDE.md#python-cryptography-migration)
- Réseau Docker : [CLAUDE.md](CLAUDE.md#multi-service-docker-architecture)

## ❓ Questions fréquentes

**Q : Où trouver les commandes Docker utiles ?**
R : [CLAUDE.md](CLAUDE.md#essential-commands) et [TROUBLESHOOTING.md](TROUBLESHOOTING.md#commandes-utiles-de-dépannage)

**Q : Comment tester mon installation ?**
R : [CLAUDE.md](CLAUDE.md#testing-the-setup-flow)

**Q : Que faire si j'ai perdu ma clé ?**
R : [README.md](README.md#que-se-passe-t-il-si) - Malheureusement, les backups sont irrécupérables

**Q : Comment migrer d'une ancienne version ?**
R : [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)

**Q : Quels problèmes ont été corrigés récemment ?**
R : [CORRECTIONS_APPLIQUEES.md](CORRECTIONS_APPLIQUEES.md)

## 🆘 Besoin d'aide ?

1. **Vérifiez d'abord** : [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
2. **Consultez les logs** : `docker-compose logs`
3. **Ouvrez une issue** sur GitHub avec :
   - Description du problème
   - Logs (sans informations sensibles)
   - Commandes exécutées
   - Informations système

## 📝 Contribuer à la documentation

La documentation peut être améliorée ! Si vous trouvez :
- Une erreur ou une imprécision
- Un point manquant ou peu clair
- Une nouvelle solution à un problème

→ Ouvrez une Pull Request ou une Issue sur GitHub

---

**Dernière mise à jour** : 2025-10-17
