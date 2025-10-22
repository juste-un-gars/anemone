# 🤝 Contribuer à Anemone

Merci de votre intérêt pour contribuer à Anemone ! Ce document vous guidera pour contribuer efficacement.

## 📋 Table des matières

- [Code de conduite](#code-de-conduite)
- [Comment contribuer](#comment-contribuer)
- [Structure du projet](#structure-du-projet)
- [Développement local](#développement-local)
- [Standards de code](#standards-de-code)
- [Processus de Pull Request](#processus-de-pull-request)

## 🌟 Code de conduite

Soyez respectueux et constructif dans vos interactions. Nous voulons une communauté accueillante pour tous.

## 📄 Licence et Contributions

**En contribuant à ce projet, vous acceptez que :**

1. ✅ Vos contributions soient publiées sous la **GNU Affero General Public License v3.0** (AGPLv3)
2. ✅ Vos contributions deviennent partie intégrante d'Anemone, un logiciel libre et open source
3. ✅ Vous détenez les droits nécessaires pour contribuer ce code
4. ✅ Votre contribution n'enfreint aucun brevet, copyright ou autre droit de propriété intellectuelle

L'AGPLv3 garantit que :
- Le code reste libre et gratuit
- Toute modification doit être partagée avec la communauté
- Même un service web utilisant Anemone doit publier son code source
- Les prestations de service commerciales sont autorisées (installation, maintenance, support)

**Note importante** : L'AGPLv3 est une licence Open Source qui **autorise l'usage commercial**. Cela signifie que n'importe qui peut utiliser, modifier et même vendre Anemone, tant qu'ils respectent les termes de la licence (notamment le partage du code source modifié).

Pour plus d'informations, consultez le fichier [LICENSE](LICENSE).

## 🚀 Comment contribuer

### Signaler un bug

1. Vérifiez que le bug n'a pas déjà été signalé dans les [Issues](https://github.com/juste-un-gars/anemone/issues)
2. Créez une nouvelle issue avec le label `bug`
3. Décrivez le problème avec :
   - Version d'Anemone
   - Système d'exploitation
   - Étapes pour reproduire
   - Comportement attendu vs observé
   - Logs pertinents

### Proposer une fonctionnalité

1. Ouvrez une issue avec le label `enhancement`
2. Décrivez clairement :
   - Le besoin / cas d'usage
   - La solution proposée
   - Les alternatives considérées

### Contribuer du code

1. **Forkez** le dépôt
2. **Créez une branche** depuis `main` :
   ```bash
   git checkout -b feature/ma-fonctionnalite
   ```
3. **Commitez** vos changements (voir [Standards de commits](#standards-de-commits))
4. **Pushez** vers votre fork
5. **Ouvrez une Pull Request**

## 📁 Structure du projet

```
anemone/
├── config/                   # Configuration
├── services/
│   ├── core/                # Service principal (VPN + SFTP + Restic)
│   │   ├── Dockerfile
│   │   ├── entrypoint.sh
│   │   ├── supervisord.conf
│   │   ├── restic-scripts/  # Scripts de backup Restic
│   │   └── scripts/         # Scripts de backup config auto
│   ├── shares/              # SMB + WebDAV (optionnel)
│   └── api/                 # API & Dashboard
│       ├── Dockerfile
│       ├── requirements.txt
│       ├── main.py
│       └── templates/       # Templates web (dont recovery.html)
├── scripts/                 # Scripts utilitaires
│   ├── init.sh             # Initialisation
│   ├── add-peer.sh         # Ajout de peer interactif
│   ├── restore-config.py   # Restauration de configuration
│   ├── discover-backups.py # Découverte backups sur peers
│   └── test-*.sh           # Suites de tests
├── docs/
│   └── archive/            # Documentation historique
├── DISASTER_RECOVERY.md    # Guide disaster recovery complet
└── docker-compose.yml
```

## 💻 Développement local

### Prérequis

- Docker & Docker Compose
- Git
- Python 3.11+ (pour développement API)
- Bash (pour scripts)

### Installation

```bash
# Cloner le dépôt
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Créer les fichiers
bash setup-project.sh

# Initialiser
./scripts/init.sh

# Lancer en mode développement
docker-compose up --build
```

### Tester vos modifications

```bash
# Logs en temps réel
docker-compose logs -f

# Tester un service spécifique
docker-compose logs -f restic

# Redémarrer après modification
docker-compose restart <service>

# Rebuild complet
docker-compose down
docker-compose up --build
```

## 📝 Standards de code

### Python (API)

- **Style** : PEP 8
- **Formatage** : Black
- **Linting** : Flake8
- **Type hints** : Obligatoires

```bash
# Formater le code
black services/api/

# Vérifier
flake8 services/api/
```

### Bash (Scripts)

- **Shebang** : `#!/bin/bash`
- **Set options** : `set -e` (exit on error)
- **Indentation** : 4 espaces
- **Variables** : `${VAR}` (toujours entre accolades)
- **Commentaires** : Décrire le "pourquoi", pas le "quoi"

### Docker

- **Image de base** : Alpine Linux (sauf si incompatible)
- **Multi-stage builds** : Quand applicable
- **Layers** : Minimiser le nombre de couches
- **Sécurité** : Pas de `root` si possible

### YAML

- **Indentation** : 2 espaces
- **Commentaires** : Au-dessus de la clé, pas à côté
- **Ordre** : Alphabétique dans chaque section

## 🔀 Processus de Pull Request

### Avant de soumettre

- [ ] Le code compile/fonctionne localement
- [ ] Les tests passent (quand applicable)
- [ ] La documentation est mise à jour
- [ ] Les commits suivent les conventions
- [ ] Pas de conflits avec `main`

### Standards de commits

Utilisez [Conventional Commits](https://www.conventionalcommits.org/) :

```
<type>(<scope>): <description>

[body optionnel]

[footer optionnel]
```

**Types** :
- `feat`: Nouvelle fonctionnalité
- `fix`: Correction de bug
- `docs`: Documentation
- `style`: Formatage (pas de changement de code)
- `refactor`: Refactoring
- `perf`: Amélioration de performance
- `test`: Ajout de tests
- `chore`: Tâches de maintenance

**Exemples** :
```
feat(backup): add live backup mode with inotify
fix(api): correct disk usage calculation
docs(readme): update installation instructions
refactor(restic): simplify backup script logic
```

### Revue de code

Votre PR sera revue selon :
- ✅ Qualité du code
- ✅ Tests (si applicable)
- ✅ Documentation
- ✅ Cohérence avec l'existant
- ✅ Impact sur les performances

## 🐛 Déboguer

### Logs utiles

```bash
# Tous les services
docker-compose logs -f

# Service spécifique
docker-compose logs -f restic

# Logs de backup
cat logs/backup.log

# État WireGuard
docker exec anemone-wireguard wg show

# État Restic
docker exec anemone-restic restic snapshots
```

### Conteneurs interactifs

```bash
# Shell dans un conteneur
docker exec -it anemone-restic /bin/bash

# Tester une commande
docker exec anemone-restic restic -r <repo> snapshots
```

## 📚 Ressources

- [Documentation WireGuard](https://www.wireguard.com/)
- [Documentation Restic](https://restic.readthedocs.io/)
- [FastAPI](https://fastapi.tiangolo.com/)
- [Docker Compose](https://docs.docker.com/compose/)

## 🎯 Priorités actuelles

Consultez les [Issues](https://github.com/juste-un-gars/anemone/issues) avec les labels :
- `good first issue` : Bon pour débuter
- `help wanted` : Aide bienvenue
- `priority:high` : Urgent

## ❓ Questions

- 💬 Ouvrez une [Discussion](https://github.com/juste-un-gars/anemone/discussions)
- 📧 Contactez les mainteneurs
- 📖 Consultez la [Wiki](https://github.com/juste-un-gars/anemone/wiki)

---

Merci de contribuer à Anemone ! 🪸
