# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Anemone is a distributed, encrypted file server with redundancy between peers. It combines WireGuard VPN, rclone encrypted mirroring, Restic config backups, SMB/WebDAV file sharing, and SFTP in a Docker-based architecture. The system enables family/friends to back up each other's data securely without cloud dependencies.

**Core Services**: 3 Docker containers
1. **core** - WireGuard VPN + SFTP server + rclone encrypted mirror + Restic config backups (all-in-one)
2. **shares** - Samba SMB + WebDAV file sharing (optional)
3. **api** - FastAPI web interface for management

**Encryption Architecture**:
- **User data**: rclone crypt with AES-256 (filenames AND content encrypted)
- **Server config**: Restic snapshots with AES-256
- **Encryption key**: Derived from setup wizard, stored encrypted with PBKDF2

## License

**Copyright (C) 2025 juste-un-gars**

This project is licensed under the **GNU Affero General Public License v3.0** (AGPLv3).

All source files should include the following header:

```python
# Anemone - Distributed encrypted file server with peer redundancy
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
# See LICENSE for details.
```

For bash scripts, use the short form header shown above. For Python files with docstrings, include the full AGPLv3 notice as shown in `services/api/main.py`.

When creating new files, always add the appropriate license header.

## Key Architecture Concepts

### Multi-Service Docker Architecture

The system consists of 3 interconnected services:

1. **core** (anemone-core): All-in-one container running via supervisord
   - **WireGuard VPN**: Peer-to-peer mesh network
   - **SFTP server**: Receives encrypted backups from peers
   - **Restic**: Encrypted incremental backups to remote peers

2. **shares** (anemone-shares): File sharing services (optional, profile-based)
   - **Samba SMB**: Windows/macOS file sharing
   - **WebDAV**: HTTP-based file access

3. **api** (anemone-api): Web management interface
   - **FastAPI**: Dashboard and secure setup interface
   - **Recovery tools**: Disaster recovery management

**Critical Detail**: All core services (WireGuard, SFTP, Restic) run in the same container via supervisord. This allows Restic to access the VPN network stack directly without complex networking configuration.

### Encryption Key Management System

The most critical security component is the Restic encryption key lifecycle:

**Setup Flow (Web Interface)**:
- User accesses `http://localhost:3000/setup` on first run
- Choose "New server" (generates key) or "Restore" (import existing key)
- Key is immediately encrypted using AES-256-CBC with PBKDF2 (100k iterations)
- Encryption password derives from **container HOSTNAME** (persistent) + random salt
- Encrypted key saved to `/config/.restic.encrypted`, salt to `/config/.restic.salt`
- Marker file `/config/.setup-completed` prevents re-setup

**Decryption Flow (Container Startup)**:
- Restic entrypoint.sh calls Python script `/scripts/decrypt_key.py`
- Script reads container HOSTNAME, salt, and encrypted key file
- Derives same encryption key via PBKDF2 (HOSTNAME is constant across restarts)
- Decrypts and exports as `RESTIC_PASSWORD` environment variable
- If decryption fails, container refuses to start

**Critical**: HOSTNAME (from `container_name` in docker-compose.yml) is used instead of UUID because Docker assigns a new UUID on each container restart, which would break decryption.

**Key Files**:
- `services/api/main.py`: Encryption logic in `encrypt_restic_key()`
- `services/restic/decrypt_key.py`: Decryption logic (standalone Python script)
- `services/restic/entrypoint.sh`: Orchestrates decryption at startup

### Encrypted Mirroring with rclone

**Critical Security Feature**: User data is encrypted BEFORE leaving the server using rclone's crypt backend.

**How it works**:
1. **Setup**: Script `configure-rclone.py` generates rclone configuration at container startup
2. **Encryption**: Uses the same Restic key (from `RESTIC_PASSWORD` env var)
3. **Dual-layer remotes**:
   - SFTP remote: Connects to peer via SSH over VPN
   - Crypt remote: Wraps SFTP, encrypts everything with AES-256

**What is encrypted**:
- ✅ File contents: AES-256-CTR
- ✅ Filenames: AES-256-EME (completely unreadable)
- ✅ Directory names: Also encrypted
- ✅ File structure: Obfuscated

**Example**: On source server `Documents/photo.jpg` → On peer `gH73kD9f/xY2pL4q.bin`

**Synchronization behavior**:
- `rclone sync` creates an exact encrypted mirror
- `--delete-during` flag: Files deleted on source are deleted on destination
- `--checksum` flag: Verifies integrity, detects file changes
- No versions kept: Simple mirror, not incremental backup

**Key Configuration Files**:
- `services/core/scripts/configure-rclone.py`: Generates `/root/.config/rclone/rclone.conf`
- `services/core/restic-scripts/sync-now.sh`: Executes encrypted sync
- `services/core/scripts/start-restic.sh`: Calls configure-rclone.py at startup

**Decryption** (for recovery):
```bash
# On peer server, to decrypt received data
rclone mount <remote>-crypt: /mnt/decrypted
# Requires same Restic password used by source server
```

**Security guarantees**:
- Peer admin CANNOT read your files (only encrypted blobs)
- Peer admin CANNOT see your file/directory names
- Only someone with your Restic password can decrypt

### Backup Modes

Simplified operational mode controlled via `config/config.yaml`:

- **periodic**: Backup loop every N minutes (30mn, 1h, 6h, or 12h intervals)
  - User chooses frequency during setup
  - Uses `sync-periodic.sh` which calls `sync-now.sh` in a loop
  - Script sleeps between synchronizations

**Note**: The "live" and "scheduled" modes have been removed for simplicity. Periodic mode with short intervals provides equivalent functionality without the complexity of inotify watchers or cron.

The entrypoint.sh switches to periodic mode and execs the sync script.


## Important Implementation Details

### Recent Critical Fixes (2025-10-17)

Three major issues were identified and fixed during deployment testing:

1. **UUID vs HOSTNAME** (most critical): Changed system key from `/proc/sys/kernel/random/uuid` (changes on restart) to `HOSTNAME` environment variable (persistent). This was causing complete decryption failure after container restart.

2. **Read-only /config volume**: API service had `/config:ro` which prevented writing encrypted keys. Changed to read-write for API service only.

3. **OpenSSL → Python cryptography**: Migrated from subprocess openssl calls to native Python cryptography library for better error handling and no external dependencies.

**See** `CORRECTIONS_APPLIQUEES.md` for detailed history and `TROUBLESHOOTING.md` for diagnostic steps.

### Python Cryptography Migration

**Recent change**: All OpenSSL subprocess calls have been replaced with Python's `cryptography` library.

**Why**: Better error handling, no external dependencies, consistent behavior across environments.

**Affected files**:
- `services/api/main.py`: Encryption functions
- `services/api/setup.py`: Key generation (uses `secrets.token_urlsafe(32)`)
- `services/restic/decrypt_key.py`: Decryption script
- `services/restic/entrypoint.sh`: Now calls Python script instead of openssl
- `scripts/init_script.sh`: Random generation uses Python secrets

**Algorithm details**:
- Key derivation: PBKDF2-HMAC-SHA256, 100,000 iterations
- Encryption: AES-256-CBC
- Padding: PKCS7 (manual implementation)
- Format: IV (16 bytes) + encrypted_data

### Enhanced Logging and Error Handling

The `encrypt_restic_key()` function now includes:
- **Detailed debug logging** at each step (system key, salt, derivation, encryption, file writes)
- **Permission checks** before attempting encryption
- **Full traceback** on errors with `flush=True` for immediate visibility in Docker logs

To view detailed encryption process:
```bash
docker logs anemone-api 2>&1 | grep -E "DEBUG|ERROR"
```

Expected debug output during successful setup:
```
DEBUG: System key obtained (length: 15)
DEBUG: Salt generated
DEBUG: Key derived
DEBUG: Cipher initialized
DEBUG: Key padded (length: 48)
DEBUG: Encryption complete
DEBUG: Encrypted key saved to /config/.restic.encrypted
DEBUG: Salt saved to /config/.restic.salt
DEBUG: Setup marker created at /config/.setup-completed
```

### Configuration Volumes

**IMPORTANT**: The `/config` volume mount configuration in `docker-compose.yml`:

```yaml
# Restic service - read-only (can only decrypt, not modify)
restic:
  volumes:
    - ./config:/config:ro

# API service - read-write (needs to write encrypted key during setup)
api:
  volumes:
    - ./config:/config     # NO :ro flag!
```

**Common error**: If the API volume has `:ro`, you'll get:
```
Encryption error: Can't open "/config/.restic.encrypted" for writing, Read-only file system
```

This separation ensures only the web interface can perform setup, while Restic can only decrypt and use keys.

### Git Safety

The following files are in `.gitignore` and must NEVER be committed:
- `config/.restic.encrypted`
- `config/.restic.salt`
- `config/.setup-completed`
- `config/restic-password` (legacy, should not exist in new installs)

These files contain or protect the encryption key. The repository only contains example config files.

## Testing Setup Flow End-to-End

```bash
# 1. Clean slate
rm -f config/.setup-completed config/.restic.* config/restic-password
docker-compose restart api

# 2. Verify redirect to setup (expect 302)
curl -v http://localhost:3000/

# 3. Access http://localhost:3000/setup in browser
# - Choose "New server" or "Restore"
# - Save the generated key
# - Check "I saved my key" and submit

# 4. Verify files created
ls -la config/.restic.encrypted config/.restic.salt config/.setup-completed

# 5. Test decryption survives restart
docker-compose restart restic
docker-compose logs restic  # Should see "✅ Restic key decrypted"

# 6. Verify dashboard accessible (expect 200)
curl http://localhost:3000/
```

## Migration Notes

When updating from older versions:

- Pre-v2 used plain text `config/restic-password` file
- Migration Guide in `MIGRATION_GUIDE.md` covers conversion to encrypted setup
- Old installs can use "Restore" mode with existing key
- Never delete `.restic.encrypted` without backing up the original key first

## Interconnecting Anemone Servers

To connect multiple Anemone servers for mutual backups:

1. **Read the complete guide**: `INTERCONNEXION_GUIDE.md` contains step-by-step instructions
2. **Use the helper script**: `./scripts/add-peer.sh` for interactive peer addition
3. **Key information to exchange** between peers:
   - WireGuard public key (`config/wireguard/public.key`)
   - SSH public key (`config/ssh/id_rsa.pub`)
   - VPN IP address (from `config/config.yaml`)
   - Public endpoint (DynDNS or static IP + port 51820)

**Security note**: Only public keys are exchanged. Private keys and Restic passwords remain secret on each server.

## Development Workflow

### Building and Running

```bash
# Recommended: Use the all-in-one startup script
./start.sh                    # Auto-checks init, then starts Docker

# OR manually:
./scripts/init.sh             # Initial setup (generates WireGuard/SSH keys)
docker compose up --build     # Build and start all services

# Rebuild single service after code changes
docker-compose build api
docker-compose restart api

# View logs
docker-compose logs -f restic          # Backup service logs
docker-compose logs -f api             # Web interface logs
docker-compose logs -f wireguard       # VPN logs
```

### Testing Changes

```bash
# Manual backup trigger
docker exec anemone-restic /scripts/backup-now.sh

# Access Restic CLI directly (requires setup completed)
docker exec -it anemone-restic sh
restic snapshots
restic check

# Test encryption/decryption
curl -X POST http://localhost:3000/setup/complete -F "key=test-key-12345"
docker exec anemone-restic python3 /scripts/decrypt_key.py

# Force re-setup (DESTRUCTIVE - testing only)
rm config/.setup-completed
docker-compose restart api
```

### Network Debugging

```bash
# Check WireGuard status
docker exec anemone-wireguard wg show

# Test peer connectivity from Restic container
docker exec anemone-restic ping 10.8.0.2

# Verify SFTP server
docker logs anemone-sftp
```

## Code Style Conventions

- **Python**: PEP 8, type hints required, Black formatting
- **Bash**: Always use `set -e`, 4-space indent, `${VAR}` syntax
- **Commits**: Conventional Commits format (feat/fix/docs/refactor)
- **Docker**: Prefer Alpine base images, minimize layers

## Security Principles

1. **Key never stored in plaintext** after initial setup
2. **Encryption key derives from container HOSTNAME** - persistent across container restarts (HOSTNAME from `container_name` in docker-compose.yml)
3. **Setup interface is one-time-use** - marker file prevents re-access
4. **Backups are encrypted before leaving the system** - peers only store ciphertext
5. **No network exposure of sensitive services** - SMB/WebDAV only on local network, VPN required for backup access
6. **Separation of concerns** - API can encrypt (write), Restic can only decrypt (read-only /config)

## Common Pitfalls

### ⚠️ CRITICAL: UUID vs HOSTNAME (Container Restart Problem)

**The #1 Problem**: Using `/proc/sys/kernel/random/uuid` as the system key causes decryption to fail after container restart because Docker assigns a new UUID each time.

**Symptom**: Setup works initially, but after `docker-compose restart`, Restic fails with "Failed to decrypt key".

**Root Cause**: The encryption key is derived from:
```python
# ❌ WRONG - UUID changes on restart!
with open('/proc/sys/kernel/random/uuid') as f:
    return f.read().strip()

# ✅ CORRECT - HOSTNAME persists
return os.getenv('HOSTNAME', 'anemone')
```

**Solution**: All three files MUST use `HOSTNAME`:
- `services/api/main.py:get_system_key()`
- `services/api/setup.py:encrypt_restic_key()`
- `services/restic/decrypt_key.py:get_system_key()`

This is already fixed in the current codebase, but if you encounter this error, verify with:
```bash
grep -A3 "def get_system_key" services/api/main.py
# Should show: return os.getenv('HOSTNAME', 'anemone')
```

### Restic Won't Start - Decryption Failed

**Symptom**: `docker logs anemone-restic` shows "❌ Failed to decrypt key"

**Causes**:
1. **UUID vs HOSTNAME mismatch** (see above - most common!)
2. Setup not completed - check for `/config/.setup-completed`
3. Encrypted key or salt file missing
4. Wrong key used during restoration

**Fix**:
- First, check the UUID/HOSTNAME issue above
- If that's correct, complete setup via web interface

### Setup Page Won't Load

**Symptom**: Accessing `/setup` redirects to `/` or vice versa

**Cause**: SetupMiddleware logic in main.py checks `.setup-completed` file

**Fix**: Verify file state matches expected:
- Setup needed: Remove `.setup-completed`
- Setup complete: Ensure `.setup-completed` exists

### Backup Fails Silently

**Symptom**: Restic runs but no snapshots created

**Check**:
1. `config/config.yaml` has valid backup targets
2. Peer is reachable via WireGuard (test with ping)
3. SSH keys are exchanged and in `authorized_keys`
4. Restic password environment variable is set: `docker exec anemone-restic printenv RESTIC_PASSWORD`

## File Locations Reference

- **Config**: `./config/config.yaml` (main configuration, YAML format)
- **Encrypted Key**: `./config/.restic.encrypted` (AES-256-CBC encrypted, binary)
- **Key Salt**: `./config/.restic.salt` (hex-encoded string)
- **Setup Marker**: `./config/.setup-completed` (empty file, presence indicates setup done)
- **WireGuard Keys**: `./config/wireguard/private.key` and `public.key`
- **SSH Keys**: `./config/ssh/id_rsa` and `id_rsa.pub`
- **Logs**: `./logs/` (persistent, rotated)
- **Data**: `./data/` (user files to be backed up)
- **Backups**: `./backups/` (received from peers, encrypted)

## Key File Paths in Code

When modifying encryption/decryption logic, these files must stay synchronized:
- `services/api/main.py` - Encryption logic (`encrypt_restic_key()`, `get_system_key()`)
- `services/api/setup.py` - Key generation (`generate_restic_key()`)
- `services/restic/decrypt_key.py` - Decryption logic (standalone Python script)
- `services/restic/entrypoint.sh` - Orchestrates decryption at startup

## API Endpoints

- `GET /setup` - Initial setup wizard (redirects to `/` if completed)
- `GET /setup/new` - Generate new key page
- `GET /setup/restore` - Restore with existing key page
- `POST /setup/complete` - Finalize setup with key
- `GET /` - Dashboard (redirects to `/setup` if not completed)
- `GET /health` - Health check endpoint
- `GET /api/status` - System status JSON

## When Modifying Encryption Logic

If you need to change encryption/decryption:

1. **Test roundtrip**: Encrypt a test key, decrypt it, verify match
2. **Update both sides**: `main.py` encryption AND `decrypt_key.py` decryption must stay in sync
3. **Maintain compatibility**: Consider migration path for existing installations
4. **Document changes**: Update MIGRATION_GUIDE.md if format changes
5. **Never log keys**: Ensure decrypted keys never appear in logs or stdout

## Backup Script Architecture

Located in `services/restic/scripts/`:

- `backup-now.sh`: One-shot backup to all configured targets
- `backup-live.sh`: Watches filesystem with inotify, triggers backups on changes
- `backup-periodic.sh`: Loop with sleep interval
- `setup-cron.sh`: Generates crontab from config schedule
- `init-repos.sh`: Initializes Restic repositories on first run

Scripts read `/config/config.yaml` via Python YAML parsing.

## Important Architecture Details

### Network Mode: service:wireguard

The Restic container uses `network_mode: "service:wireguard"` in docker-compose.yml, which means:
- Restic shares the WireGuard container's network stack
- Restic can access VPN peers without exposing additional ports
- When debugging network from Restic, you're testing through the VPN

### Volume Mount Permissions

Critical volume configuration in docker-compose.yml:
```yaml
restic:
  volumes:
    - ./config:/config:ro        # Read-only (can only decrypt)

api:
  volumes:
    - ./config:/config           # Read-write (needs to write encrypted key during setup)
```

If API has `:ro`, setup will fail with "Read-only file system" error.

### Docker Network Configuration

**Important**: Anemone uses automatic subnet allocation by Docker to avoid conflicts.

The `anemone-net` network is defined with just `driver: bridge`, without specifying a subnet:
```yaml
networks:
  anemone-net:
    driver: bridge
    # No ipam/subnet config - Docker chooses automatically
```

**Do NOT add static IPs or subnets** unless absolutely necessary, as this causes "Address already in use" errors on machines with multiple Docker projects.
