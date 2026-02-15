# Installation

## Requirements

- **System**: Linux (Debian/Ubuntu, Fedora/RHEL)
- **Access**: sudo
- **Network**: Internet connection (for downloading dependencies)

The installer automatically handles: Go, GCC, Git, Samba, storage tools.

## Standard Installation

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
sudo ./install.sh
```

The installer will:
1. Install dependencies
2. Compile the binary
3. Create data directory
4. Configure sudoers and firewall
5. Create systemd service
6. Start Anemone in setup mode

Then open `https://localhost:8443` for the setup wizard.

## One-liner Installation

```bash
# Debian/Ubuntu
sudo apt update -y && sudo apt upgrade -y && \
git clone https://github.com/juste-un-gars/anemone.git && \
cd anemone && sudo ./install.sh

# Fedora/RHEL
sudo dnf update -y && \
git clone https://github.com/juste-un-gars/anemone.git && \
cd anemone && sudo ./install.sh
```

## Installer Options

```bash
# Custom data directory
sudo ./install.sh --data-dir=/data/anemone

# Specific service user
sudo ./install.sh --user=anemone

# Help
sudo ./install.sh --help
```

## Manual Installation

```bash
# Clone
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Build
CGO_ENABLED=1 go build -o anemone cmd/anemone/main.go

# Create data directory
sudo mkdir -p /srv/anemone
sudo chown $USER:$USER /srv/anemone

# Run
ANEMONE_DATA_DIR=/srv/anemone ./anemone
```

## Setup Wizard

After installation, the web interface guides you through:

1. **Accept certificate** - Normal warning (self-signed)
2. **Installation mode**
   - New installation
   - Restore from backup
   - Import existing pool
3. **Storage configuration**
   - Default path (`/srv/anemone`)
   - Existing ZFS pool
   - New ZFS pool
   - Custom path
4. **Backup storage** - Same location or separate disk
5. **Admin account** - Username, password, email, language
6. **Done** - Save your encryption key and sync password

## Specific Version

```bash
# See versions: https://github.com/juste-un-gars/anemone/releases

# Install specific version
git clone --branch v0.9.1-beta https://github.com/juste-un-gars/anemone.git
cd anemone
sudo ./install.sh
```

## Update

### To Latest Version

```bash
cd /path/to/anemone
git pull origin main
go build -o anemone cmd/anemone/main.go
go build -o anemone-dfree cmd/anemone-dfree/main.go
sudo systemctl restart anemone
sudo systemctl reload smbd
```

### To Specific Version

```bash
cd /path/to/anemone
git fetch --tags --force
git checkout v0.9.2-beta
go build -o anemone cmd/anemone/main.go
go build -o anemone-dfree cmd/anemone-dfree/main.go
sudo systemctl restart anemone
```

### Update Notifications

Anemone automatically checks for new versions (daily).

- Banner displayed when update is available
- Manual check: Admin → Updates → "Check now"

## Network Ports

| Port | Protocol | Usage |
|------|----------|-------|
| 8443 | HTTPS | Web interface (default) |
| 8080 | HTTP | Web interface (disabled by default) |
| 445 | SMB | File shares |

## See Also

- [Storage Setup](storage-setup.md) - RAID, ZFS, Btrfs
- [Advanced Configuration](advanced.md) - Environment variables
- [Troubleshooting](troubleshooting.md) - Installation issues
