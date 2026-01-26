# P2P Synchronization

## Overview

Anemone synchronizes backups between multiple servers with encryption.

```
Server A                     Server B
┌─────────────┐              ┌─────────────┐
│ shares/     │   ──sync──►  │ incoming/   │
│   user/     │  (encrypted) │   serverA/  │
│     backup/ │              │     user/   │
└─────────────┘              └─────────────┘
```

- Files are **encrypted before transfer** (AES-256-GCM)
- Only the user with their key can decrypt
- **Incremental** sync: only modified files are transferred

## Adding a Peer

1. Go to **Peers** (admin)
2. Click **Add Peer**
3. Enter:
   - Name (identifier)
   - IP address or hostname
   - Port (default: 8443)
   - Password (optional, recommended)
4. **Test Connection**
5. Save

## Scheduler Configuration

1. Go to **Synchronization** (admin)
2. Enable automatic sync
3. Choose frequency mode:

| Mode | Options |
|------|---------|
| **Interval** | 15min, 30min, 1h, 2h, 4h, 8h, 12h, 24h |
| **Daily** | Every day at specified time (HH:MM) |
| **Weekly** | Every week on specified day and time |
| **Monthly** | Every month on specified day (1-28) and time |

4. Save

## Manual Sync

### Per Share
1. User dashboard
2. Click **Sync** on the backup share

### Force Full Sync (Admin)
1. Go to **Synchronization**
2. Click **Force Sync**

## Authentication

### Server Password

Protects your server against unauthorized syncs.

1. Go to **Settings** (admin)
2. **Synchronization** section
3. Set the password
4. Remote peers must provide it

### Peer Password

To authenticate with a remote server.

1. Go to **Peers**
2. Edit the peer
3. Enter the remote server's password

## Incoming Backups

View peers storing backups on your server.

1. Go to **Incoming Backups** (admin)
2. Statistics shown:
   - Number of peers
   - File count
   - Space used
3. Available actions:
   - View details
   - Delete backups

## Restore

### Web Interface

1. Go to **Restore** (user)
2. Select source peer
3. Browse files
4. Download needed files

### Admin Restore (Bulk)

To restore all users after disaster:

1. Go to **Admin Restore**
2. Select source peer
3. Choose users to restore
4. Start restore

## Encrypted File Format

Files on the remote peer:

```
incoming/
  source_server/
    username/
      share_name/
        file.txt.enc         # Encrypted file
        _manifest.json.enc   # Encrypted manifest
```

Format: `[nonce 12 bytes][encrypted data + auth tag]`

## Monitoring

### Sync Logs

Stored in database with:
- Start/end time
- Status (success/failure)
- File count
- Bytes transferred
- Error message if any

### Dashboard Indicators

- Last successful sync per share
- Connected peer count
- Status of each peer

## Troubleshooting

### "Connection refused" Error

- Check peer is reachable (ping)
- Check port (8443)
- Check firewall

### "401 Unauthorized" Error

- Incorrect peer password
- Update password in peer config

### "403 Forbidden" Error

- Remote server not accepting syncs
- Check remote server settings

### Stuck Sync

```bash
# View running syncs
sudo journalctl -u anemone | grep sync

# Clean zombie syncs (automatic on restart)
sudo systemctl restart anemone
```

## See Also

- [Security](security.md) - Encryption details
- [Advanced Configuration](advanced.md) - Timeouts and parameters
- [Troubleshooting](troubleshooting.md) - Other issues
