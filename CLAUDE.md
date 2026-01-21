# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

**Key Documentation Files:**
- **[CLAUDE.md](CLAUDE.md)** - Project overview, philosophy, session management
- **[SESSION_STATE.md](SESSION_STATE.md)** - Current session status and recent work
- **[.claude/REFERENCE.md](.claude/REFERENCE.md)** - Quick reference: URLs, credentials, cmdlets
- **[README.md](README.md)** - Installation and setup guide

---

## Project Context

**Project Name:** Anemone v2
**Tech Stack:** Go 1.21+, SQLite (CGO), Samba, HTML + Tailwind CSS
**Primary Language(s):** Go
**Key Dependencies:** CGO (for SQLite), Samba (SMB file sharing), Btrfs (optional, for quotas)
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

**NEVER hardcode in source files:**
- Passwords, API keys, tokens, secrets
- Database credentials
- Absolute paths (`/home/user/...`)
- IP addresses, server names (production)

**ALWAYS use instead:**
- Environment variables (`.env` files, never committed)
- Configuration files (with `.example` templates)
- Relative paths or configurable base paths (`cfg.DataDir`, `cfg.IncomingDir`, etc.)

### Development Order

1. **Foundation first** — Config, DB, Auth
2. **Test foundation** — Don't continue if broken
3. **Core features** — One by one, tested
4. **Advanced features** — Only after core works

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
│   └── anemone-dfree/main.go    # Samba dfree helper
├── internal/
│   ├── config/                  # Configuration management
│   ├── database/                # SQLite + migrations
│   ├── users/                   # User management & auth
│   ├── shares/                  # SMB share management
│   ├── peers/                   # P2P peers management
│   ├── smb/                     # Samba configuration
│   ├── sync/                    # P2P synchronization (incremental + manifest)
│   ├── syncauth/                # Sync authentication
│   ├── syncconfig/              # Sync scheduler configuration
│   ├── scheduler/               # Automatic sync scheduler
│   ├── incoming/                # Incoming backups management
│   ├── crypto/                  # Encryption utilities (AES-256-GCM)
│   ├── quota/                   # Quota enforcement (Btrfs)
│   ├── trash/                   # Trash management
│   ├── updater/                 # Update notification system
│   ├── i18n/                    # Internationalization (FR/EN)
│   │   └── locales/             # JSON translation files
│   └── web/                     # HTTP handlers
├── web/
│   ├── static/                  # CSS, JS, images
│   └── templates/               # HTML templates (Go templates)
├── scripts/                     # Installation scripts
└── install.sh                   # Automated installation

/srv/anemone/                    # Production data directory
├── db/anemone.db               # SQLite database
├── shares/                     # User shares
├── incoming/                   # Backups from remote peers
├── certs/                      # TLS certificates
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

### Security Audit Checklist

#### 1. Code Review (Full Scan)
- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No hardcoded paths (use configurable: `cfg.DataDir`, `cfg.IncomingDir`)
- [ ] No sensitive data in logs
- [ ] All user inputs validated and sanitized

#### 2. OWASP Top 10 Check
- [ ] **Injection** — SQL parameterized, OS command injection protected
- [ ] **Broken Auth** — Strong passwords, session management
- [ ] **Sensitive Data Exposure** — Encryption at rest and in transit (HTTPS)
- [ ] **Broken Access Control** — Authorization verified on all routes
- [ ] **Security Misconfiguration** — Default credentials removed
- [ ] **XSS** — Output encoding in templates
- [ ] **Vulnerable Components** — Dependencies updated

#### 3. Dependency Audit
```bash
# Go
go list -m -u all          # Check for updates
govulncheck ./...          # Check for vulnerabilities
```

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

## Quick Commands

| Command | Action |
|---------|--------|
| `continue` | Resume current session |
| `new session: [name]` | Start new session |
| `save progress` | Update session file |
| `validate` | Mark current module as validated |

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
- **internal/i18n/locales/README.md** - Translation guide
- **docs/storage-setup.md** - ZFS storage configuration

---

**Last Updated:** 2026-01-21
**Version:** 2.0.0
