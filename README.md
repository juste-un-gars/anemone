# Anemone

**Multi-user NAS with P2P encrypted backup synchronization**

---

> **BETA** - This project is under active development. Use in production at your own risk.
> See [DISCLAIMER](#disclaimer) for liability limitations.

---

## Features

- **Multi-user** with individual SMB shares
- **P2P synchronization** with end-to-end encryption (AES-256-GCM)
- **USB backup** to external drives with scheduling
- **Storage management** - Format, mount, eject disks from web UI
- **Incremental sync** - Only modified files are transferred
- **Quotas** per user (Btrfs recommended)
- **Trash** with configurable retention
- **Web interface** for administration
- **Bilingual** French / English

## Quick Install

```bash
sudo apt update
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
sudo ./install.sh
```

Then open `https://localhost:8443` to complete the setup.

**Requirements**: Linux (Debian/Ubuntu or Fedora/RHEL), sudo access, internet connection.

## Documentation

| Guide | Description |
|-------|-------------|
| [Installation](docs/installation.md) | Full installation and options |
| [Storage Setup](docs/storage-setup.md) | RAID, ZFS, Btrfs |
| [User Guide](docs/user-guide.md) | Users, shares, quotas |
| [P2P Sync](docs/p2p-sync.md) | Peers, scheduler, restore |
| [USB Backup](docs/usb-backup.md) | Backup to USB drives |
| [Security](docs/security.md) | Encryption, keys, architecture |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |
| [Advanced Configuration](docs/advanced.md) | Environment variables, external drives |
| [Translation](docs/i18n.md) | Add a new language |
| [Uninstall](docs/uninstall.md) | Complete removal |

## Update

```bash
cd /path/to/anemone
git pull
go build -o anemone cmd/anemone/main.go
sudo systemctl restart anemone
```

## Support

- **Issues**: https://github.com/juste-un-gars/anemone/issues
- **Discussions**: https://github.com/juste-un-gars/anemone/discussions
- **Support the project**: [PayPal](https://paypal.me/justeungars83)

## License

GNU Affero General Public License v3.0 (AGPLv3)

Copyright (C) 2025 juste-un-gars

---

## Disclaimer

This software is provided "AS IS", without warranty of any kind.

The author shall not be held liable for:
- Data loss or corruption
- Direct or indirect damages
- Service interruptions
- Security issues

**Recommendations**:
- Test in development environment before production
- Maintain external backups
- Do not use as sole backup solution

See AGPL v3.0 license (sections 15 and 16) for details.
