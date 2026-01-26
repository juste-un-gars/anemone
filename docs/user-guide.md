# User Guide

## User Management

### Adding a User (Admin)

1. Go to **Users** from admin dashboard
2. Click **Add User**
3. Enter username and email
4. System generates an **activation link** (valid 24h)
5. Send the link to the user

### Account Activation (User)

1. Click the activation link
2. Choose a password (minimum 8 characters)
3. System generates an **encryption key**
4. **CRITICAL PAGE**:
   - Key is displayed **ONCE ONLY**
   - Save the key (copy/download)
   - Check the confirmation boxes
5. Account activated → redirect to dashboard

### Change Password (User)

1. Go to **Settings**
2. Click **Change Password**
3. Enter old then new password
4. SMB password is synced automatically
5. Encryption key remains unchanged

### Reset Password (Admin)

1. Go to **Users**
2. Find the activated user
3. Click **Reset Password**
4. Copy the generated link (valid 24h)
5. Send it to the user

User clicks the link and chooses a new password.

## File Shares

### Default Structure

Each user gets:

```
/shares/username/
├── backup/     # Synced to peers (encrypted)
└── data/       # Local only (no sync)
```

### SMB Access

| System | Address |
|--------|---------|
| Windows | `\\nas.local\username-backup` |
| Mac | `smb://nas.local/username-backup` |
| Linux | `smb://nas.local/username-backup` |

Use your Anemone username and password.

## Quotas

### How It Works

- **Total quota**: Overall limit for the user
- **Backup quota**: Limit for backup share
- **Data quota**: = Total - Backup

Defaults: 100 GB total, 50 GB backup.

### Filesystem Support

| Filesystem | Quotas |
|------------|--------|
| Btrfs | Enforced by kernel |
| ext4, XFS, ZFS | Displayed but not enforced |

### Visual Alerts

| Color | Usage |
|-------|-------|
| Green | 0-75% |
| Yellow | 75-90% |
| Orange | 90-100% |
| Red | >100% |

### Modify Quotas (Admin)

1. Go to **Users**
2. Click on a user
3. **Edit Quota**
4. Enter new values (0 = unlimited)

## Trash

### How It Works

- Each user has their own trash
- Deleted files kept for **30 days** (configurable)
- Automatic purge after expiration

### Restore a File

1. Go to **Trash** from dashboard
2. Select file(s)
3. Click **Restore**

### Empty Trash

1. Go to **Trash**
2. Select files or "Select All"
3. Click **Delete Permanently**

## User Settings

### Change Language

1. Go to **Settings**
2. Select language (Français / English)
3. Interface updates immediately

### Account Information

Visible in **Settings**:
- Username
- Email
- Creation date
- Last login
- Quota usage

## Dashboard

### Standard User

- Storage usage
- Trash statistics
- Last sync
- Quick access to shares

### Administrator

Additional access to:
- User management
- Peer management
- Sync configuration
- System settings
- Updates

## USB Backup

Back up your server to external USB drives.

### Creating a Backup

1. Connect a USB drive (or use **Storage** to format one)
2. Go to **USB Backup** (admin)
3. Click **Add Backup**
4. Configure name and paths

### Backup Types

| Type | Content | Size |
|------|---------|------|
| **Config only** | Database, certificates, Samba config | ~10 MB |
| **Config + Data** | Config + selected user shares | Varies |

### Automatic Scheduling

1. Edit a USB backup
2. Enable **Automatic sync**
3. Choose frequency: interval, daily, weekly, or monthly

### Manual Sync

Click **Sync Now** to run immediately.

See [USB Backup Guide](usb-backup.md) for complete documentation.

## Storage Management

Manage disks directly from the web interface.

### Formatting a Disk

1. Go to **Storage** (admin)
2. Find unformatted or unmounted disk
3. Click **Format**
4. Choose filesystem:
   - **ext4** - Linux (best performance)
   - **XFS** - Linux (large files)
   - **exFAT** - Windows/Mac/Linux compatible
   - **FAT32** - Universal (4 GB file limit)

### Mounting a Disk

1. Go to **Storage**
2. Click **Mount** on an unmounted disk
3. Choose mount path (e.g., `/mnt/backup-drive`)
4. Options:
   - **Shared access** - All users can read/write
   - **Persistent mount** - Survives reboots (fstab)

### Ejecting Safely

Always eject before disconnecting:

1. Go to **Storage** or **USB Backup**
2. Click **Eject** or **Unmount**
3. Wait for confirmation
4. Remove the drive

## See Also

- [USB Backup](usb-backup.md) - Complete USB backup guide
- [P2P Sync](p2p-sync.md) - Backups between servers
- [Security](security.md) - Encryption keys
- [Troubleshooting](troubleshooting.md) - SMB share issues
