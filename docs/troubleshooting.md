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

```bash
# Edit service
sudo systemctl edit anemone

# Add environment variable
[Service]
Environment="DEBUG=true"

# Restart
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

## Getting Help

If issues persist:

1. Collect logs: `sudo journalctl -u anemone > anemone.log`
2. Check GitHub issues: https://github.com/juste-un-gars/anemone/issues
3. Open new issue with logs and system info

## See Also

- [Installation](installation.md) - Setup issues
- [P2P Sync](p2p-sync.md) - Sync-specific issues
- [Advanced Configuration](advanced.md) - Custom settings
