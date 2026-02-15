# Troubleshooting

## Web Interface

### Can't Access Web Interface

```bash
# Check if service is running
sudo systemctl status anemone

# View logs
sudo journalctl -u anemone -f

# Check if process is running
ps aux | grep anemone
```

### Certificate Error

Self-signed certificate warning is normal. Add exception in browser:
- Chrome: Click "Advanced" → "Proceed to site"
- Firefox: Click "Advanced" → "Accept the Risk"

### "502 Bad Gateway" or Blank Page

```bash
# Restart service
sudo systemctl restart anemone

# Check for errors
sudo journalctl -u anemone -n 100
```

## SMB Shares

### Can't Access Shares

```bash
# Check Samba service
sudo systemctl status smb    # Fedora
sudo systemctl status smbd   # Debian/Ubuntu

# Test configuration
sudo testparm -s

# List SMB users
sudo pdbedit -L
```

### "Access Denied" Error

```bash
# Verify user exists in Samba
sudo pdbedit -L | grep username

# Reset SMB password
sudo smbpasswd username

# Check SELinux (Fedora/RHEL)
ls -laZ /srv/anemone/shares/
sudo ausearch -m avc -ts recent | grep samba
```

### Shares Not Visible

```bash
# Regenerate SMB config
sudo systemctl restart anemone
sudo systemctl reload smbd

# Check generated config
cat /srv/anemone/smb/smb.conf
```

## Database

### "Database Locked" Error

```bash
# Stop service
sudo systemctl stop anemone

# Check for locks
fuser /srv/anemone/db/anemone.db

# Restart
sudo systemctl start anemone
```

### Reset Database (WARNING: Deletes All Data)

```bash
sudo systemctl stop anemone
sudo rm /srv/anemone/db/anemone.db
sudo systemctl start anemone
# Then complete setup wizard again
```

## Synchronization

### Sync Not Working

```bash
# Check peer connectivity
ping peer-address
curl -k https://peer-address:8443/health

# View sync logs
sudo journalctl -u anemone | grep -i sync
```

### "Connection Refused"

- Verify peer address and port
- Check firewall on both servers
- Ensure Anemone is running on peer

```bash
# Open port on firewall
sudo firewall-cmd --add-port=8443/tcp --permanent
sudo firewall-cmd --reload
```

### "401 Unauthorized"

Incorrect sync password. Update in:
- Admin → Peers → Edit peer → Password

### "403 Forbidden"

Remote server rejecting sync. Check:
- Remote server's sync password
- Remote server's sync settings

### Sync Stuck at 0%

```bash
# Check for zombie syncs
sudo journalctl -u anemone | grep "sync started"

# Restart cleans zombie syncs
sudo systemctl restart anemone
```

## Storage

### "No Space Left" Error

```bash
# Check disk usage
df -h /srv/anemone

# Check user quotas
btrfs qgroup show /srv/anemone/shares
```

### Quota Not Enforced

Quotas only work on Btrfs filesystem.

```bash
# Check filesystem type
df -T /srv/anemone

# If not Btrfs, quotas are tracking-only
```

### ZFS Pool Issues

```bash
# Check pool status
zpool status anemone-pool

# Check pool usage
zpool list

# Import existing pool
sudo zpool import anemone-pool
```

## Permissions

### "Permission Denied" on Files

```bash
# Fix ownership
sudo chown -R username:username /srv/anemone/shares/username

# Fix permissions
sudo chmod -R 755 /srv/anemone/shares/username
```

### SELinux Blocking Access (Fedora/RHEL)

```bash
# Check for SELinux denials
sudo ausearch -m avc -ts recent

# Apply Samba context
sudo semanage fcontext -a -t samba_share_t "/srv/anemone/shares(/.*)?"
sudo restorecon -Rv /srv/anemone/shares
```

## Service

### Service Won't Start

```bash
# Check configuration
sudo journalctl -u anemone -n 50

# Verify binary exists
ls -la /usr/local/bin/anemone

# Check data directory
ls -la /srv/anemone
```

### Service Crashes

```bash
# Check for panics
sudo journalctl -u anemone | grep -i panic

# Check memory
free -h

# Increase limits if needed
sudo systemctl edit anemone
# Add: LimitNOFILE=65535
```

### Auto-Start Not Working

```bash
# Enable service
sudo systemctl enable anemone

# Check mount dependencies
sudo systemctl edit anemone
# Add: RequiresMountsFor=/srv/anemone
```

## Logs

### View All Logs

```bash
# Follow live
sudo journalctl -u anemone -f

# Last 100 lines
sudo journalctl -u anemone -n 100

# Since last boot
sudo journalctl -u anemone -b

# Errors only
sudo journalctl -u anemone -p err
```

### Enable Debug Mode

Change log level from the web UI (Admin → System Logs) or via environment variable:

```bash
ANEMONE_LOG_LEVEL=debug
sudo systemctl restart anemone
```

### Setup Wizard Issues

If the setup wizard gets stuck or you need to reconfigure storage options:

```bash
# Restart Anemone to reset the wizard state
sudo systemctl restart anemone
```

## Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| "session manager not initialized" | DB connection issue | Restart service |
| "failed to generate session ID" | Random source issue | Check `/dev/urandom` |
| "no such table" | Migration failed | Delete DB and restart |
| "TLS handshake error" | Certificate issue | Check cert paths |
| "address already in use" | Port conflict | Check for other services on 8443 |

## Advanced Sync Diagnostics

### Understanding Sync Phases

A synchronization goes through several phases:

1. **Building local manifest** (can take 15-30 minutes for 40k+ files)
   - Scans all files in the share and calculates SHA256 hashes
   - CPU usage: 40-70%
2. **Fetching remote manifest** (2-5 seconds)
3. **Comparing manifests** (1-2 seconds)
4. **Uploading files** (depends on dataset size and network speed)

### Check Sync Status

```bash
# Check if a sync is currently running
sqlite3 /srv/anemone/db/anemone.db "SELECT id, user_id, peer_id, started_at, status FROM sync_log WHERE status = 'running';"

# View recent sync history
sqlite3 /srv/anemone/db/anemone.db "SELECT id, started_at, completed_at, status, files_synced, bytes_synced, error_message FROM sync_log ORDER BY started_at DESC LIMIT 5;"
```

### Identify Zombie Syncs (Stuck > 2 Hours)

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, user_id, peer_id, started_at,
  CAST((julianday('now') - julianday(started_at)) * 24 AS INT) || 'h' AS duration
  FROM sync_log
  WHERE status = 'running'
  AND datetime(started_at) < datetime('now', '-2 hours');"
```

### Clean Up Zombie Syncs

```bash
sqlite3 /srv/anemone/db/anemone.db "UPDATE sync_log
  SET status = 'error',
      completed_at = CURRENT_TIMESTAMP,
      error_message = 'Sync timeout - manually cleaned'
  WHERE status = 'running'
  AND datetime(started_at) < datetime('now', '-2 hours');"
```

### Monitor Memory During Sync

```bash
watch -n 1 'free -h; echo "---"; ps aux | grep anemone | grep -v grep'
```

If RAM keeps growing or OOM killer appears, upgrade to v0.9.8-beta+ (chunked encryption with 128MB chunks).

### Estimate BuildManifest Duration

| Files | Estimated time |
|-------|---------------|
| ~1,000 | 30 seconds |
| ~10,000 | 5 minutes |
| ~40,000 | 15-20 minutes |
| ~100,000 | 45-60 minutes |

```bash
# Count files in a share
find /srv/anemone/shares/USERNAME/SHARENAME -type f | wc -l
```

## Database Queries

### List All Users

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, username, is_admin, is_active, created_at FROM users;"
```

### View Sync Statistics

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT
  COUNT(*) as total_syncs,
  SUM(files_synced) as total_files,
  ROUND(SUM(bytes_synced) / 1024.0 / 1024.0 / 1024.0, 2) || ' GB' as total_data
  FROM sync_log
  WHERE status = 'success';"
```

### Verify Database Integrity

```bash
sqlite3 /srv/anemone/db/anemone.db "PRAGMA integrity_check;"
# Expected output: ok
```

### Backup Database Manually

```bash
cp /srv/anemone/db/anemone.db /srv/anemone/db/anemone.db.backup-$(date +%Y%m%d-%H%M%S)
```

## Maintenance

### All-in-One Health Check

```bash
echo "=== Service Status ===" && sudo systemctl status anemone --no-pager
echo -e "\n=== Running Syncs ===" && sqlite3 /srv/anemone/db/anemone.db "SELECT COUNT(*) FROM sync_log WHERE status='running';"
echo -e "\n=== Disk Space ===" && df -h /srv/anemone
echo -e "\n=== Memory Usage ===" && free -h
```

### Update Issues

**"go: command not found" during update:**

```bash
# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
```

**Git tag conflict during update:**

```bash
cd ~/anemone
git tag -d [conflicting_tag]
# Then retry update from web interface
```

## Getting Help

If issues persist:

1. Collect logs: `sudo journalctl -u anemone > anemone.log`
2. Check GitHub issues: https://github.com/juste-un-gars/anemone/issues
3. Open new issue with logs and system info

## See Also

- [Installation](installation.md) - Setup issues
- [P2P Sync](p2p-sync.md) - Sync-specific issues
- [Advanced Configuration](advanced.md) - Custom settings
