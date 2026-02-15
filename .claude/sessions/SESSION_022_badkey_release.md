# Session 22: !BADKEY Fix + Bugfixes + Release v0.20.0-beta

## Meta
- **Date:** 2026-02-15
- **Goal:** Corriger !BADKEY slog, fix bugs installation, release v0.20.0-beta
- **Status:** Complete

## Completed
| # | Type | Description |
|---|------|-------------|
| 1 | Fix | **!BADKEY slog** — Converti ~482 appels printf-style en key-value dans ~40 fichiers (commit `b5f67b0`) |
| 2 | Fix | **Logs directory** — `install.sh` crée `/srv/anemone/logs` + messages d'erreur explicites (commit `bb24481`) |
| 3 | Fix | **Docker HTTP security** — Serveur HTTP interne lié au bridge docker0 uniquement, pas exposé au réseau (commit `6058db6`) |
| 4 | Chore | **Version bump** — `0.15.3-beta` → `0.20.0-beta` |
| 5 | Docs | **CHANGELOG.md** — Toutes les nouveautés depuis v0.15.3-beta |
| 6 | Docs | **README.md** — Ajout téléchargement dernière release avant git clone |
| 7 | Release | **v0.20.0-beta** — Tag + GitHub release |

## Bugs trouvés et corrigés
1. **Service crash au démarrage** — `install.sh` ne créait pas `/srv/anemone/logs`, le logger crashait
2. **OnlyOffice ECONNREFUSED** — Le serveur HTTP n'écoutait pas sur le port 8080 car `OnlyOfficeEnabled=false` au boot (activé après via UI). Fix : toujours démarrer HTTP sur docker0 bridge IP quand HTTPS actif
3. **Sécurité HTTP** — Le serveur HTTP écoutait sur `0.0.0.0` (toutes interfaces). Fix : écoute uniquement sur l'IP docker0 (172.17.x.x)

## Tests validés (FR2 - 192.168.83.37)
- [x] Installation propre via install.sh
- [x] Service démarre sans erreur
- [x] OnlyOffice s'ouvre et télécharge le document
- [x] Compilation OK, tests OK, go vet OK

## Files Modified
- `internal/updater/updater.go` — Version 0.15.3-beta → 0.20.0-beta
- `cmd/anemone/main.go` — Docker bridge HTTP server
- `install.sh` — Création répertoire logs
- `internal/logger/rotation.go` — Messages d'erreur explicites
- `CHANGELOG.md` — Release notes v0.20.0-beta
- `README.md` — Option téléchargement release
