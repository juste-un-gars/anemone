# Quick Reference - Anemone

Quick reference for common commands, URLs, and project info.

---

## URLs

| Resource | URL |
|----------|-----|
| GitHub Repo | https://github.com/juste-un-gars/anemone |
| GitHub Releases | https://github.com/juste-un-gars/anemone/releases |
| GitHub Issues | https://github.com/juste-un-gars/anemone/issues |
| PayPal Support | https://paypal.me/justeungars83 |

---

## Ports

| Service | Port | Protocol |
|---------|------|----------|
| Web UI (HTTPS) | 8443 | HTTPS |
| Web UI (HTTP) | 8080 | HTTP |
| Samba (SMB) | 445 | TCP |
| P2P Sync | 8443 | HTTPS |

---

## Common Commands

### Build
```bash
# Main binary
go build -o anemone cmd/anemone/main.go

# Dfree helper (Samba quota display)
go build -o anemone-dfree cmd/anemone-dfree/main.go

# All binaries
go build -o anemone cmd/anemone/main.go && go build -o anemone-dfree cmd/anemone-dfree/main.go
```

### Test
```bash
# All tests
go test ./...

# Verbose
go test -v ./...

# Specific package
go test ./internal/sync/...
go test ./internal/usermanifest/...

# With coverage
go test -cover ./...
```

### Run (Development)
```bash
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

### Service (Production)
```bash
# Restart Anemone
sudo systemctl restart anemone

# Reload Samba
sudo systemctl reload smbd

# View logs
sudo journalctl -u anemone -f

# Status
sudo systemctl status anemone
```

### Git & Release
```bash
# Tag new version
git tag -a v0.9.X-beta -m "Description"
git push origin v0.9.X-beta

# Create GitHub release
gh release create v0.9.X-beta --title "Anemone v0.9.X" --notes "Release notes"
```

---

## Project Paths

### Source Code
```
~/anemone/                    # Git repository
├── cmd/anemone/main.go       # Entry point
├── internal/                 # Go packages
├── web/templates/            # HTML templates
└── web/static/               # CSS, JS, images
```

### Production Data
```
/srv/anemone/                 # Data directory
├── db/anemone.db            # SQLite database
├── shares/                  # User files
├── incoming/                # Remote backups
├── certs/                   # TLS certificates
└── smb/smb.conf             # Samba config
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ANEMONE_DATA_DIR` | `./data` | Data directory path |
| `PORT` | `8080` | HTTP port |
| `HTTPS_PORT` | `8443` | HTTPS port |
| `ENABLE_HTTP` | `false` | Enable HTTP |
| `ENABLE_HTTPS` | `true` | Enable HTTPS |
| `LANGUAGE` | `fr` | Default language (fr/en) |

---

## Database

```bash
# Open database
sqlite3 /srv/anemone/db/anemone.db

# Common queries
.tables                           # List tables
SELECT * FROM users;              # List users
SELECT * FROM peers;              # List peers
SELECT * FROM sync_log ORDER BY id DESC LIMIT 10;  # Recent syncs
```

### Main Tables
- `system_config` - System settings
- `system_info` - Version, updates
- `users` - User accounts
- `shares` - SMB shares
- `peers` - P2P peers
- `sync_log` - Sync history
- `sync_config` - Scheduler config

---

## Version

Update version in: `internal/updater/updater.go`

```go
const CurrentVersion = "0.9.16-beta"
```

---

## Translations

Files: `internal/i18n/locales/fr.json` and `en.json`

Add new keys to both files, then use in templates:
```html
{{.T.key_name}}
```

---

## Useful File Locations

| Purpose | Path |
|---------|------|
| Main entry | `cmd/anemone/main.go` |
| HTTP handlers | `internal/web/handlers.go` |
| Database migrations | `internal/database/migrations.go` |
| Translations | `internal/i18n/locales/*.json` |
| HTML templates | `web/templates/*.html` |
| Install script | `install.sh` |
| Changelog | `CHANGELOG.md` |

---

*Last updated: 2025-01-18*
