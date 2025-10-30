# 🪸 Anemone v2

**Multi-user NAS with P2P encrypted backup synchronization**

---

## ⚠️ BETA WARNING

**This project is currently in BETA and should NOT be used in production environments.**

Anemone is under active development and may contain bugs, security vulnerabilities, or data loss risks. Use at your own risk for testing purposes only.

---

## 💖 Support the Project

If you find this project useful and would like to support its development:

**[Support via PayPal](https://paypal.me/justeungars83)**

Your support helps maintain and improve Anemone. Thank you!

---

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
~/anemone/                       # Code (cloned repo)
├── cmd/anemone/main.go          # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── database/                # SQLite + migrations
│   ├── users/                   # User management & auth
│   ├── shares/                  # SMB share management
│   ├── peers/                   # P2P peers management
│   ├── smb/                     # Samba configuration
│   ├── sync/                    # P2P synchronization
│   ├── crypto/                  # Encryption utilities
│   ├── quota/                   # Quota enforcement
│   ├── trash/                   # Trash management
│   └── web/                     # HTTP handlers
├── web/
│   ├── static/                  # CSS, JS, images
│   └── templates/               # HTML templates
├── scripts/                     # Installation scripts
└── install.sh                   # Automated installation

/srv/anemone/                    # Data (production)
├── db/anemone.db               # SQLite database
├── shares/                     # User shares
│   └── username/
│       ├── backup/             # Synced to peers
│       └── data/               # Local only
├── certs/                      # TLS certificates
└── smb/smb.conf                # Generated Samba config
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+ - [Installation guide](https://go.dev/doc/install)
- Samba (for SMB file sharing)
- Sudo access (for system configuration)

### Automated Installation (Recommended)

```bash
# Clone repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Run installer (requires sudo)
sudo ./install.sh

# The installer will:
# - Compile the binary
# - Create /srv/anemone data directory
# - Install and configure Samba
# - Configure SELinux (Fedora/RHEL)
# - Set up firewall rules
# - Create systemd service (auto-start)
# - Generate TLS certificates

# Access web interface
open https://localhost:8443
```

### Manual Installation

```bash
# Clone repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Build
CGO_ENABLED=1 go build -o anemone ./cmd/anemone

# Create data directory
sudo mkdir -p /srv/anemone
sudo chown $USER:$USER /srv/anemone

# Run
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

### Docker (Alternative)

```bash
# Clone repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Start services
docker compose up -d

# Access web interface
open http://localhost:8080
```

## 📋 Initial Setup

1. **Access web interface** at `https://localhost:8443`
   - Accept self-signed certificate warning (normal for local use)
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
ANEMONE_DATA_DIR=/srv/anemone  # Data directory (default: ./data)
PORT=8080                       # HTTP port (default: 8080)
HTTPS_PORT=8443                 # HTTPS port (default: 8443)
ENABLE_HTTP=false               # Enable HTTP (default: false)
ENABLE_HTTPS=true               # Enable HTTPS (default: true)
LANGUAGE=fr                     # Default language (fr/en)
TLS_CERT_PATH=                  # Custom TLS certificate path
TLS_KEY_PATH=                   # Custom TLS key path
```

## 🐛 Troubleshooting

### Can't access web interface

```bash
# Check if server is running
systemctl status anemone

# Check logs
journalctl -u anemone -f

# Or if running manually:
ps aux | grep anemone
```

### SMB shares not accessible

```bash
# Check Samba service
sudo systemctl status smb    # Fedora
sudo systemctl status smbd   # Debian/Ubuntu

# Check Samba configuration
sudo testparm -s

# Check SELinux (Fedora/RHEL only)
ls -laZ /srv/anemone/shares/
sudo ausearch -m avc -ts recent | grep samba

# Verify SMB users
sudo pdbedit -L
```

### Database issues

```bash
# Reset database (WARNING: deletes all data)
sudo rm /srv/anemone/db/anemone.db
systemctl restart anemone
```

## 📝 Development Status

**Current Status**: 🟡 BETA (Core features complete)

**Implemented** ✅:
- [x] Setup page & initial configuration
- [x] User authentication (login/logout)
- [x] Multi-user management
- [x] Activation tokens system (24h validity)
- [x] Automatic SMB share creation (backup + data per user)
- [x] Samba dynamic configuration & auto-reload
- [x] P2P peers management (CRUD, connection testing)
- [x] HTTPS with self-signed certificates
- [x] SELinux configuration (Fedora/RHEL)
- [x] Automated installation script
- [x] Privacy (users only see their own shares)

**In Progress** 🔨:
- [ ] P2P synchronization (rclone with encryption)
- [ ] User quotas & monitoring
- [ ] Trash system (30 days retention)
- [ ] System dashboard & statistics

**Planned** 📅:
- [ ] Settings page (workgroup, network config)
- [ ] Conflict resolution for sync
- [ ] API endpoints for external integrations
- [ ] Docker official image

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
