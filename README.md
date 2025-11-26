# 🪸 Anemone v2

**Multi-user NAS with P2P encrypted backup synchronization**

---

## ⚠️ BETA WARNING

**This project is currently in BETA and should NOT be used in production environments.**

Anemone is under active development and may contain bugs, security vulnerabilities, or data loss risks. Use at your own risk for testing purposes only.

---

## ⚠️ DISCLAIMER - LIABILITY LIMITATION

**IMPORTANT - READ CAREFULLY**

This software is provided "AS IS", without warranty of any kind, express or implied.

The author and contributors shall not be held liable under any circumstances for:
- ❌ Data loss, corruption or deletion of files
- ❌ Direct or indirect damages resulting from the use of the software
- ❌ Service interruptions or malfunctions
- ❌ Security issues or data breaches
- ❌ Any other damages, even if the author has been advised of their possibility

**You use this software at your own risk.**

It is STRONGLY recommended to:
- ✅ Test in development environment before any production use
- ✅ Maintain external backups of your critical data
- ✅ Not use as the sole backup solution
- ✅ Regularly verify the integrity of your backups

**This software should NOT be used for critical data without external backups.**

For more details, see the AGPL v3.0 license (sections 15 and 16).

---

## 📥 Quick Installation

### Latest Release (Recommended)

```bash
# Clone latest release (v0.9.1-beta)
git clone --branch v0.9.1-beta https://github.com/juste-un-gars/anemone.git
cd anemone

# Run installer (requires sudo)
sudo ./install.sh fr  # For French
# OR
sudo ./install.sh en  # For English

# Access web interface
open https://localhost:8443
```

**What the installer does**:
- ✅ Compiles the binary
- ✅ Creates `/srv/anemone` data directory
- ✅ Installs and configures Samba
- ✅ Sets up firewall rules (if needed)
- ✅ Creates systemd service (auto-start)
- ✅ Generates TLS certificates

### Prerequisites

Before installing, ensure you have:
- **Go 1.21+** - [Installation guide](https://go.dev/doc/install)
- **Samba** (for SMB file sharing)
- **Sudo access** (for system configuration)
- **Optional**: Btrfs filesystem (for quota enforcement)

> ⚠️ **Note**: Anemone works on any filesystem (ext4, XFS, ZFS), but **quota enforcement requires Btrfs**. On other filesystems, quotas are displayed but not enforced.

---

## 🔄 Updating Anemone

### Update to Latest Release

```bash
cd /path/to/anemone

# Fetch latest tags
git fetch --tags --force

# Checkout latest version
git checkout v0.9.1-beta

# Rebuild binaries
go build -o anemone cmd/anemone/main.go
go build -o anemone-dfree cmd/anemone-dfree/main.go

# Restart services
sudo systemctl restart anemone
sudo systemctl reload smbd
```

### Check for Updates

Anemone includes an automatic update notification system:
- **Automatic checks**: Daily verification of GitHub releases
- **Manual check**: Admin interface → ⚡ Updates → "Check now" button
- **Notifications**: Banner displayed when new version available

### Update Notifications

When a new version is available:
1. Log in as admin
2. You'll see a notification banner
3. Click "⚡ Updates" in the navigation menu
4. View release notes and changelog
5. Follow update instructions above

---


## 💖 Support the Project

If you find this project useful and would like to support its development:

**[Support via PayPal](https://paypal.me/justeungars83)**

Your support helps maintain and improve Anemone. Thank you!

---

## 🎯 Overview

Anemone is a self-hosted Network Attached Storage (NAS) solution designed for families and small teams. It provides:

- 🔐 **Multi-user support** with individual encrypted backups
- 🌐 **Peer-to-peer synchronization** with end-to-end encryption (AES-256-GCM)
- ⚡ **Incremental sync** - Only modified files are transferred (manifest-based)
- ⏰ **Automatic scheduler** - Configurable sync intervals (30min/1h/2h/6h/fixed time)
- 🔒 **P2P authentication** - Password protection for sync endpoints
- 📦 **SMB file sharing** (Windows/Mac/Linux compatible)
- 🗑️ **Per-user trash** with configurable retention
- 💾 **Quota management** per user with unlimited quota support (Btrfs only)
- 👥 **Incoming backups management** - View and manage remote peers storing backups on your server
- 🔄 **Web restore interface** - Browse and download encrypted backups from peers
- 🏥 **Disaster recovery** - Server configuration export/import with encryption
- 🌍 **Multilingual** (French & English)
- 🔒 **End-to-end encryption** with user-specific keys and master key protection
- ⚡ **Automatic update notifications** - Daily checks for new releases from GitHub
- 🔄 **Manual update checks** - Force update verification via admin interface

## 🏗️ Architecture

### Stack

- **Backend**: Go (fast, single binary, easy deployment)
- **Database**: SQLite (simple, reliable, no external dependencies)
- **Frontend**: HTML templates + Tailwind CSS
- **File sharing**: Samba (SMB protocol)
- **Backup sync**: Custom incremental sync with AES-256-GCM encryption
- **Scheduler**: Background goroutine for automatic sync

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
│   ├── sync/                    # P2P synchronization (incremental + manifest)
│   ├── syncauth/                # Sync authentication (password protection)
│   ├── syncconfig/              # Sync scheduler configuration
│   ├── scheduler/               # Automatic sync scheduler
│   ├── incoming/                # Incoming backups management
│   ├── crypto/                  # Encryption utilities
│   ├── quota/                   # Quota enforcement
│   ├── trash/                   # Trash management
│   ├── updater/                 # Update notification system
│   ├── i18n/                    # Internationalization (FR/EN)
│   │   └── locales/             # JSON translation files (fr.json, en.json)
│   └── web/                     # HTTP handlers
├── web/
│   ├── static/                  # CSS, JS, images
│   └── templates/               # HTML templates
├── scripts/                     # Installation scripts
└── install.sh                   # Automated installation

/srv/anemone/                    # Data (production)
├── db/anemone.db               # SQLite database
├── shares/                     # User shares (local data)
│   └── username/
│       ├── backup/             # Synced to peers
│       └── data/               # Local only
├── incoming/                   # Backups from remote peers
│   └── source_server_name/     # Organized by source server
│       └── username/           # Then by username
│           └── share_name/     # Then by share name
│               └── *.enc       # Encrypted backup files
├── certs/                      # TLS certificates
└── smb/smb.conf                # Generated Samba config
```

## 🚀 Quick Start

### Prerequisites

- **Btrfs filesystem** (required for quota enforcement) - [Why Btrfs?](#-quotas)
- Go 1.21+ - [Installation guide](https://go.dev/doc/install)
- Samba (for SMB file sharing)
- Sudo access (for system configuration)

> ⚠️ **Important**: Anemone works on any filesystem (ext4, XFS, ZFS), but **quota enforcement requires Btrfs**. On other filesystems, quotas will be displayed but not enforced by the kernel.

### One-Line Installation (Fresh Server)

For a completely new server installation, you can install all dependencies and Anemone in one command:

```bash
# Update system and install dependencies + Anemone (Debian/Ubuntu) - French
sudo apt update -y && \
sudo apt upgrade -y && \
sudo apt-get install -y golang-go samba git && \
git clone https://github.com/juste-un-gars/anemone.git && \
cd anemone && \
sudo ./install.sh fr

# English version
sudo apt update -y && \
sudo apt upgrade -y && \
sudo apt-get install -y golang-go samba git && \
git clone https://github.com/juste-un-gars/anemone.git && \
cd anemone && \
sudo ./install.sh en
```

**For RHEL/Fedora:**
```bash
# Update system and install dependencies + Anemone (RHEL/Fedora) - French
sudo dnf update -y && \
sudo dnf install -y golang samba git && \
git clone https://github.com/juste-un-gars/anemone.git && \
cd anemone && \
sudo ./install.sh fr

# English version
sudo dnf update -y && \
sudo dnf install -y golang samba git && \
git clone https://github.com/juste-un-gars/anemone.git && \
cd anemone && \
sudo ./install.sh en
```

### Standard Installation

```bash
# Install dependencies first
sudo apt update -y                      # Update package lists
sudo apt upgrade -y                     # Upgrade existing packages
sudo apt-get install -y golang-go       # Install Go compiler
sudo apt install -y samba               # Install Samba server

# Clone repository
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Run installer with language (requires sudo)
sudo ./install.sh fr       # For French
# OR
sudo ./install.sh en       # For English
# OR
sudo ./install.sh          # Defaults to French

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

### Password Management

#### User: Change Own Password

1. Go to **Settings** (user menu)
2. Click **Change Password**
3. Enter current password
4. Enter new password (minimum 8 characters)
5. Confirm new password
6. System updates:
   - Database password hash
   - SMB password (automatic sync)
   - Encryption key remains unchanged

#### Admin: Reset User Password

1. Go to **Users** section
2. Find activated user
3. Click **Reset Password**
4. System generates a **password reset link** (valid 24h)
5. Copy link and send to user via email/chat

**User receives link**:
1. User clicks reset link
2. User enters new password (minimum 8 characters)
3. Confirms new password
4. System updates:
   - Database password hash
   - SMB password (automatic sync)
   - Encryption key remains unchanged
5. Redirect to login with success message

**Security notes**:
- Reset tokens are single-use and expire after 24 hours
- Admin does not see or set the new password
- Encryption key is preserved (no data loss)

## 🔐 Security

### End-to-End Encryption

**Backup synchronization uses AES-256-GCM encryption**:
- Backups are **encrypted before leaving the source server**
- Only the destination server with the user's key can decrypt
- Even if the peer server is compromised, backups remain encrypted
- Encryption format: `[nonce (12 bytes)][encrypted data + auth tag]`

### Encryption Keys Architecture

- **Master Key**: Generated at setup, stored in `system_config`
  - Used to encrypt/decrypt user encryption keys
  - Never leaves the server
- **User Encryption Keys**: Unique 32-byte key per user
  - Generated automatically during user activation
  - Shown **once** to user (download recommended)
  - Stored encrypted with master key in `encryption_key_encrypted`
  - Hash stored in `encryption_key_hash` for verification
  - **Without the key, backup data cannot be decrypted**

### P2P Sync Security

- Each user's backups are encrypted with their personal key
- Peers cannot decrypt data from other users (key isolation)
- HTTPS for secure transfer (TLS + application-level encryption)
- No VPN required (network security assumed external)

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

1. **Admin adds peer** (another Anemone instance)
   - Enter peer IP address and port
   - Optionally configure authentication password
   - Test connection to verify accessibility
2. **Incremental synchronization** (manifest-based)
   - Only modified files are transferred (saves bandwidth)
   - Manifest tracks file checksums (SHA-256)
   - Automatic detection of added/modified/deleted files
3. **Encryption before transfer**
   - Data encrypted with user's personal key (AES-256-GCM)
   - Files stored encrypted on peer (`.enc` extension)
   - Peer cannot read content without user's key
4. **Automatic scheduler** (configurable)
   - Intervals: 30min, 1h, 2h, 6h, or fixed daily time
   - Background goroutine checks every minute
   - Configurable via `/admin/sync` interface

### Sync Security

**Password Authentication** (optional but recommended):
- **Server password**: Protect your server's sync endpoints (`/admin/settings`)
  - Remote peers must provide this password to store backups on your server
  - Stored hashed with bcrypt
- **Peer password**: Authenticate with remote servers when syncing
  - Configured when adding/editing a peer
  - Sent in `X-Sync-Password` header
- **Connection testing**: Automatically validates authentication when testing peers
  - Detects invalid passwords before sync attempts
  - Returns specific error codes (401/403)

### Sync Monitoring

- **Dashboard statistics**:
  - Last backup time per user
  - Number of connected peers
  - Sync success/failure indicators
- **Admin interfaces**:
  - `/admin/sync` - Configure automatic synchronization
  - `/admin/peers` - Manage peers (add/edit/delete/test)
  - `/admin/incoming` - View peers storing backups on this server
- **Sync logs** stored in database with detailed history
- **Manual sync** button available per share
- **Force sync** button for admins to trigger immediate full sync

### Incoming Backups Management

View and manage remote peers storing backups on your server:
- **Statistics**: Number of peers, files count, total space used
- **Per-backup details**: Username, share name, file count, size, last modified
- **Delete backups**: Remove incoming backups if needed
- Access via `/admin/incoming` or dashboard card

## 🗑️ Trash System

- Each user has personal trash
- Deleted files retained for **30 days** (configurable)
- Files automatically purged after expiration
- Restore from trash available in dashboard

## 💾 Quotas

**Filesystem Support**:
- ✅ **Btrfs**: Full quota support with kernel enforcement (recommended)
- ⚠️ **Other filesystems (ext4, XFS, ZFS)**: Anemone works but quotas are NOT enforced by the kernel

**⚠️ Important**: For quota enforcement to work, you must use **Btrfs filesystem**.
- On ext4/XFS/ZFS, Anemone will show quota usage but cannot prevent users from exceeding limits
- Automatic filesystem detection

**Admin Management**:
- Set individual quotas per user via web interface (`/admin/users/{id}/quota`)
- Two quota types:
  - **Total Quota**: Overall storage limit for the user
  - **Backup Quota**: Specific limit for backup share
- Default: 100 GB total, 50 GB backup
- **Unlimited quotas**: Set quota to 0 for unlimited storage
- Kernel-level enforcement (quotas enforced by the filesystem)

**Usage Monitoring**:
- Real-time calculation of disk usage
- Automatic scanning of all user shares
- Separate tracking for backup vs data folders

**Visual Alerts** (color-coded progress bars):
- 🟢 **Green** (0-75%): Normal usage
- 🟡 **Yellow** (75-90%): Warning - approaching limit
- 🟠 **Orange** (90-100%): Danger - quota almost reached
- 🔴 **Red** (>100%): Critical - quota exceeded

**User Dashboard**:
- Visual quota display with percentage
- Color-coded alerts
- Detailed breakdown of usage

## 🌍 Internationalization

Anemone features a modern JSON-based internationalization system that makes it easy to add new languages.

**Supported languages**:
- 🇫🇷 French (Français) - 584 keys
- 🇬🇧 English - 584 keys

**Language Selection**:
- Default language chosen during initial setup
- Users can change their language anytime in **Settings** page
- Language preference saved per user in database
- All UI elements fully translated (templates, forms, buttons, messages, JavaScript)

**Architecture**:
- JSON-based translation files (`internal/i18n/locales/*.json`)
- Single binary with embedded translations (`//go:embed`)
- Easy to add new languages (~15 minutes)
- Non-technical translators can contribute
- Complete guide in `internal/i18n/locales/README.md`

**Adding a new language**:
1. Copy `internal/i18n/locales/fr.json` to `[language_code].json`
2. Translate the 584 values
3. Add 5 lines of code to `internal/i18n/i18n.go`
4. Update `GetAvailableLanguages()`
5. Compile and test

See `internal/i18n/locales/README.md` for detailed instructions.

## 📊 Database Schema

See `internal/database/migrations.go` for complete schema.

Main tables:
- `system_config` - System settings (including sync auth password hash)
- `system_info` - System information (version tracking, update notifications)
- `users` - User accounts (with language preference and encryption keys)
- `activation_tokens` - Temporary activation links (24h validity)
- `password_reset_tokens` - Password reset links (24h validity, single-use)
- `shares` - File shares (with sync_enabled flag)
- `trash_items` - Deleted files
- `peers` - Connected Anemone instances (with optional password for auth)
- `sync_log` - Synchronization history (detailed logs with file counts and bytes)
- `sync_config` - Automatic sync scheduler configuration (interval, last_sync, enabled)

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

## 💿 Use an External Drive for Anemone

You can store all Anemone data on an external USB drive by mounting it to `/srv/anemone`.

### 🎯 Why Use an External Drive?

- **Larger storage capacity** - Store more user backups
- **Portability** - Move data between servers easily
- **Hardware isolation** - Separate data from system disk
- **Easy expansion** - Upgrade storage without system changes

### 📝 Step-by-Step Setup

#### 1. Identify Your USB Drive

```bash
lsblk
# or
sudo fdisk -l
```

Look for your USB drive (example: `/dev/sdb1`)

#### 2. Format the Drive (Optional - ⚠️ Erases All Data)

```bash
# Format as ext4 (recommended for Linux)
sudo mkfs.ext4 /dev/sdb1

# Give it a label for easy identification
sudo e2label /dev/sdb1 ANEMONE_DATA
```

**For Btrfs** (recommended if you need quota enforcement):
```bash
sudo mkfs.btrfs -L ANEMONE_DATA /dev/sdb1
```

#### 3. Fresh Installation (No Existing Data)

```bash
# Create mount point
sudo mkdir -p /srv/anemone

# Mount the drive
sudo mount /dev/sdb1 /srv/anemone

# Set correct permissions
sudo chown -R anemone:anemone /srv/anemone
sudo chmod 755 /srv/anemone
```

#### 4. Migration from Existing Installation

```bash
# Stop Anemone service
sudo systemctl stop anemone

# Mount drive temporarily
sudo mkdir -p /mnt/usb
sudo mount /dev/sdb1 /mnt/usb

# Copy all existing data
sudo rsync -av /srv/anemone/ /mnt/usb/

# Unmount and remount on correct path
sudo umount /mnt/usb
sudo mount /dev/sdb1 /srv/anemone

# Set permissions
sudo chown -R anemone:anemone /srv/anemone
sudo chmod 755 /srv/anemone

# Start Anemone
sudo systemctl start anemone
```

#### 5. Configure Automatic Mounting (/etc/fstab)

**⚠️ IMPORTANT**: Use UUID, not `/dev/sdX` (device names can change!)

```bash
# Get the drive's UUID
sudo blkid /dev/sdb1
# Example output: UUID="abc123-def456-..."

# Edit fstab
sudo nano /etc/fstab
```

**Add this line** (replace UUID with your actual value):
```
UUID=abc123-def456-...  /srv/anemone  ext4  defaults,nofail  0  2
```

**For Btrfs:**
```
UUID=abc123-def456-...  /srv/anemone  btrfs  defaults,nofail  0  2
```

**Important flags**:
- `defaults` - Standard mount options
- `nofail` - System boots even if USB drive is disconnected
- `0` - No dump (backup) needed
- `2` - Check filesystem on boot (after root filesystem)

#### 6. Test the Configuration

```bash
# Test fstab configuration
sudo mount -a

# Verify mount
df -h | grep anemone
mount | grep anemone

# Check permissions
ls -la /srv/anemone

# Test Anemone
sudo systemctl restart anemone
sudo systemctl status anemone
```

### 🔧 Configure Systemd (Optional but Recommended)

Edit `/etc/systemd/system/anemone.service` to ensure mount is ready:

```ini
[Unit]
Description=Anemone Backup System
After=network.target
RequiresMountsFor=/srv/anemone

[Service]
Type=simple
User=anemone
ExecStart=/usr/local/bin/anemone
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

Apply changes:
```bash
sudo systemctl daemon-reload
sudo systemctl restart anemone
```

### ⚠️ Important Precautions

1. **Always use UUID in /etc/fstab**
   - ❌ Never use `/dev/sdX` (unstable if you plug other devices)
   - ✅ Always use `UUID=...` (stable identifier)

2. **Use `nofail` option**
   - System will boot even if USB drive is disconnected
   - Without it, boot may fail or hang

3. **USB 3.0+ recommended**
   - USB 2.0 works but will be slow for large backups
   - USB 3.0 or faster recommended for best performance

4. **Backup considerations**
   - External drives can fail
   - Use Anemone's P2P sync to other servers for redundancy
   - Consider having multiple backup locations

5. **Power considerations**
   - Some USB drives need external power
   - Ensure stable power supply to avoid data corruption

### ✅ Verification Checklist

```bash
# 1. Mount is active
mount | grep anemone
# Expected: /dev/sdb1 on /srv/anemone type ext4 (rw,relatime)

# 2. Permissions are correct
ls -la /srv/anemone
# Expected: drwxr-xr-x ... anemone anemone ...

# 3. Anemone service is running
sudo systemctl status anemone
# Expected: Active: active (running)

# 4. Data is on external drive
df -h /srv/anemone
# Check that mounted device is your USB drive

# 5. Files are being created correctly
ls -la /srv/anemone/
# Expected: db/ shares/ certs/ smb/
```

### 🔄 Unmounting Safely

If you need to disconnect the USB drive:

```bash
# 1. Stop Anemone service
sudo systemctl stop anemone

# 2. Ensure no processes are using the mount
sudo lsof +D /srv/anemone
sudo fuser -m /srv/anemone

# 3. Unmount safely
sudo umount /srv/anemone

# 4. Now safe to disconnect USB drive
```

### 🚨 Troubleshooting External Drive

**Problem**: Drive not mounting on boot
```bash
# Check fstab syntax
sudo mount -a

# Check system logs
sudo journalctl -xe | grep mount
```

**Problem**: Permission denied errors
```bash
# Fix ownership
sudo chown -R anemone:anemone /srv/anemone
sudo chmod 755 /srv/anemone
```

**Problem**: Anemone won't start after mounting
```bash
# Check if mount is correct
mount | grep anemone

# Check Anemone logs
sudo journalctl -u anemone -n 50

# Verify data directory structure
ls -la /srv/anemone/
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

## 🗑️ Complete Uninstall

To completely remove Anemone from your system, follow these steps:

### 1. Stop the server

```bash
# If running as systemd service
sudo systemctl stop anemone
sudo systemctl disable anemone

# If running manually
pkill -f anemone
# Or force kill if needed
killall -9 anemone
```

### 2. Remove all data

**⚠️ WARNING**: This will delete ALL user data, shares, and configuration permanently!

```bash
# Remove all Anemone data (database, shares, certificates, SMB config)
sudo rm -rf /srv/anemone

# Explanation:
# - /srv/anemone/db/       → SQLite database (users, shares, sync logs)
# - /srv/anemone/shares/   → All user files (backup + data shares)
# - /srv/anemone/certs/    → TLS certificates
# - /srv/anemone/smb/      → Generated Samba configuration
```

### 3. Remove system users (optional)

Anemone creates system users for each activated user. Remove them if needed:

```bash
# List Anemone users (non-system users typically have UID > 1000)
awk -F: '$3 >= 1000 {print $1}' /etc/passwd

# Remove a specific user (replace 'username' with actual username)
sudo userdel username

# Remove user's home directory (if any)
sudo rm -rf /home/username
```

### 4. Remove SMB users

```bash
# List SMB users
sudo pdbedit -L

# Remove a specific SMB user (replace 'username')
sudo smbpasswd -x username

# Or remove all non-standard SMB users
for user in $(sudo pdbedit -L | cut -d: -f1); do
    if [ "$user" != "root" ] && [ "$user" != "nobody" ]; then
        echo "Removing SMB user: $user"
        sudo smbpasswd -x "$user"
    fi
done
```

### 5. Clean Samba configuration

```bash
# Remove Anemone's SMB configuration
sudo rm -f /etc/samba/smb.conf.anemone

# If you replaced the main smb.conf, restore the original
sudo cp /etc/samba/smb.conf.orig /etc/samba/smb.conf 2>/dev/null || true

# Reload Samba
sudo systemctl reload smb     # Fedora
sudo systemctl reload smbd    # Debian/Ubuntu
```

### 6. Remove sudoers configuration

```bash
# Remove Anemone sudoers rules
sudo rm -f /etc/sudoers.d/anemone-smb
```

### 7. Remove systemd service (if installed)

```bash
# Remove service file
sudo rm -f /etc/systemd/system/anemone.service

# Reload systemd
sudo systemctl daemon-reload
```

### 8. Remove binary and source code

```bash
# Remove installed binary
sudo rm -f /usr/local/bin/anemone

# Remove source code (if you want to delete the git repo)
rm -rf ~/anemone
```

### Complete one-liner (dangerous!)

**⚠️ USE WITH EXTREME CAUTION**: This removes everything in one command:

```bash
sudo systemctl stop anemone 2>/dev/null; \
sudo systemctl disable anemone 2>/dev/null; \
killall -9 anemone 2>/dev/null; \
sudo rm -rf /srv/anemone; \
sudo rm -f /etc/sudoers.d/anemone-smb; \
sudo rm -f /etc/systemd/system/anemone.service; \
sudo systemctl daemon-reload; \
echo "✓ Anemone removed (system users and SMB users NOT removed - see above)"
```

## 📝 Development Status

**Current Status**: 🟡 BETA - All core features complete, pending long-term testing

**Implemented** ✅:
- [x] Setup page & initial configuration
- [x] User authentication (login/logout)
- [x] Multi-user management
- [x] Activation tokens system (24h validity)
- [x] Password management:
  - [x] User: Change own password (Settings page)
  - [x] Admin: Reset user password (24h tokens)
  - [x] Automatic DB + SMB sync
  - [x] Encryption key preservation
- [x] Settings page:
  - [x] Language selector (FR/EN)
  - [x] Password change form
  - [x] Account information display
- [x] Internationalization (i18n):
  - [x] French (100% complete)
  - [x] English (100% complete)
  - [x] Per-user language preference
  - [x] All UI translated (templates, dashboards, forms)
- [x] Trash system:
  - [x] Per-user SMB recycle bin
  - [x] File listing with restore/delete
  - [x] Bulk operations support
  - [x] Dashboard statistics
- [x] Dashboard & Statistics:
  - [x] Storage usage (real-time)
  - [x] Trash count
  - [x] Last sync timestamp
  - [x] User information
  - [x] Quick access cards for all admin features
- [x] Automatic SMB share creation (backup + data per user)
- [x] Samba dynamic configuration & auto-reload
- [x] HTTPS with self-signed certificates
- [x] SELinux configuration (Fedora/RHEL)
- [x] Automated installation script
- [x] Privacy (users only see their own shares)
- [x] Quota management (Btrfs only):
  - [x] Per-user quotas (backup + data)
  - [x] Kernel-level enforcement (Btrfs subvolumes + qgroups)
  - [x] Real-time usage display with alerts
  - [x] Admin quota configuration interface
  - [x] Fallback mode for non-Btrfs filesystems (tracking only, no enforcement)
- [x] **P2P Synchronization** (Sessions 8-11):
  - [x] **Incremental sync** (manifest-based, only changed files)
  - [x] **Automatic scheduler** (30min/1h/2h/6h/fixed time)
  - [x] **Password authentication** (server + peer passwords)
  - [x] Manual sync button per share
  - [x] Force sync for admins
  - [x] AES-256-GCM encryption per file
  - [x] End-to-end encrypted transfer over HTTPS
  - [x] SHA-256 checksums for integrity
- [x] **Peers Management**:
  - [x] CRUD operations (create/read/update/delete)
  - [x] Connection testing with auth validation
  - [x] Edit interface for modifying peer config
  - [x] Password management (add/modify/remove)
  - [x] Enable/disable peers
- [x] **Incoming Backups Management**:
  - [x] View peers storing backups on this server
  - [x] Statistics (peers count, files, space used)
  - [x] Delete incoming backups
  - [x] Dashboard integration

**Recently Added** ✨:
- [x] **Web restore interface** - Browse and download encrypted backups from peers
- [x] **Server config export/import** - Disaster recovery with encrypted config backups
- [x] **Admin bulk restore** - Restore all users' files from backup peers
- [x] **Source server separation** - Support multiple source servers backing up to same peer
- [x] **Automatic update system** - Daily GitHub release checks with admin notifications
- [x] **Update management page** - Dedicated interface with manual check and release notes display

**Next Features** 📅:
- [ ] Per-peer sync frequency (daily/weekly/monthly snapshots)
- [ ] Audit trail and logging system
- [ ] Backup integrity verification tool
- [ ] Rate limiting (anti-bruteforce)
- [ ] Advanced statistics and monitoring
- [ ] Notification system (webhooks, Home Assistant, email)
- [ ] Complete user guide with screenshots
- [ ] Web-based admin restore interface improvements

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
