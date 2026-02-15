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

## Web File Browser

Browse and manage files directly from the web interface.

### Browsing Files

1. Go to **Files** from the dashboard
2. Navigate through your shares and directories
3. Click on a file to download it

### Uploading Files

1. Navigate to the target directory
2. Click **Upload** and select files
3. Files are uploaded to the current directory

### Creating Directories

1. Navigate to the parent directory
2. Click **New Folder**
3. Enter the directory name

### Renaming and Deleting

- **Rename**: Click the rename icon next to a file or directory
- **Delete**: Click the delete icon (file is moved to trash)

## OnlyOffice Document Editing

Edit Office documents directly in the browser using OnlyOffice.

### Supported Formats

- **Editable**: DOCX, XLSX, PPTX (and other formats that survive roundtrip editing)
- **Viewable**: PDF, images (PNG, JPG, etc.)

### Editing a Document

1. Go to **Files** and navigate to your document
2. Click **Edit** on a supported Office file
3. The document opens in the OnlyOffice editor
4. Changes are saved back to your share

### Viewing PDFs and Images

1. Go to **Files**
2. Click **View** on a PDF or image file
3. The file opens in the browser

### OnlyOffice Setup (Admin)

OnlyOffice is configured from the web UI:

1. Go to **OnlyOffice** (admin)
2. Click **Enable OnlyOffice**
3. Anemone automatically pulls and configures the Docker container
4. JWT authentication is set up automatically

## Cloud Backup (Rclone)

Back up your server to remote storage using rclone. Multiple provider types are supported.

### Supported Providers

| Provider | Examples |
|----------|----------|
| **SFTP** | Any SSH server |
| **S3** | AWS S3, Backblaze B2, Wasabi, MinIO |
| **WebDAV** | Nextcloud, ownCloud, SharePoint |
| **Named Remote** | Any rclone-configured remote (Google Drive, Dropbox, pCloud...) |

### Adding a Backup Destination

1. Go to **Cloud Backup** (admin)
2. Click **Add Destination**
3. Select the provider type
4. Configure provider-specific settings (host, credentials, bucket, etc.)
5. Optionally enable **Encryption** (rclone crypt, data encrypted before upload)
6. Click **Add Destination**

### SSH Key Authentication (SFTP)

1. In Cloud Backup, click **Generate SSH Key**
2. Copy the public key displayed
3. Add it to the remote server's `~/.ssh/authorized_keys`
4. Test connection before syncing

### Automatic Scheduling

1. Edit a backup destination
2. Enable **Automatic sync**
3. Choose frequency: interval, daily, weekly, or monthly

See [Cloud Backup Guide](rclone-backup.md) for complete documentation.

## WireGuard VPN

Connect to remote peers through a secure VPN tunnel.

### Importing Configuration

1. Go to **WireGuard** (admin)
2. Paste your WireGuard `.conf` file content
3. Click **Import**

### Connection Control

- **Connect**: Establish VPN tunnel
- **Disconnect**: Close VPN tunnel
- **Auto-start**: Automatically connect when Anemone starts

### Status Display

The dashboard shows:
- Connection status (connected/disconnected)
- Last handshake time
- Data transferred (sent/received)

## System Logs

View and manage application logs.

### Accessing Logs

1. Go to **System Logs** (admin)
2. View recent log entries
3. Download log files for analysis

### Log Levels

| Level | Description |
|-------|-------------|
| DEBUG | Verbose debugging information |
| INFO | General operational messages |
| WARN | Warning conditions (default) |
| ERROR | Error conditions |

### Changing Log Level

1. Go to **System Logs**
2. Select desired level
3. Level persists across restarts

Override with environment variable: `ANEMONE_LOG_LEVEL=debug`

## See Also

- [Cloud Backup](rclone-backup.md) - Complete rclone backup guide
- [USB Backup](usb-backup.md) - Complete USB backup guide
- [P2P Sync](p2p-sync.md) - Backups between servers
- [Security](security.md) - Encryption keys
- [Troubleshooting](troubleshooting.md) - SMB share issues
