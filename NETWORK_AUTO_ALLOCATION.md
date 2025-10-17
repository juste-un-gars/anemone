# 🌐 Allocation automatique du subnet Docker

## Problème résolu

**Avant** : Anemone utilisait un subnet Docker fixe (172.20.0.0/16, puis 172.25.0.0/16, etc.) ce qui causait régulièrement l'erreur :

```
Error response from daemon: failed to set up container networking: Address already in use
```

**Après** : Docker choisit automatiquement un subnet libre parmi les ranges disponibles.

## Changements effectués

### docker-compose.yml

**Avant** :
```yaml
version: '3.8'

services:
  wireguard:
    ...
    networks:
      anemone-net:
        ipv4_address: 172.25.0.2  # IP statique
    ...

networks:
  anemone-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.25.0.0/16  # Subnet fixe
```

**Après** :
```yaml
services:
  wireguard:
    ...
    networks:
      - anemone-net  # IP dynamique
    ...

networks:
  anemone-net:
    driver: bridge  # Pas de config IPAM = auto
```

## Pourquoi ça fonctionne

1. **Aucune IP statique requise** : Les conteneurs n'ont pas besoin d'IPs fixes
2. **Service réseau partagé** : Restic utilise `network_mode: "service:wireguard"` donc partage la stack réseau de WireGuard
3. **Communication interne** : Les services utilisent les noms de conteneur pour communiquer (`anemone-api`, `anemone-samba`, etc.)
4. **Docker intelligent** : Docker parcourt automatiquement les ranges 172.17-31.0.0/16 et 192.168.0-255.0/24 pour trouver un subnet libre

## Impact

### Positif ✅
- Plus de conflits réseau entre projets Docker
- Installation simplifiée (pas besoin de modifier le subnet)
- Portable entre machines

### Neutre ⚠️
- Les IPs internes peuvent changer entre les redémarrages
- Non problématique car :
  - Communication par nom de conteneur, pas par IP
  - Restic partage la stack réseau de WireGuard
  - Pas de référence hardcodée aux IPs dans le code

### Négatif ❌
- Aucun impact négatif identifié

## Debugging réseau

Si vous avez besoin de connaître le subnet alloué :

```bash
# Voir le subnet utilisé
docker network inspect anemone_anemone-net | grep Subnet

# Voir l'IP d'un conteneur
docker inspect anemone-wireguard | grep IPAddress

# Voir tous les réseaux Docker
docker network ls
```

## Migration depuis ancienne version

Si vous avez une installation existante avec subnet fixe :

```bash
# 1. Arrêter les conteneurs
docker compose down

# 2. Supprimer l'ancien réseau (optionnel mais recommandé)
docker network rm anemone_anemone-net

# 3. Mettre à jour docker-compose.yml (déjà fait dans cette version)

# 4. Redémarrer
docker compose up -d
```

## Références

- [Docker networking documentation](https://docs.docker.com/network/)
- [Best practices for Docker Compose networks](https://docs.docker.com/compose/networking/)
- Issue GitHub : "Address already in use on multiple machines"

---

**Date de changement** : 2025-10-17
**Version** : Post-migration allocation automatique
