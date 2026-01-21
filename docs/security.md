# Security

## End-to-End Encryption

Backups are encrypted **before leaving the source server**.

```
Original file → AES-256-GCM encryption → HTTPS transfer → Encrypted storage
```

- Destination server cannot read the files
- Only the user with their key can decrypt
- Even if peer is compromised, data remains protected

## Key Architecture

### Master Key

- Generated at initial setup
- Stored in `system_config`
- Used to encrypt user keys
- Never leaves the server

### User Keys

- Generated at account activation (32 random bytes)
- Displayed **once only** to user
- Stored encrypted with master key (`encryption_key_encrypted`)
- Hash stored for verification (`encryption_key_hash`)

```
User key → Encrypted with Master Key → Stored in DB
```

### Lost User Key

**Without the key, backups are unrecoverable.**

Recommendations:
- Download the key during activation
- Store in a password manager
- Keep a secure paper copy

## Encryption Format

**AES-256-GCM** (Galois/Counter Mode)

```
[Nonce 12 bytes][Encrypted data][Auth Tag 16 bytes]
```

- **Nonce**: Unique per file, randomly generated
- **Auth Tag**: Ensures integrity (detects modifications)
- Encrypted file extension: `.enc`

## Authentication

### Sessions

- Stored in database (persistent)
- Duration: 2 hours (standard) or 30 days ("Remember me")
- Cookie `HttpOnly`, `Secure`, `SameSite=Strict`
- Renewed on each request

### Passwords

- Hashed with bcrypt (cost 10)
- Never stored in plain text
- Minimum 8 characters required

### P2P Authentication

- Optional password per server
- Transmitted via `X-Sync-Password` header
- Hashed with bcrypt server-side

## HTTPS

### Self-Signed Certificate

By default, Anemone generates a self-signed certificate.

- Browser warning is normal
- Secure for local/private use
- Recommended: add exception in browser

### Custom Certificate

```bash
# Environment variables
TLS_CERT_PATH=/path/to/cert.pem
TLS_KEY_PATH=/path/to/key.pem
```

### Security Headers

- `Strict-Transport-Security` (HSTS)
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

## Data Isolation

### Between Users

- Each user sees only their own shares
- Unique encryption keys per user
- Cannot decrypt another user's data

### Between Peers

- Backups organized by source server
- Peer cannot access user keys
- Files remain encrypted on peer

## Server Backup

### Configuration Export

Server configuration can be exported (encrypted) for disaster recovery.

Contains:
- System configuration
- User list (without passwords)
- Peer configuration

**Does not contain**:
- Master key (backup separately)
- User keys (each user keeps their own)

### Restore

1. Install Anemone on new server
2. Choose "Restore from backup"
3. Import configuration file
4. Users must reactivate their accounts

## Best Practices

### Administrator

- [ ] Backup master key separately
- [ ] Use HTTPS (enabled by default)
- [ ] Set a sync password
- [ ] Keep Anemone updated
- [ ] Check logs regularly

### User

- [ ] Backup encryption key
- [ ] Use strong password (>12 characters)
- [ ] Enable "Remember me" only on personal devices
- [ ] Verify backups regularly

## See Also

- [P2P Sync](p2p-sync.md) - Peer configuration
- [Advanced Configuration](advanced.md) - Custom certificates
- [API](API.md) - API authentication
