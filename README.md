# 🪸 Anemone v2

**Multi-user NAS with P2P encrypted backup synchronization**

## 🎯 Overview

Anemone is a self-hosted Network Attached Storage (NAS) solution designed for families and small teams. It provides:

- 🔐 **Multi-user support** with individual encrypted backups
- 🌐 **Peer-to-peer synchronization** of encrypted data
- 📦 **SMB file sharing** (Windows/Mac/Linux compatible)
- 🗑️ **Per-user trash** with configurable retention
- 💾 **Quota management** per user
- 🌍 **Multilingual** (French & English)
- 🔒 **End-to-end encryption** with user-specific keys

## 🏗️ Architecture

### Stack

- **Backend**: Go (fast, single binary, easy deployment)
- **Database**: SQLite (simple, reliable, no external dependencies)
- **Frontend**: HTML templates + HTMX + Tailwind CSS
- **File sharing**: Samba (SMB protocol)
- **Backup sync**: rclone with encryption

### Project Structure

```
anemone/
├── cmd/anemone/main.go          # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── database/                # SQLite + migrations
│   ├── users/                   # User management & auth
│   ├── shares/                  # SMB share management
│   ├── sync/                    # P2P synchronization
│   ├── crypto/                  # Encryption utilities
│   ├── quota/                   # Quota enforcement
│   ├── trash/                   # Trash management
│   └── web/                     # HTTP handlers
├── web/
│   ├── static/                  # CSS, JS, images
│   └── templates/               # HTML templates
├── data/                        # Runtime data (gitignored)
│   ├── db/anemone.db           # SQLite database
│   ├── shares/                 # User shares
│   └── config/                 # Generated configs
└── docker-compose.yml
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose - [Installation guide](https://docs.docker.com/engine/install/)
- OR: Go 1.21+ (for local development)

### With Docker (Recommended)

```bash
# Clone repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Start services
docker compose up -d

# Access web interface
open http://localhost:8080
```

### Local Development

```bash
# Install Go 1.21+
# https://go.dev/doc/install

# Clone repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Install dependencies
go mod download

# Run
go run cmd/anemone/main.go
```

## 📋 Initial Setup

1. **Access web interface** at `http://localhost:8080`
2. **Choose language** (French or English)
3. **Set NAS name** and timezone
4. **Create first admin user**
   - Username
   - Password
   - Email (optional)
5. System generates encryption key automatically
6. **Done!** Redirect to admin dashboard

## 👥 User Management

### Adding a User (Admin)

1. Go to **Users** section in admin dashboard
2. Click **Add User**
3. Enter username and email
4. System generates a **temporary activation link** (valid 24h)
5. Send link to user via email/chat

### User Activation

1. User clicks activation link
2. User chooses password
3. System generates **encryption key** (32 bytes random)
4. ⚠️ **CRITICAL PAGE**:
   - Key is displayed ONE TIME only
   - User must save it (copy/print/download)
   - Checkboxes: "I saved my key" + "I understand I cannot recover without it"
   - User must re-type key to confirm
5. Account activated → Redirect to dashboard

## 🔐 Security

### Encryption Keys

- Each user has a **unique encryption key** (32 bytes)
- Key is generated automatically and shown **once** during activation
- Key is stored encrypted in database (using system master key)
- Hash stored for verification without exposing the key
- **Without the key, backup data cannot be decrypted**

### P2P Sync Security

- Each user's backups are encrypted with their personal key
- Peers cannot decrypt data from other users
- No VPN required (assume firewall/network security handled externally)
- HTTPS recommended for peer connections

## 📂 File Shares

### Default Structure

Each user gets:

```
/shares/
  └── username/
      └── backup/     # Auto-synced to peers (encrypted)
```

Optional additional shares can be created (local only, no sync).

### SMB Access

```
Windows: \\nas.local\username-backup
Mac:     smb://nas.local/username-backup
Linux:   smb://nas.local/username-backup
```

## 🔄 P2P Synchronization

### How it works

1. Admin adds **peer** (another Anemone instance)
2. Enter peer IP address
3. Each user's `backup/` folder syncs automatically
4. Data is encrypted **before** leaving source NAS
5. Peer stores encrypted blobs (cannot read content)

### Sync Monitoring

- Dashboard shows last sync time per user
- Sync logs stored in database
- Manual sync button available

## 🗑️ Trash System

- Each user has personal trash
- Deleted files retained for **30 days** (configurable)
- Files automatically purged after expiration
- Restore from trash available in dashboard

## 💾 Quotas

- Admin sets per-user quotas (total + backup)
- System monitors usage
- Alerts when approaching limit
- Blocks writes when quota exceeded

## 🌍 Internationalization

Supported languages:
- 🇫🇷 French
- 🇬🇧 English

Language selected during initial setup.

## 📊 Database Schema

See `internal/database/migrations.go` for complete schema.

Main tables:
- `system_config` - System settings
- `users` - User accounts
- `activation_tokens` - Temporary activation links
- `shares` - File shares
- `trash_items` - Deleted files
- `peers` - Connected Anemone instances
- `sync_log` - Synchronization history

## 🔧 Configuration

Environment variables:

```bash
ANEMONE_DATA_DIR=/app/data  # Data directory
PORT=8080                    # HTTP port
LANGUAGE=fr                  # Default language (fr/en)
```

## 🐛 Troubleshooting

### Can't access web interface

```bash
# Check if server is running
docker compose ps

# Check logs
docker compose logs anemone
```

### Database issues

```bash
# Reset database (WARNING: deletes all data)
rm data/db/anemone.db
docker compose restart anemone
```

## 📝 Development Status

**Current**: ✅ Base structure created

**Next**:
- [ ] Setup page implementation
- [ ] User authentication
- [ ] Activation tokens system
- [ ] Samba dynamic configuration
- [ ] rclone multi-user sync
- [ ] Dashboard pages

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📄 License

GNU Affero General Public License v3.0 (AGPLv3)

Copyright (C) 2025 juste-un-gars

See [LICENSE](LICENSE) for full license text.

## 📚 Old Version

The previous Python/Bash version is archived in `_old/` directory for reference.

## 🆘 Support

- **Issues**: https://github.com/juste-un-gars/anemone/issues
- **Discussions**: https://github.com/juste-un-gars/anemone/discussions
