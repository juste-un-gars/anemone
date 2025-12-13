# Anemone Troubleshooting Guide

This guide provides commands and procedures for diagnosing and resolving common issues with Anemone NAS.

## Table of Contents

- [Check Synchronization Status](#check-synchronization-status)
- [Diagnose Stuck/Zombie Syncs](#diagnose-stuckzombie-syncs)
- [Check Peer Connectivity](#check-peer-connectivity)
- [View and Analyze Logs](#view-and-analyze-logs)
- [Check Disk Space and Quotas](#check-disk-space-and-quotas)
- [Manual Cleanup Operations](#manual-cleanup-operations)
- [Database Queries](#database-queries)
- [Common Issues and Solutions](#common-issues-and-solutions)

---

## Check Synchronization Status

### Check if a sync is currently running

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, user_id, peer_id, started_at, status FROM sync_log WHERE status = 'running';"
```

**Expected output:**
- If empty: No sync is running
- If shows a row: Sync is in progress (note the `started_at` time to check if it's stuck)

### View recent sync history

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, started_at, completed_at, status, files_synced, bytes_synced, error_message FROM sync_log ORDER BY started_at DESC LIMIT 5;"
```

### Check sync progress in real-time

```bash
# Watch the sync_log table (refresh every 2 seconds)
watch -n 2 'sqlite3 /srv/anemone/db/anemone.db "SELECT id, started_at, status, files_synced, bytes_synced FROM sync_log WHERE status = '\''running'\'';"'
```

### Monitor memory usage during sync

```bash
watch -n 1 'free -h; echo "---"; ps aux | grep anemone | grep -v grep'
```

**What to look for:**
- RAM should stay stable (under 500MB even with large files on v0.9.8-beta+)
- If RAM keeps growing → possible memory leak
- If OOM killer appears in logs → upgrade to v0.9.8-beta (chunked encryption)

---

## Diagnose "Silent" Syncs (BuildManifest Phase)

### Understanding the sync phases

A synchronization goes through several phases:

1. **Building local manifest** (can take 15-30 minutes for 40k+ files)
   - Scans all files in the share
   - Calculates SHA256 hash for each file
   - **No logs during this phase** (improvement planned)
   - CPU usage: 40-70%

2. **Fetching remote manifest** (2-5 seconds)
   - Downloads manifest from peer

3. **Comparing manifests** (1-2 seconds)
   - Determines which files to add/update/delete

4. **Uploading files** (hours for large datasets)
   - Uploads new/modified files to peer
   - **No progress logs** (improvement planned)

### Check if sync is building manifest

If sync shows "running" but no logs appear, it's likely building the manifest:

```bash
# 1. Check sync age
sqlite3 /srv/anemone/db/anemone.db "SELECT id, started_at,
  CAST((julianday('now') - julianday(started_at)) * 60 AS INT) || ' min' as duration
  FROM sync_log WHERE status = 'running';"

# 2. Check CPU usage (should be 40-70% during BuildManifest)
ps aux | grep "[a]nemone" | awk '{print "CPU: " $3 "% | MEM: " $4 "% | TIME: " $10}'

# 3. Count files in the share (to estimate BuildManifest time)
find /srv/anemone/shares/USERNAME/SHARENAME -type f | wc -l
```

**Estimation:**
- ~1000 files = 30 seconds
- ~10,000 files = 5 minutes
- ~40,000 files = 15-20 minutes
- ~100,000 files = 45-60 minutes

### Check if sync is uploading files

Once BuildManifest completes, logs will show:

```
📊 Sync user X to peer Y:
   Local manifest has XXXXX files
   Remote manifest is nil (first sync)
   Delta: XXXXX to add, 0 to update, 0 to delete
```

To check upload progress:

```bash
# On the receiving peer (destination server)
find /srv/anemone/backups/incoming/SOURCE_SERVER_NAME -type f | wc -l

# Compare with total files to sync (from logs above)
# Calculate percentage: (files_received / total_files) * 100
```

### Monitor sync in real-time

```bash
# Terminal 1: Watch database for updates
watch -n 5 'sqlite3 /srv/anemone/db/anemone.db "SELECT id, status, files_synced,
  ROUND(bytes_synced/1024.0/1024.0/1024.0, 2) || '\'' GB'\'' as data_synced
  FROM sync_log WHERE status = '\''running'\'';"'

# Terminal 2: Monitor CPU/Memory
watch -n 2 'ps aux | grep "[a]nemone" | awk '\''{print "CPU: " $3 "% | MEM: " $4 "% | TIME: " $10}'\'

# Terminal 3: Count received files on peer (if accessible)
# ssh user@peer-server
watch -n 10 'find /srv/anemone/backups/incoming/YOUR_SERVER -type f | wc -l'
```

---

## Diagnose Stuck/Zombie Syncs

### Identify zombie syncs (stuck for >2 hours)

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, user_id, peer_id, started_at,
  CAST((julianday('now') - julianday(started_at)) * 24 AS INT) || 'h' AS duration
  FROM sync_log
  WHERE status = 'running'
  AND datetime(started_at) < datetime('now', '-2 hours');"
```

### Check logs for a specific sync

```bash
# Get the start time from sync_log first, then:
sudo journalctl -u anemone --since "2025-12-13 12:08:00" | grep -E "sync|Sync|upload|Upload|error|Error"
```

### Check if sync is making progress

```bash
# Run this multiple times with a few seconds interval
sqlite3 /srv/anemone/db/anemone.db "SELECT files_synced, bytes_synced FROM sync_log WHERE id = 5;"
```

**If values don't change** → sync is stuck

---

## Check Peer Connectivity

### List configured peers

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, name, address, port FROM peers;"
```

### Test network connectivity to a peer

```bash
# Replace 10.8.0.5 with your peer's address
ping -c 3 10.8.0.5
```

### Test if peer's web service is responding

```bash
# Replace with your peer's address and port
curl -k -m 5 https://10.8.0.5:8443/ 2>&1 | head -5
```

**Expected output:**
- `<a href="/login">See Other</a>` → Service is running correctly
- `Connection refused` → Service is down
- `Connection timed out` → Network issue or firewall

### Check peer authentication

```bash
# Get peer password from database
sqlite3 /srv/anemone/db/anemone.db "SELECT name, password FROM peers WHERE id = 1;"
```

**Note:** Passwords are encrypted. This is just to verify a password is configured.

### Verify peer is reachable via SSH (if VPN/WireGuard)

```bash
# Test SSH connection to peer
ssh franck@10.8.0.5 'echo "Connection OK"'
```

---

## View and Analyze Logs

### View logs in real-time

```bash
sudo journalctl -u anemone -f
```

### View logs from a specific time

```bash
# Today's logs
sudo journalctl -u anemone --since "today"

# Last hour
sudo journalctl -u anemone --since "1 hour ago"

# Specific date/time
sudo journalctl -u anemone --since "2025-12-13 12:00:00"
```

### Search logs for errors

```bash
sudo journalctl -u anemone --since "today" | grep -i error
```

### Search logs for OOM (Out of Memory) issues

```bash
sudo journalctl -u anemone --since "yesterday" | grep -i "oom\|out of memory\|killed"
```

### View service status

```bash
sudo systemctl status anemone
```

### Check if service is enabled (auto-start on boot)

```bash
systemctl is-enabled anemone
```

---

## Check Disk Space and Quotas

### Check overall disk usage

```bash
df -h
```

### Check Btrfs quota usage (if using Btrfs)

```bash
# Show quota usage for all users
sudo btrfs qgroup show /srv/anemone/shares
```

### Check space used by a specific user

```bash
# Replace 'franck' with username
sudo du -sh /srv/anemone/shares/franck
```

### Check backup space usage

```bash
# Local backups (incoming from peers)
sudo du -sh /srv/anemone/backups/incoming/*

# Server backups
sudo du -sh /srv/anemone/backups/server
```

### Check trash space usage

```bash
sudo du -sh /srv/anemone/trash/*
```

---

## Manual Cleanup Operations

### Clean up a zombie sync

```bash
# Replace ID with the stuck sync's ID
sqlite3 /srv/anemone/db/anemone.db "UPDATE sync_log
  SET status = 'error',
      completed_at = CURRENT_TIMESTAMP,
      error_message = 'Sync timeout - manually cleaned (stuck >2h)'
  WHERE id = 5;"
```

### Clean up all zombie syncs at once

```bash
sqlite3 /srv/anemone/db/anemone.db "UPDATE sync_log
  SET status = 'error',
      completed_at = CURRENT_TIMESTAMP,
      error_message = 'Sync timeout - automatically cleaned up'
  WHERE status = 'running'
  AND datetime(started_at) < datetime('now', '-2 hours');"
```

**Note:** v0.9.6-beta+ does this automatically on startup.

### Manually trigger trash cleanup

```bash
# Delete trash items older than 30 days
# This is normally done automatically at 3 AM daily
sudo journalctl -u anemone -f &
# Then trigger from admin web interface: Settings → Trash Management
```

### Restart the service

```bash
sudo systemctl restart anemone
```

---

## Database Queries

### Check database version

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT key, value FROM system_info WHERE key = 'current_version';"
```

### List all users

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, username, is_admin, is_active, created_at FROM users;"
```

### Check user encryption keys

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT id, username,
  CASE WHEN encryption_key IS NOT NULL THEN 'YES' ELSE 'NO' END AS has_key
  FROM users;"
```

### View sync statistics

```bash
# Total syncs by status
sqlite3 /srv/anemone/db/anemone.db "SELECT status, COUNT(*) as count
  FROM sync_log
  GROUP BY status;"

# Total data synced
sqlite3 /srv/anemone/db/anemone.db "SELECT
  COUNT(*) as total_syncs,
  SUM(files_synced) as total_files,
  ROUND(SUM(bytes_synced) / 1024.0 / 1024.0 / 1024.0, 2) || ' GB' as total_data
  FROM sync_log
  WHERE status = 'success';"
```

### Check update information

```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT key, value, updated_at FROM system_info WHERE key LIKE 'update_%';"
```

---

## Common Issues and Solutions

### Issue: Sync shows "running" but nothing happens

**Symptoms:**
- Sync status = 'running' in database
- No logs after "triggered forced synchronization"
- `files_synced` and `bytes_synced` stay at 0

**Diagnosis:**
```bash
# Check peer connectivity
ping -c 3 [peer_address]
curl -k -m 5 https://[peer_address]:8443/

# Check logs for errors
sudo journalctl -u anemone --since "[sync_start_time]" -n 100
```

**Solution:**
1. Check peer is online and reachable
2. Verify peer password is correct
3. Check firewall rules (port 8443 must be open)
4. If stuck >2h, clean up the zombie sync manually
5. Retry synchronization

---

### Issue: OOM Killer terminating Anemone during sync

**Symptoms:**
```
kernel: Out of memory: Killed process (anemone)
systemd: anemone.service: A process of this unit has been killed by the OOM killer.
```

**Solution:**
- **Upgrade to v0.9.8-beta or later** (implements chunked encryption with 128MB chunks)
- v0.9.7-beta and earlier load entire files into RAM, causing OOM on systems with <4GB RAM

**Verify version:**
```bash
sqlite3 /srv/anemone/db/anemone.db "SELECT value FROM system_info WHERE key='current_version';"
```

---

### Issue: Path traversal error during sync

**Symptoms:**
```
Failed to upload file: path traversal detected
```

**Cause:**
- Filenames containing `...` (three dots) were incorrectly flagged as path traversal in v0.9.6-beta and earlier

**Solution:**
- Upgrade to v0.9.7-beta or later (improved path validation)

---

### Issue: Update system shows new version but update fails

**Common errors:**
```
go: command not found
ERROR: Failed to build anemone binary
```

**Solution 1 - Missing Go PATH:**
```bash
# Add to /etc/profile.d/go.sh (if not exists)
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Then logout and login again
```

**Solution 2 - Missing sudo permission:**
```bash
# Add permission for service restart
echo "$USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart anemone" | sudo tee /etc/sudoers.d/anemone-restart
sudo chmod 0440 /etc/sudoers.d/anemone-restart
```

---

### Issue: Git tag conflict during update

**Symptoms:**
```
! [rejected]        v0.9.2-beta -> v0.9.2-beta  (would clobber existing tag)
ERROR: Failed to fetch tags from GitHub
```

**Solution:**
```bash
cd ~/anemone
git tag -d [conflicting_tag]  # e.g., v0.9.2-beta
# Then retry update from web interface
```

---

### Issue: Database locked errors

**Symptoms:**
```
Failed to cleanup zombie sync: database is locked
```

**Cause:**
- SQLite doesn't allow UPDATE during an active SELECT query

**Solution:**
- This is fixed in v0.9.9-beta and later
- For older versions: wait a few seconds and retry, or restart the service

---

## Getting Help

If you encounter an issue not covered here:

1. **Check logs first:**
   ```bash
   sudo journalctl -u anemone --since "1 hour ago" | grep -i error
   ```

2. **Check GitHub issues:**
   - https://github.com/juste-un-gars/anemone/issues

3. **Create a new issue with:**
   - Anemone version
   - Operating system
   - Full error logs
   - Steps to reproduce

---

## Maintenance Commands

### Check service health

```bash
# All-in-one health check
echo "=== Service Status ===" && sudo systemctl status anemone --no-pager
echo -e "\n=== Version ===" && sqlite3 /srv/anemone/db/anemone.db "SELECT value FROM system_info WHERE key='current_version';"
echo -e "\n=== Running Syncs ===" && sqlite3 /srv/anemone/db/anemone.db "SELECT COUNT(*) FROM sync_log WHERE status='running';"
echo -e "\n=== Disk Space ===" && df -h /srv/anemone
echo -e "\n=== Memory Usage ===" && free -h
```

### Verify database integrity

```bash
sqlite3 /srv/anemone/db/anemone.db "PRAGMA integrity_check;"
```

Expected output: `ok`

### Backup database manually

```bash
cp /srv/anemone/db/anemone.db /srv/anemone/db/anemone.db.backup-$(date +%Y%m%d-%H%M%S)
```

---

**Last updated:** 2025-12-13
**Compatible with:** Anemone v0.9.0-beta and later
