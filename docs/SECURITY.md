# Security Documentation - Anemone

## Overview

This document describes known security considerations, risks, and mitigations for the Anemone NAS system.

---

## TLS Certificate Verification (InsecureSkipVerify)

### Current Behavior

Anemone uses self-signed TLS certificates for peer-to-peer communication. By default, the system skips TLS certificate verification when connecting to peers:

```go
TLSClientConfig: &tls.Config{InsecureSkipVerify: true}
```

### Affected Components

The following files/functions use `InsecureSkipVerify: true`:

| File | Function | Purpose |
|------|----------|---------|
| `internal/web/router.go` | Multiple handlers | Peer connectivity tests |
| `internal/peers/peers.go` | `TestConnection()` | Peer health checks |
| `internal/sync/sync.go` | `SyncShareIncremental()` | Sync file transfers |
| `internal/bulkrestore/bulkrestore.go` | `DownloadEncryptedFile()` | Restore operations |

### Security Risk: Man-in-the-Middle (MITM)

**Risk Level:** Medium (network-dependent)

Without certificate verification, an attacker on the network path between two Anemone peers could:

1. **Intercept sync traffic** - Though files are encrypted with AES-256-GCM, metadata (file names, sizes) could be exposed
2. **Impersonate a peer** - Present a fake certificate and receive backup data
3. **Modify data in transit** - Though less likely with encrypted payloads

### Why Self-Signed Certificates?

Anemone uses self-signed certificates because:

1. **No external dependencies** - No need for Let's Encrypt, public domains, or external CAs
2. **Fully local operation** - Works on LANs without internet access
3. **Simplified setup** - Users don't need to manage CA infrastructure
4. **Peer-to-peer model** - Traditional PKI doesn't fit the decentralized model

### Risk Mitigation

The following factors reduce the practical risk:

1. **Encrypted payloads** - All file data is encrypted with AES-256-GCM before transmission
2. **Password authentication** - The `X-Sync-Password` header is required for all sync operations
3. **Source server validation** - The `X-Source-Server` header prevents peer impersonation
4. **LAN deployment** - Most Anemone deployments are on trusted local networks

### Recommended Deployment Practices

To minimize MITM risk:

1. **Use a trusted network** - Deploy Anemone peers on a dedicated VLAN or trusted LAN
2. **Enable firewall rules** - Restrict sync ports (8443) to known peer IPs
3. **Monitor network traffic** - Use IDS/IPS to detect suspicious activity
4. **Consider VPN** - For WAN sync, use a VPN tunnel between sites

### Future Improvements (Not Yet Implemented)

Potential future enhancements:

1. **Certificate pinning** - Store peer certificate fingerprints and verify on connection
2. **Optional CA mode** - Allow users to provide a custom CA for verification
3. **TOFU (Trust On First Use)** - Accept certificate on first connection, warn on change

---

## Authentication Security

### Sync API Authentication

The sync API uses two authentication headers:

| Header | Purpose | Validation |
|--------|---------|------------|
| `X-Sync-Password` | Shared secret between peers | Bcrypt hash comparison |
| `X-Source-Server` | Peer identity for authorization | Must match `source_server` parameter |

### Protection Against Common Attacks

| Attack | Protection |
|--------|------------|
| Brute force login | Rate limiting (5 attempts / 15 min per IP) |
| Brute force password reset | Rate limiting (3 attempts / 30 min per IP) |
| SQL injection | Parameterized queries throughout |
| XSS | Template escaping, Content-Security-Policy |
| CSRF | SameSite=Strict cookies |
| Path traversal | `isPathTraversal()` validation on all paths |

### Master Key Storage

The master encryption key is stored in the SQLite database (`system_config` table).

**Current risk:** If an attacker gains access to the database file, they can extract the master key.

**Mitigations:**
- File permissions (600) on database file
- Consider environment variable storage for higher security deployments
- TPM integration for hardware key storage (future consideration)

---

## Rate Limiting

### Protected Endpoints

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/login` (POST) | 5 requests | 15 minutes |
| `/reset-password` (POST) | 3 requests | 30 minutes |

### Implementation

- Sliding window algorithm
- Per-IP tracking (supports X-Forwarded-For)
- Automatic cleanup of old entries (every 5 minutes)
- Returns HTTP 429 with `Retry-After` header when exceeded

---

## Encryption

### Data at Rest

- Algorithm: AES-256-GCM
- Key derivation: Per-user encryption keys derived from master key
- Coverage: All synced files, manifests, and sensitive metadata

### Data in Transit

- Protocol: HTTPS (TLS 1.2+)
- Certificates: Self-signed (see TLS section above)
- Additional: File content encrypted before transmission

---

## Security Headers

All HTTP responses include:

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; ...
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

---

## Reporting Security Issues

If you discover a security vulnerability in Anemone, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Contact the maintainers directly
3. Allow time for a fix before public disclosure

---

## Changelog

| Date | Change |
|------|--------|
| 2025-01-18 | Added rate limiting documentation |
| 2025-01-18 | Added X-Source-Server authorization |
| 2025-01-18 | Initial security documentation |
