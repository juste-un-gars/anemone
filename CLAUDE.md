# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Anemone is a distributed, encrypted file server with redundancy between peers. It combines WireGuard VPN, Restic encrypted backups, SMB/WebDAV file sharing, and SFTP in a Docker-based architecture. The system enables family/friends to back up each other's data securely without cloud dependencies.

## Key Architecture Concepts

### Multi-Service Docker Architecture

The system consists of 5 interconnected services:

1. **WireGuard (VPN)**: Peer-to-peer mesh network
2. **Restic**: Encrypted incremental backups to remote peers
3. **Samba/WebDAV**: Local file access interfaces
4. **SFTP**: Receives encrypted backups from peers
5. **API**: Web dashboard and secure setup interface

**Critical Network Detail**: The Restic service uses `network_mode: "service:wireguard"` which means it shares the WireGuard container's network stack. This allows Restic to access peers via the VPN without exposing additional ports.

### Encryption Key Management System

The most critical security component is the Restic encryption key lifecycle:

**Setup Flow (Web Interface)**:
- User accesses `http://localhost:3000/setup` on first run
- Choose "New server" (generates key) or "Restore" (import existing key)
- Key is immediately encrypted using AES-256-CBC with PBKDF2 (100k iterations)
- Encryption password derives from system UUID + random salt
- Encrypted key saved to `/config/.restic.encrypted`, salt to `/config/.restic.salt`
- Marker file `/config/.setup-completed` prevents re-setup

**Decryption Flow (Container Startup)**:
- Restic entrypoint.sh calls Python script `/scripts/decrypt_key.py`
- Script reads system UUID, salt, and encrypted key file
- Derives same encryption key via PBKDF2
- Decrypts and exports as `RESTIC_PASSWORD` environment variable
- If decryption fails, container refuses to start

**Key Files**:
- `services/api/main.py`: Encryption logic in `encrypt_restic_key()`
- `services/restic/decrypt_key.py`: Decryption logic (standalone Python script)
- `services/restic/entrypoint.sh`: Orchestrates decryption at startup

### Backup Modes

Three operational modes controlled via `config/config.yaml`:

- **live**: Uses `inotify-tools` to watch filesystem, triggers immediate backup on changes (debounced)
- **periodic**: Backup loop every N minutes
- **scheduled**: Cron-based backup at specific times

The entrypoint.sh switches between modes using bash case statement and exec.

## Essential Commands

### Development & Testing

```bash
# Initial setup (generates WireGuard/SSH keys, creates structure)
./scripts/init.sh

# Build and start all services
docker-compose up --build

# Rebuild single service after code changes
docker-compose build api
docker-compose restart api

# View logs
docker-compose logs -f restic          # Backup service logs
docker-compose logs -f api             # Web interface logs
docker-compose logs -f wireguard       # VPN logs

# Manual backup trigger
docker exec anemone-restic /scripts/backup-now.sh

# Access Restic CLI directly (requires setup completed)
docker exec -it anemone-restic sh
restic snapshots
restic check
```

### Testing Key Encryption/Decryption

```bash
# Test encryption (API must be running)
curl -X POST http://localhost:3000/setup/complete -F "key=test-key-12345"

# Verify encrypted files exist
ls -la config/.restic.encrypted config/.restic.salt config/.setup-completed

# Test decryption (simulates Restic startup)
docker exec anemone-restic python3 /scripts/decrypt_key.py

# Force re-setup (DESTRUCTIVE - only for testing)
rm config/.setup-completed
docker-compose restart api
# Access http://localhost:3000/setup
```

### Debugging Network Issues

```bash
# Check WireGuard status
docker exec anemone-wireguard wg show

# Test peer connectivity from Restic container (shares WireGuard network)
docker exec anemone-restic ping 10.8.0.2  # Replace with peer IP

# Verify SFTP server is receiving connections
docker logs anemone-sftp
```

## Important Implementation Details

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

### Configuration Volumes

The `/config` volume is mounted as **read-only** in most containers EXCEPT the API service which needs write access during setup. The Restic service can read encrypted keys but cannot modify them.

This separation ensures only the web interface can perform setup, while Restic can only decrypt and use keys.

### Git Safety

The following files are in `.gitignore` and must NEVER be committed:
- `config/.restic.encrypted`
- `config/.restic.salt`
- `config/.setup-completed`
- `config/restic-password` (legacy, should not exist in new installs)

These files contain or protect the encryption key. The repository only contains example config files.

## Testing the Setup Flow

Complete test scenario:

```bash
# 1. Clean slate
rm -f config/.setup-completed config/.restic.* config/restic-password
docker-compose restart api

# 2. Should redirect to setup
curl -v http://localhost:3000/  # Expect 302 to /setup

# 3. Access setup in browser
# Go to http://localhost:3000/setup
# Choose "New server"
# Copy the generated key to clipboard

# 4. Complete setup
# Check "I saved my key" and submit

# 5. Verify files created
ls -la config/.restic.encrypted config/.restic.salt config/.setup-completed

# 6. Restart Restic to test decryption
docker-compose restart restic
docker-compose logs restic  # Should see "✅ Restic key decrypted"

# 7. Verify dashboard accessible
curl http://localhost:3000/  # Expect 200 OK, not redirect
```

## Migration Notes

When updating from older versions:

- Pre-v2 used plain text `config/restic-password` file
- Migration Guide in `MIGRATION_GUIDE.md` covers conversion to encrypted setup
- Old installs can use "Restore" mode with existing key
- Never delete `.restic.encrypted` without backing up the original key first

## Code Style Conventions

Based on CONTRIBUTING.md:

- **Python**: PEP 8, type hints required, Black formatting
- **Bash**: Always use `set -e`, 4-space indent, `${VAR}` syntax
- **Commits**: Conventional Commits format (feat/fix/docs/refactor)
- **Docker**: Prefer Alpine base images, minimize layers

## Security Principles

1. **Key never stored in plaintext** after initial setup
2. **Encryption key derives from system UUID** - tied to specific machine
3. **Setup interface is one-time-use** - marker file prevents re-access
4. **Backups are encrypted before leaving the system** - peers only store ciphertext
5. **No network exposure of sensitive services** - SMB/WebDAV only on local network, VPN required for backup access

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

- **Config**: `./config/config.yaml` (main configuration)
- **Encrypted Key**: `./config/.restic.encrypted` (AES-256-CBC encrypted)
- **Key Salt**: `./config/.restic.salt` (hex-encoded)
- **Setup Marker**: `./config/.setup-completed` (empty file)
- **Logs**: `./logs/` (persistent, rotated)
- **Data**: `./data/` (user files to be backed up)
- **Backups**: `./backups/` (received from peers, encrypted)

## API Endpoints

- `GET /setup` - Initial setup wizard (only accessible if not completed)
- `GET /setup/new` - Generate new key
- `GET /setup/restore` - Restore with existing key
- `POST /setup/complete` - Finalize setup with key
- `GET /` - Dashboard (only accessible after setup)
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
- `backup-live.sh`: Watches filesystem with inotify, triggers backups
- `backup-periodic.sh`: Loop with sleep interval
- `setup-cron.sh`: Generates crontab from config schedule
- `init-repos.sh`: Initializes Restic repositories on first run

All scripts source `/config/config.yaml` via Python for YAML parsing.
