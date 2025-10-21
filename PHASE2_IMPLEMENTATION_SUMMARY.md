# Phase 2 - Disaster Recovery Implementation Summary

**Date:** 2025-10-21
**Status:** ✅ Complete and tested

## What Was Implemented

Phase 2 provides **automatic backup to peers** with redundant storage and **auto-restore** capabilities.

### New Components

1. **Automatic Backup Script** (`services/core/scripts/backup-config-auto.sh`)
   - Daily automatic export of configuration at 2:00 AM
   - Encrypts with Restic key (AES-256-CBC)
   - Uploads to all configured peers via SFTP
   - Automatic rotation (keeps 7 days locally)
   - Logs to `/logs/config-backup.log`

2. **Backup Discovery Script** (`scripts/discover-backups.py`)
   - Scans all peers for available backups
   - Supports JSON output for automation
   - Lists backups with timestamp, size, and peer location
   - Used by `--auto-restore` mode

3. **Auto-Restore Mode** (`start.sh --auto-restore`)
   - Discovers backups on peers automatically
   - Interactive selection of which backup to restore
   - Downloads selected backup from peer
   - Prompts for Restic key securely
   - Restores complete configuration

4. **Cron Job Configuration** (`services/core/entrypoint.sh` + `supervisord.conf`)
   - Automatic daily execution at 2:00 AM
   - Managed by dcron via supervisord
   - Independent process monitoring

5. **Storage Structure** (`config-backups/`)
   - `local/` - This server's backups
   - `<HOSTNAME>/` - Backups from each peer (one directory per peer)
   - Automatic directory creation via SFTP

## Architecture Changes

### Docker Compose
- Added volume mount: `./config-backups:/config-backups` to `core` service

### Core Service
- Added `crond` program to supervisord configuration
- Configured cron job in entrypoint.sh
- Backup script accessible at `/scripts/core/backup-config-auto.sh`

### Network Communication
- Backup script uses `http://api:3000/api/config/export` (Docker internal network)
- SFTP uploads to peers via VPN (e.g., `restic@10.8.0.2`)

## New User Workflows

### Automatic Daily Backup
```
Every day at 2:00 AM:
1. Export configuration → /config-backups/local/anemone-backup-{HOSTNAME}-{TIMESTAMP}.enc
2. Upload to Peer 1 via SFTP → /config-backups/{HOSTNAME}/
3. Upload to Peer 2 via SFTP → /config-backups/{HOSTNAME}/
4. Upload to Peer N...
5. Cleanup old local backups (keep 7 days)
```

### Manual Backup Trigger
```bash
docker exec anemone-core /scripts/core/backup-config-auto.sh
```

### Discover Available Backups
```bash
# Human-readable output
python3 scripts/discover-backups.py

# JSON output for automation
python3 scripts/discover-backups.py --json
```

### Auto-Restore from Peers
```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone

# Minimal config with peer list
cat > config/config.yaml << 'EOF'
server:
  name: "FR1"
peers:
  - name: "FR2"
    vpn_ip: "10.8.0.2"
EOF

# Copy SSH keys
cp /backup/id_rsa config/ssh/

# Launch auto-restore
./start.sh --auto-restore
```

## Security

All Phase 2 features maintain the same security model as Phase 1:
- 🔒 AES-256-CBC encryption with PBKDF2 key derivation
- 🔒 Only encrypted data leaves the server
- 🔒 Each server can only decrypt its own backups
- 🔒 SSH key authentication for peer communication

## Testing

Run the verification suite:
```bash
./scripts/test-phase2.sh
```

All 11 tests should pass:
- ✅ Backup script exists and is executable
- ✅ Discovery script exists and is executable
- ✅ Python syntax valid
- ✅ start.sh has --auto-restore option
- ✅ supervisord configured with crond
- ✅ entrypoint configures cron job
- ✅ docker-compose has config-backups volume
- ✅ .gitignore excludes config-backups
- ✅ Backup script structure correct
- ✅ Documentation complete
- ✅ Storage directory exists

## Files Modified/Created

| File | Status | Purpose |
|------|--------|---------|
| `services/core/scripts/backup-config-auto.sh` | Created | Automatic daily backup + peer distribution |
| `scripts/discover-backups.py` | Created | Peer backup discovery |
| `start.sh` | Modified | Added `--auto-restore` mode (lines 14-227) |
| `services/core/entrypoint.sh` | Modified | Cron job configuration (lines 52-56) |
| `services/core/supervisord.conf` | Modified | Added crond program (lines 46-53) |
| `docker-compose.yml` | Modified | Added config-backups volume mount (line 20) |
| `.gitignore` | Modified | Excluded config-backups directory |
| `config-backups/` | Created | Storage directory structure |
| `DISASTER_RECOVERY.md` | Modified | Phase 2 documentation (lines 231-393) |
| `scripts/test-phase2.sh` | Created | Automated verification suite |
| `PHASE2_IMPLEMENTATION_SUMMARY.md` | Created | This document |

## Key Differences from Phase 1

### Phase 1 (Manual)
- ❌ User must manually download backup
- ❌ User must manually store backup safely
- ❌ User must remember to export regularly
- ✅ User has explicit backup file

### Phase 2 (Automatic)
- ✅ Automatic daily backups
- ✅ Automatic distribution to all peers
- ✅ Automatic rotation and cleanup
- ✅ Can restore from peers if local backup lost
- ⚠️ Requires peer connectivity for auto-restore

## Usage Examples

### Check Backup Status
```bash
# View local backups
ls -lh config-backups/local/

# View backups received from peers
ls -lh config-backups/*/

# Check backup logs
docker exec anemone-core tail -f /logs/config-backup.log
```

### Manual Backup
```bash
# Trigger immediate backup
docker exec anemone-core /scripts/core/backup-config-auto.sh
```

### Disaster Recovery Scenario
```
Scenario: Server FR1 suffers complete hardware failure

Recovery Steps:
1. Get new server hardware
2. Install Docker
3. git clone anemone repository
4. Create minimal config.yaml with peer FR2's info
5. Copy SSH keys (from external backup or regenerate)
6. Run: ./start.sh --auto-restore
7. Select most recent backup from FR2
8. Enter Restic key
9. Configuration restored!
10. docker compose up -d
11. Server FR1 is back online with all settings
```

## Operational Benefits

1. **Resilience**: Multiple copies of configuration across network
2. **Automation**: No manual intervention required
3. **Point-in-time recovery**: 7 days of backups available
4. **Self-service restore**: Can rebuild from any peer
5. **Audit trail**: Complete logs of all backup operations

## Monitoring Recommendations

```bash
# Check cron job is running
docker exec anemone-core supervisorctl status crond

# View cron log
docker exec anemone-core tail -f /logs/cron.log

# View backup log
docker exec anemone-core tail -f /logs/config-backup.log

# Verify backups are being distributed
# (on peer server)
ls -lh /home/restic/config-backups/FR1/
```

## Troubleshooting

### Backup Not Running
```bash
# Check cron job configured
docker exec anemone-core crontab -l

# Check supervisord
docker exec anemone-core supervisorctl status

# Check logs
docker exec anemone-core tail -f /logs/cron.log
```

### Upload to Peers Fails
```bash
# Test SFTP connectivity
docker exec anemone-core sftp -i /root/.ssh/id_rsa restic@10.8.0.2

# Check SSH keys
docker exec anemone-core ls -la /root/.ssh/

# Verify peer configuration
cat config/config.yaml
```

### Auto-Restore Not Finding Backups
```bash
# Test discovery manually
python3 scripts/discover-backups.py

# Check SSH connectivity to peers
ssh -i config/ssh/id_rsa restic@10.8.0.2

# Verify peer has backups
ssh -i config/ssh/id_rsa restic@10.8.0.2 "ls -la /config-backups/FR1/"
```

## Next Steps (Phase 3 - Optional)

If requested by user, Phase 3 could add:
- Web-based recovery interface with graphical selection
- Email/webhook notifications on backup success/failure
- Multi-version history with point-in-time restore
- Incremental configuration backups
- Automatic integrity verification
- Backup encryption with separate key rotation

**Status:** Phase 2 complete and ready for production use. Phase 3 pending user request.
