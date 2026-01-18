# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Key Documentation Files:**
- **[CLAUDE.md](CLAUDE.md)** - Project overview, objectives, architecture, session management
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

## File Encoding Standards

- **All files:** UTF-8 with LF (Unix) line endings
- **Timestamps:** ISO 8601 (YYYY-MM-DD HH:mm)
- **Time format:** 24-hour (HH:mm)

---

## Claude Code Session Management

### Quick Start (TL;DR)

**Continue work:** `"continue"` or `"let's continue"`
**New session:** `"new session: Feature Name"` or `"start new session"`

**Claude handles everything automatically** - no need to manage session numbers or files manually.

---

### Session File Structure

**Two-Tier System:**
1. **SESSION_STATE.md** (root) - Overview and index of all sessions
2. **.claude/sessions/SESSION_XXX_[name].md** - Detailed session files

**Naming:** `SESSION_001_project_setup.md` (three digits, 001-999)

**Session Limits (Recommendations):**
- Max tasks: 20-25 per session
- Max files modified: 15-20 per session
- Recommended duration: 2-4 hours

---

### Automatic Session Workflow

#### 1. Session Start
- Read CLAUDE.md, SESSION_STATE.md, current session file
- Display status and next tasks

#### 2. During Development (AUTO-UPDATE)
**Individual Session File:**
- Mark completed tasks immediately
- Log technical decisions and issues in real-time
- Track all modified files
- Document all code created

**SESSION_STATE.md:**
- Update timestamp and session reference
- Update current status
- Add to recent sessions summary

#### 3. Session File Template

```markdown
# Session XXX: [Feature Name]

## Date: YYYY-MM-DD
## Duration: [Start - Current]
## Goal: [Brief description]

## Completed Tasks
- [x] Task 1 (HH:mm)
- [ ] Task 2 - In progress

## Current Status
**Currently working on:** [Task]
**Progress:** [Status]

## Next Steps
1. [ ] Next immediate task
2. [ ] Following task

## Technical Decisions
- **Decision:** [What]
  - **Reason:** [Why]
  - **Trade-offs:** [Pros/cons]

## Issues & Solutions
- **Issue:** [Problem]
  - **Solution:** [Resolution]
  - **Root cause:** [Why]

## Files Modified
### Created
- path/file.go - [Description]
### Updated
- path/file.go - [Changes]

## Dependencies Added
- package@version - [Reason]

## Testing Notes
- [ ] Tests written/passing
- **Coverage:** [%]

## Session Summary
[Paragraph summarizing accomplishments]

## Handoff Notes
- **Critical context:** [Must-know info]
- **Blockers:** [If any]
- **Next steps:** [Recommendations]
```

---

### Session Management Rules

#### MANDATORY Actions:
1. Always read CLAUDE.md first for context
2. Always read current session file
3. Update session in real-time as tasks complete
4. Never lose context between messages
5. Auto-save progress every 10-15 minutes

#### When to Create New Session:
- New major feature/module
- Completed session goal
- Different project area
- After long break
- Approaching session limits

---

### Common Commands

**Continue:** "continue", "let's continue", "keep going"
**New session:** "new session: [name]", "start new session"
**Save:** "save progress", "checkpoint"
**Update:** "update session", "update SESSION_STATE.md"

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

## Git Workflow Integration

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

## Quick Reference Commands

### Starting
```bash
"continue"                    # Continue existing work
"new session: Feature Name"   # Start new session
```

### During Work
```bash
"save progress"               # Save current state
"update session"              # Update session file
```

### Build & Test
```bash
go build -o anemone cmd/anemone/main.go
go test ./...
sudo systemctl restart anemone
```

---

## Additional Resources

- **README.md** - Full installation and usage guide
- **internal/i18n/locales/README.md** - Translation guide
- **docs/STORAGE_SETUP.md** - RAID/storage configuration

---

**Last Updated:** 2025-01-18
**Version:** 1.0.0
