# Anemone Documentation

Welcome to the Anemone documentation, a multi-user NAS with P2P encrypted synchronization.

## Guides

### Getting Started
- [Installation](installation.md) - Installation and initial setup
- [Storage Setup](storage-setup.md) - RAID, ZFS, Btrfs configuration

### Usage
- [User Guide](user-guide.md) - Users, shares, quotas, trash
- [P2P Sync](p2p-sync.md) - Peer configuration and restore

### Administration
- [Security](security.md) - Encryption, keys, authentication
- [Advanced Configuration](advanced.md) - Environment variables, external drives
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

### Development
- [Translation (i18n)](i18n.md) - Add a new language
- [API](API.md) - REST API endpoints
- [User Manifests](user-manifests.md) - Manifest system for sync

### Maintenance
- [Uninstall](uninstall.md) - Complete system removal

## Architecture

```
/srv/anemone/              # Data (production)
├── db/anemone.db          # SQLite database
├── shares/                # User shares
│   └── username/
│       ├── backup/        # Synced with peers
│       └── data/          # Local only
├── incoming/              # Backups received from peers
├── certs/                 # TLS certificates
└── smb/smb.conf           # Samba configuration
```

## Useful Links

- [Main README](../README.md)
- [GitHub Issues](https://github.com/juste-un-gars/anemone/issues)
- [Discussions](https://github.com/juste-un-gars/anemone/discussions)
