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

## See Also

- [P2P Sync](p2p-sync.md) - Backups between servers
- [Security](security.md) - Encryption keys
- [Troubleshooting](troubleshooting.md) - SMB share issues
