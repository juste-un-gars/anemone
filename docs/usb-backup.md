# USB Backup

## Overview

USB Backup allows you to back up your Anemone server to external USB drives or mounted storage devices. This provides a portable, offline backup solution that can be stored securely off-site.

```
Anemone Server                USB Drive
┌─────────────────┐           ┌─────────────────┐
│ db/anemone.db   │           │ anemone-backup/ │
│ certs/          │  ──sync─► │   config/       │
│ smb/smb.conf    │ (encrypted)│   shares/       │
│ shares/user/    │           │   manifest.json │
└─────────────────┘           └─────────────────┘
```

- Files are **encrypted** using AES-256-GCM
- **Incremental sync**: only modified files are transferred
- **Two backup types**: Config only (~10 MB) or Config + selected shares

## Requirements

- A USB drive or external storage device mounted on the server
- Mount point in `/mnt/`, `/media/`, or `/run/media/`
- Sufficient space for selected backup content

## Detecting USB Drives

Anemone automatically detects external drives:

1. Go to **USB Backup** (admin)
2. Available drives appear in the list
3. Each drive shows:
   - Device path (e.g., `/dev/sdb1`)
   - Mount point
   - Available space

### Not Seeing Your Drive?

- Ensure the drive is mounted (check **Storage** page)
- Drive must be mounted in `/mnt/`, `/media/`, or `/run/media/`
- On NVMe systems, USB drives appear as `/dev/sda*` (this is normal)

## Creating a USB Backup

1. Go to **USB Backup** (admin)
2. Click **Add Backup**
3. Configure:
   - **Name**: Identifier for this backup (e.g., "Office Backup")
   - **Mount Path**: Where the USB drive is mounted
   - **Backup Path**: Folder name on the drive (default: `anemone-backup`)

### Backup Types

#### Config Only

Backs up essential configuration (~10 MB):
- Database (`db/anemone.db`)
- TLS certificates (`certs/`)
- Samba configuration (`smb/smb.conf`)

**Use case**: Quick backup to any USB drive, disaster recovery of server configuration.

#### Config + Data

Backs up configuration plus selected user shares:
- All configuration files (same as above)
- Selected user shares with all files

**Use case**: Full backup including user data. Requires more space.

### Share Selection

When using "Config + Data":

1. Available shares are listed with their sizes
2. Check the shares you want to include
3. Estimated total size is displayed
4. Ensure your USB drive has sufficient space

## Automatic Scheduling

USB Backup can run automatically when the drive is connected.

### Enabling Auto-Sync

1. Go to **USB Backup**
2. Edit an existing backup
3. Enable **Automatic sync**
4. Choose frequency:

| Mode | Description |
|------|-------------|
| **Interval** | Every 15min, 30min, 1h, 2h, 4h, 8h, 12h, or 24h |
| **Daily** | Every day at a specific time (e.g., 02:00) |
| **Weekly** | Every week on a specific day and time |
| **Monthly** | Every month on a specific day (1-28) and time |

### How It Works

- Scheduler checks every minute for pending backups
- Backup only runs if the USB drive is mounted
- Last sync time is tracked to respect intervals
- Logs are stored in the database

## Manual Sync

To run a backup immediately:

1. Go to **USB Backup**
2. Find your backup in the list
3. Click **Sync Now**
4. Progress is displayed in real-time

### Sync Status

During sync, you can see:
- Current operation
- Files processed
- Bytes transferred
- Any errors

## Safe Ejection

Always eject USB drives safely to prevent data corruption:

1. Go to **USB Backup**
2. Click **Eject** next to the backup
3. Wait for confirmation
4. Remove the drive physically

Alternatively, use the **Storage** page:
1. Find the mounted drive
2. Click **Unmount** or **Eject**

## Backup Structure

On the USB drive:

```
anemone-backup/
├── config/
│   ├── db/
│   │   └── anemone.db.enc      # Encrypted database
│   ├── certs/
│   │   ├── server.crt.enc
│   │   └── server.key.enc
│   └── smb/
│       └── smb.conf.enc
├── shares/                      # If Config + Data
│   └── username/
│       └── data/
│           └── file.txt.enc
└── manifest.json                # Backup manifest
```

### Encryption

- All files are encrypted with AES-256-GCM
- Encryption key derived from server's master key
- Files can only be decrypted by the same Anemone installation
- Format: `[nonce 12 bytes][encrypted data + auth tag]`

## Restore from USB Backup

To restore from a USB backup:

1. Mount the USB drive
2. Use the **Import Existing Installation** option in Setup Wizard
3. Point to the backup location
4. Anemone will decrypt and restore configuration

### Manual Restore

For advanced users:

```bash
# Stop Anemone
sudo systemctl stop anemone

# Decrypt and restore database
# (requires encryption key from original installation)

# Restart
sudo systemctl start anemone
```

## Troubleshooting

### "No USB drives detected"

- Check if drive is mounted: `lsblk` or `mount`
- Drive must be in `/mnt/`, `/media/`, or `/run/media/`
- Format the drive first if needed (see **Storage** page)

### "Permission denied" during sync

- Check mount permissions
- Ensure mount has write access
- For FAT32/exFAT, remount with `umask=000` option

### Sync fails with "disk full"

- Choose fewer shares to backup
- Use "Config only" mode
- Use a larger USB drive

### Scheduled sync not running

- Verify the USB drive is mounted
- Check if automatic sync is enabled
- View logs: `sudo journalctl -u anemone | grep usb`

### Slow sync performance

- USB 2.0 drives are slower (~30 MB/s max)
- Large files take longer to encrypt
- Consider using USB 3.0 drives

## Best Practices

1. **Regular rotation**: Use multiple USB drives and rotate them
2. **Off-site storage**: Keep at least one backup off-site
3. **Test restores**: Periodically verify backups work
4. **Label drives**: Mark drives with backup date and content
5. **Encrypt drives**: Consider full-disk encryption for extra security

## See Also

- [Storage Management](storage.md) - Formatting and mounting drives
- [P2P Sync](p2p-sync.md) - Network-based backup alternative
- [Security](security.md) - Encryption details
- [User Guide](user-guide.md) - General usage
