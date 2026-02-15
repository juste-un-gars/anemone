# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

---

## QUICK REFERENCE - READ FIRST

**Critical rules that apply to EVERY session:**

1. **Incremental only** → Max 150 lines per iteration, STOP for validation
2. **No hardcoding** → No secrets, paths, credentials in code (use .env)
3. **Logs from day 1** → Configurable via LOG_LEVEL, LOG_TO_FILE, LOG_PATH
4. **Security audit** → MANDATORY before "project complete" status
5. **Stop points** → Wait for "OK"/"validated" after each module

**If unsure, read the relevant section below.**

---

**Key Documentation Files:**
- **[CLAUDE.md](CLAUDE.md)** - Project overview, philosophy, session management
- **[SESSION_STATE.md](SESSION_STATE.md)** - Current session status and recent work
- **[.claude/REFERENCE.md](.claude/REFERENCE.md)** - Quick reference: URLs, credentials, cmdlets
- **[README.md](README.md)** - Installation and setup guide

---

## Project Context

**Project Name:** Anemone v2
**Current Version:** v0.21.0-beta
**Tech Stack:** Go 1.21+, SQLite (CGO), Samba, HTML + Tailwind CSS
**Primary Language(s):** Go
**Key Dependencies:** CGO (for SQLite), Samba (SMB file sharing), Btrfs (optional, for quotas), Docker (OnlyOffice), rclone (cloud backup), WireGuard (VPN)
**Architecture Pattern:** Monolith with internal packages
**Development Environment:** Linux (Fedora/RHEL, Debian/Ubuntu)

---

## Development Philosophy

### Golden Rule: Incremental Development

**NEVER write large amounts of code without validation.**

```
One module → Test → User validates → Next module
```

**Per iteration limits:**
- 1-3 related files maximum
- ~50-150 lines of new code
- Must be independently testable

### Mandatory Stop Points

Claude MUST stop and wait for user validation after:
- Database connection/schema changes
- Authentication/authorization code
- Each API endpoint or route group
- File system or external service integrations
- Any security-sensitive code

**Stop format:**
```
✅ [Module] complete.

**Test it:**
1. [Step 1]
2. [Step 2]
Expected: [Result]

Waiting for your validation before continuing.
```

### Code Hygiene Rules (MANDATORY)

**Goal: Application must be portable and deployable anywhere without code changes.**

**NEVER hardcode in source files:**
- Passwords, API keys, tokens, secrets
- Database credentials or connection strings
- Absolute paths (`/home/user/...`, `/root/...`)
- IP addresses, hostnames, ports (production)
- Email addresses, usernames for services
- Environment-specific URLs (dev, staging, prod)

**ALWAYS use instead:**
- Environment variables (`.env` files, never committed)
- Configuration files (with `.example` templates)
- Relative paths or configurable base paths (`cfg.DataDir`, `cfg.IncomingDir`, etc.)
- Secret managers for production (Vault, etc.)

**Portability Checklist:**
- [ ] App starts with only environment configuration (no code edits)
- [ ] All paths relative or from config (`ANEMONE_DATA_DIR`)
- [ ] Database path from config
- [ ] External service URLs from config
- [ ] Port configurable (`ANEMONE_PORT`)

**Config Pattern (Go):**
```go
// internal/config/config.go
type Config struct {
    DataDir      string // from ANEMONE_DATA_DIR or "/srv/anemone"
    SharesDir    string // from ANEMONE_SHARES_DIR or DataDir/shares
    IncomingDir  string // from ANEMONE_INCOMING_DIR or DataDir/backups/incoming
    Port         string // from PORT or "8080"
    HTTPSPort    string // from HTTPS_PORT or "8443"
    LogLevel     string // from ANEMONE_LOG_LEVEL or "warn"
    LogDir       string // from ANEMONE_LOG_DIR or DataDir/logs
    // OnlyOffice
    OnlyOfficeEnabled bool   // from ANEMONE_OO_ENABLED
    OnlyOfficeURL     string // from ANEMONE_OO_URL
    OnlyOfficeSecret  string // from ANEMONE_OO_SECRET
}
```

### Logging Standards

**Goal: Comprehensive, configurable logging for debugging, auditing, and monitoring.**

**MUST configure logging infrastructure in Stage 1 (Foundation)** before any other code.

**Log Levels (configurable via `ANEMONE_LOG_LEVEL` env var):**
```
DEBUG   → Everything (dev only, verbose)
INFO    → Normal operations (API calls, user actions)
WARN    → Suspicious behavior (rate limit hits, deprecated usage)
ERROR   → Handled errors (connection failures, validation errors)
FATAL   → Unrecoverable errors (app crash)
```

**Environment Variables (in `.env.example`):**
```env
ANEMONE_LOG_LEVEL=info       # debug|info|warn|error (default: warn, overrides DB setting)
ANEMONE_LOG_DIR=/srv/anemone/logs  # Log file directory (default: $DATA_DIR/logs)
```

**What to Log:**
- API requests (route, method, status code, response time)
- Auth events (login, logout, failed attempts)
- Database operations (if DEBUG level)
- File operations (create, read, update, delete)
- External service calls (API calls, webhooks)
- Errors with stack traces
- Security events (injection attempts, unauthorized access)

**NEVER Log (Security Critical):**
- Passwords, tokens, API keys, secrets
- Credit card numbers, SSNs, personal IDs
- Session tokens, JWTs (log hash/ID only)
- Full request/response bodies if they contain sensitive data
- Database connection strings with credentials

**Logger Implementation (internal/logger/):**

The logger uses Go's `log/slog` with dual output (stdout + file), daily rotation, and runtime level changes:

```go
// Usage in code — always use slog key-value format, NOT printf-style
slog.Info("User created", "username", username, "admin", isAdmin)
slog.Error("Failed to sync", "peer", peerName, "error", err)
// WRONG: slog.Info("User %s created", username)  // causes !BADKEY
```

**Configuration:**
- Log level configurable from web UI (Admin → System Logs) or `ANEMONE_LOG_LEVEL` env var
- Daily log rotation with 30-day retention and 200 MB max total size
- Log files: `$DATA_DIR/logs/anemone-YYYY-MM-DD.log`
- Default level: WARN (reduces noise in production)

### Development Order (Enforce)

1. **Foundation first** — Config, DB, Auth, **Logging**
2. **Test foundation** — Don't continue if broken
3. **Core features** — One by one, tested
4. **Advanced features** — Only after core works

### File Size Guidelines

**Target sizes (lines of code):**
- **< 300** : ideal
- **300-500** : acceptable
- **500-800** : consider splitting
- **> 800** : must split

**When to split a file:**
- Multiple unrelated concerns in the same file
- Hard to find functions/methods
- File has too many responsibilities

**Naming convention for split files:**
```
app.go           → Core struct, New(), Run(), Shutdown()
app_jobs.go      → Job-related methods
app_sync.go      → Sync-related methods
app_settings.go  → Config/settings methods
```

---

## Build Commands

```bash
# Build main binary
go build -o anemone cmd/anemone/main.go

# Build dfree helper (for Samba quota display)
go build -o anemone-dfree cmd/anemone-dfree/main.go

# Run tests
go test ./...

# Run with verbose tests
go test -v ./...

# Run specific package tests
go test ./internal/sync/...

# Start development server
ANEMONE_DATA_DIR=/srv/anemone ./anemone

# Restart production service
sudo systemctl restart anemone
sudo systemctl reload smbd
```

---

## Project Structure

```
~/anemone/                       # Source code
├── cmd/
│   ├── anemone/main.go          # Main entry point
│   ├── anemone-dfree/main.go    # Samba dfree helper
│   ├── anemone-decrypt/         # Decrypt backup files (CLI tool)
│   ├── anemone-decrypt-password/ # Decrypt peer passwords (CLI tool)
│   ├── anemone-encrypt-peer-password/ # Encrypt peer passwords (CLI tool)
│   ├── anemone-migrate/         # Database migration tool
│   ├── anemone-reencrypt-key/   # Re-encrypt user keys with new master key
│   ├── anemone-restore-decrypt/ # Decrypt and restore files from backups
│   └── anemone-smbgen/          # Generate Samba configuration
├── internal/
│   ├── activation/              # User activation tokens
│   ├── adminverify/             # Admin verification (rate limiting)
│   ├── auth/                    # Authentication middleware & sessions
│   ├── backup/                  # User file backup to peers
│   ├── btrfs/                   # Btrfs quota utilities
│   ├── bulkrestore/             # Bulk file restoration
│   ├── config/                  # Configuration management
│   ├── crypto/                  # Encryption utilities (AES-256-GCM)
│   ├── database/                # SQLite + migrations
│   ├── i18n/                    # Internationalization (FR/EN)
│   │   └── locales/             # JSON translation files
│   ├── incoming/                # Incoming backups management
│   ├── logger/                  # Structured logging with rotation
│   ├── onlyoffice/              # OnlyOffice Docker integration
│   ├── peers/                   # P2P peers management
│   ├── quota/                   # Quota enforcement (Btrfs)
│   ├── rclone/                  # Cloud backup (SFTP, S3, WebDAV, named remotes)
│   ├── reset/                   # Password reset tokens
│   ├── restore/                 # File restoration from backups
│   ├── scheduler/               # Automatic sync scheduler
│   ├── serverbackup/            # Complete server backup/restore
│   ├── setup/                   # Setup wizard logic
│   ├── shares/                  # SMB share management
│   ├── smb/                     # Samba configuration
│   ├── storage/                 # ZFS pool & disk management
│   ├── sync/                    # P2P synchronization (incremental + manifest)
│   ├── syncauth/                # Sync authentication
│   ├── syncconfig/              # Sync scheduler configuration
│   ├── sysconfig/               # System-wide settings (DB-backed)
│   ├── tls/                     # TLS certificate generation
│   ├── trash/                   # Trash management & scheduler
│   ├── updater/                 # Update notification system
│   ├── usbbackup/               # USB backup with scheduling
│   ├── usermanifest/            # User share manifest generation
│   ├── users/                   # User management & auth
│   ├── web/                     # HTTP handlers (router, all handlers)
│   └── wireguard/               # WireGuard VPN client
├── web/
│   ├── static/                  # CSS, JS, images
│   └── templates/               # HTML templates (Go templates, dark theme v2)
├── docs/                        # Documentation
├── scripts/                     # Installation scripts
└── install.sh                   # Automated installation

/srv/anemone/                    # Production data directory
├── db/anemone.db               # SQLite database
├── shares/                     # User shares (per-user backup/ + data/)
├── backups/incoming/            # Backups from remote peers
├── certs/                      # TLS certificates + rclone SSH keys
├── logs/                       # Log files (daily rotation)
└── smb/smb.conf                # Generated Samba config
```

---

## Session Management

### Quick Start

**Continue work:** `"continue"` or `"let's continue"`
**New session:** `"new session: Feature Name"`

### File Structure

- **SESSION_STATE.md** (root) — Overview and session index
- **.claude/sessions/SESSION_XXX_[name].md** — Detailed session logs

**Naming:** `SESSION_001_project_setup.md`

### SESSION_STATE.md Header (Required)

SESSION_STATE.md **must** start with this reminder block:

```markdown
# Anemone - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur
```

### Session Template

```markdown
# Session XXX: [Feature Name]

## Meta
- **Date:** YYYY-MM-DD
- **Goal:** [Brief description]
- **Status:** In Progress / Blocked / Complete

## Current Module
**Working on:** [Module name]
**Progress:** [Status]

## Module Checklist
- [ ] Module planned (files, dependencies, test procedure)
- [ ] Code written
- [ ] Self-tested by Claude
- [ ] User validated ← **REQUIRED before next module**

## Completed Modules
| Module | Validated | Date |
|--------|-----------|------|
| Example | ✅ | YYYY-MM-DD |

## Next Modules (Prioritized)
1. [ ] [Next module]
2. [ ] [Following module]

## Technical Decisions
- **[Decision]:** [Reason]

## Issues & Solutions
- **[Issue]:** [Solution]

## Files Modified
- `path/file.ext` — [What/Why]

## Handoff Notes
[Critical context for next session]
```

### Session Rules

**MUST DO:**
1. Read CLAUDE.md and current session first
2. Update session file in real-time
3. Wait for validation after each module
4. Fix bugs before new features

**NEW SESSION when:**
- New major feature/module
- Current session goal complete
- Different project area

---

## Module Workflow

### 1. Plan (Before Coding)

```markdown
📋 **Module:** [Name]
📝 **Purpose:** [One sentence]
📁 **Files:** [List]
🔗 **Depends on:** [Previous modules]
🧪 **Test procedure:** [How to verify]
🔒 **Security concerns:** [If any]
```

### 2. Implement

- Write minimal working code
- Include error handling
- Document as you go (headers, comments)

### 3. Validate

**Functional:**
- [ ] Runs without errors
- [ ] Expected output verified
- [ ] Errors handled gracefully

**Security (if applicable):**
- [ ] Input validated
- [ ] No hardcoded secrets, paths, or credentials
- [ ] Parameterized queries (SQL)
- [ ] Output encoded (XSS)

### 4. User Confirmation

**DO NOT proceed until user says "OK", "validated", or "continue"**

---

## Documentation Standards

### Go File Header (Recommended)
```go
// Package name provides [brief description].
//
// This file handles [specific purpose].
```

### Function Documentation (Go)
```go
// FunctionName does something specific.
//
// Parameters:
//   - param1: description
//   - param2: description
//
// Returns the result or error if [condition].
func FunctionName(param1 Type, param2 Type) (Result, error) {
```

### HTML Templates
Templates are in `web/templates/` using Go's `html/template` syntax.

### Translations
Add new strings to both `internal/i18n/locales/fr.json` and `en.json`.

---

## Pre-Launch Security Audit

### When to Run

**MANDATORY before any deployment or "project complete" status.**

Plan this phase from the start — it's not optional.

### Security Audit Checklist

#### 1. Code Review (Full Scan)
- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No hardcoded paths (use configurable: `cfg.DataDir`, `cfg.IncomingDir`)
- [ ] No hardcoded credentials or connection strings
- [ ] No sensitive data in logs
- [ ] All user inputs validated and sanitized
- [ ] No debug/dev code left in production

#### 2. OWASP Top 10 Check
- [ ] **Injection** — SQL parameterized, OS command injection protected
- [ ] **Broken Auth** — Strong passwords, session management
- [ ] **Sensitive Data Exposure** — Encryption at rest and in transit (HTTPS)
- [ ] **XXE** — XML parsing secured (if applicable)
- [ ] **Broken Access Control** — Authorization verified on all routes
- [ ] **Security Misconfiguration** — Default credentials removed, error messages generic
- [ ] **XSS** — Output encoding in templates
- [ ] **Insecure Deserialization** — Untrusted data not deserialized
- [ ] **Vulnerable Components** — Dependencies updated, no known CVEs
- [ ] **Insufficient Logging** — Security events logged, logs protected

#### 3. Dependency Audit
```bash
# Go
go list -m -u all          # Check for updates
govulncheck ./...          # Check for vulnerabilities
```
- [ ] All critical/high vulnerabilities addressed
- [ ] Outdated packages updated or justified

#### 4. Online Vulnerability Research
- [ ] Search CVE databases for stack components
- [ ] Check GitHub security advisories for dependencies
- [ ] Review recent security news for frameworks used

**Resources:**
- https://cve.mitre.org
- https://nvd.nist.gov
- https://github.com/advisories
- https://pkg.go.dev/vuln

#### 5. Logging Security
- [ ] No passwords, tokens, API keys, secrets in logs
- [ ] No credit card numbers, SSNs, personal IDs in logs
- [ ] Session tokens not logged (only hash/ID if needed)
- [ ] Full request/response bodies not logged if sensitive
- [ ] Log files not publicly accessible (outside webroot)
- [ ] Log rotation configured (max size/count)
- [ ] Production uses INFO or WARN level (not DEBUG)
- [ ] File permissions restrict log access (chmod 640 or stricter)
- [ ] Security events are logged (auth failures, injection attempts)

#### 6. Configuration Security
- [ ] HTTPS enforced
- [ ] Security headers present (HSTS, CSP, X-Frame-Options, etc.)
- [ ] Cookies secured (HttpOnly, Secure, SameSite)
- [ ] Error pages don't leak stack traces
- [ ] Admin interfaces protected

### Post-Audit Actions

1. **Critical/High issues** → Fix immediately, re-test
2. **Medium issues** → Fix before launch or document accepted risk
3. **Low issues** → Add to backlog
4. **Re-run audit** after fixes

---

## Git Integration

### Branch Naming
**Format:** `feature/brief-description` or `bugfix/brief-description`
**Examples:** `feature/user-auth`, `bugfix/memory-leak`

### Commit Messages
```
type: Brief summary

Details if needed.

Changes:
- Change 1
- Change 2
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`

---

## Git Best Practices & .gitignore

### Critical Rules (MANDATORY)

**NEVER commit to repository:**
- Secrets, credentials, API keys, tokens
- `.env` files (commit `.env.example` only)
- Database files (SQLite .db, etc.)
- Log files, debug outputs
- IDE/editor configurations (user-specific)
- Build artifacts, compiled binaries
- Dependency directories (vendor/)
- OS-specific files (.DS_Store, Thumbs.db)
- Temporary files, caches

**ALWAYS commit:**
- `.env.example` (template with placeholders)
- `.gitignore` (comprehensive)
- README.md, CLAUDE.md
- Source code, configuration templates
- Database migrations/schemas (NOT the actual data)
- Documentation, LICENSE

### Go Project .gitignore Template

```gitignore
# === SECRETS & CREDENTIALS (CRITICAL) ===
.env
.env.local
.env.*.local
*.key
*.pem
secrets/
credentials/

# === LOGS ===
logs/
*.log
*.log.*

# === DATABASES ===
*.db
*.sqlite
*.sqlite3
*.db-shm
*.db-wal
data/

# === GO BUILD ===
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
vendor/
go.work
go.work.sum

# === IDE & EDITORS ===
.vscode/
.idea/
*.swp
*.swo
*~

# === OS FILES ===
.DS_Store
Thumbs.db
Desktop.ini

# === TEMPORARY & CACHE ===
tmp/
temp/
*.tmp
*.bak
*.cache
.cache/
```

### Pre-Commit Checklist

**Before every commit, verify:**
- [ ] No `.env` or secrets in staged files
- [ ] No absolute paths in code
- [ ] No hardcoded credentials or API keys
- [ ] No temporary debug code
- [ ] `.gitignore` is up to date
- [ ] Commit message follows convention

### Emergency: Committed Secrets by Mistake

**If you accidentally committed secrets:**

```bash
# 1. Remove from last commit (not pushed yet)
git rm --cached .env
git commit --amend --no-edit

# 2. If already pushed (CRITICAL - ACT IMMEDIATELY)
# a) Rotate/revoke ALL exposed credentials NOW
# b) Remove from history (force push required)
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch .env' \
  --prune-empty --tag-name-filter cat -- --all
git push --force --all
```

**IMPORTANT:** Removing from Git history is not enough. Secrets must be:
1. **Immediately revoked/rotated** (API keys, passwords, tokens)
2. **Reported** if company/team credentials
3. **Monitored** for unauthorized use

---

## Quick Commands

| Command | Action |
|---------|--------|
| `continue` | Resume current session |
| `new session: [name]` | Start new session |
| `save progress` | Update session file |
| `validate` | Mark current module as validated |
| `show plan` | Display remaining modules |
| `security audit` | Run full pre-launch security checklist |
| `dependency check` | Audit dependencies for vulnerabilities |

### Build & Test
```bash
go build -o anemone cmd/anemone/main.go
go test ./...
sudo systemctl restart anemone
```

---

## File Standards

- **Encoding:** UTF-8 with LF line endings
- **Timestamps:** ISO 8601 (YYYY-MM-DD HH:mm)
- **Time format:** 24-hour

---

## Additional Resources

- **README.md** - Full installation and usage guide
- **docs/installation.md** - Detailed installation options
- **docs/user-guide.md** - User guide (file browser, OnlyOffice, backups, etc.)
- **docs/rclone-backup.md** - Cloud backup (SFTP, S3, WebDAV, named remotes)
- **docs/usb-backup.md** - USB backup guide
- **docs/storage-setup.md** - ZFS/RAID storage configuration
- **docs/advanced.md** - Environment variables, reverse proxy, TLS
- **docs/API.md** - REST API reference (70+ endpoints)
- **docs/troubleshooting.md** - Diagnostics, DB queries, maintenance
- **docs/security.md** - Encryption, keys, architecture
- **docs/i18n.md** - Translation guide
- **CHANGELOG.md** - Version history

---

**Last Updated:** 2026-02-15
**Version:** 3.3.0
