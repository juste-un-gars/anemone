# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

---

## Project Context

**Project Name:** [To be filled]  
**Tech Stack:** [To be filled]  
**Primary Language(s):** [To be filled]  
**Key Dependencies:** [To be filled]  
**Architecture Pattern:** [To be filled]

---

## ⚠️ Development Philosophy

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
- ❌ Passwords, API keys, tokens, secrets
- ❌ Database credentials or connection strings
- ❌ Absolute paths (`C:\Users\...`, `/home/user/...`)
- ❌ IP addresses, hostnames, ports (production)
- ❌ Email addresses, usernames for services
- ❌ Environment-specific URLs (dev, staging, prod)

**ALWAYS use instead:**
- ✅ Environment variables (`.env` files, never committed)
- ✅ Configuration files (with `.example` templates)
- ✅ Relative paths or configurable base paths
- ✅ Secret managers for production (Vault, AWS Secrets, etc.)

**Project must include:**
```
├── .env.example          # Template with ALL variables, placeholder values
├── .gitignore            # Excludes .env, secrets, logs, build artifacts
├── config/               # Centralized configuration module
│   ├── index.js          # Loads from env vars with defaults
│   └── config.example.json  # Template if using JSON config
└── README.md             # Setup instructions with env vars list
```

**Portability Checklist:**
- [ ] App starts with only `.env` configuration (no code edits)
- [ ] All paths relative or from env vars (`DATA_DIR`, `LOG_PATH`)
- [ ] Database connection string from env (`DATABASE_URL`)
- [ ] External service URLs from env (`API_BASE_URL`, `SMTP_HOST`)
- [ ] Port configurable (`PORT=3000`)
- [ ] Works on Windows, Linux, macOS (if cross-platform)

**Config Module Pattern:**
```javascript
// config/index.js - Example pattern
module.exports = {
  port: process.env.PORT || 3000,
  db: {
    url: process.env.DATABASE_URL || 'sqlite://local.db',
  },
  dataDir: process.env.DATA_DIR || './data',
  logLevel: process.env.LOG_LEVEL || 'info',
};
```

### Development Order (Enforce)

1. **Foundation first** — Config, DB, Auth
2. **Test foundation** — Don't continue if broken
3. **Core features** — One by one, tested
4. **Advanced features** — Only after core works

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
# [Project] - Session State

> **Claude : Appliquer le protocole de session (CLAUDE.md)**
> - Créer/mettre à jour la session en temps réel
> - Valider après chaque module avec : ✅ [Module] complete. **Test it:** [...] Waiting for validation.
> - Ne pas continuer sans validation utilisateur
```

This ensures Claude applies the session protocol when the user asks to read SESSION_STATE.md.

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
| DB Connection | ✅ | YYYY-MM-DD |
| Auth | ✅ | YYYY-MM-DD |

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

**⚠️ DO NOT proceed until user says "OK", "validated", or "continue"**

---

## Build Order Templates

### Web Application

```
Stage 1: Foundation (validate before Stage 2)
├── [ ] Project structure + config module → starts without error
├── [ ] .env.example with all variables documented
├── [ ] Database connection (from env var) → can connect
├── [ ] Auth (register/login/logout) → full flow works
├── [ ] Session/JWT management → persists correctly
└── [ ] SECURITY REVIEW

Stage 2: Core (validate before Stage 3)
├── [ ] User profile CRUD
├── [ ] Basic API routes
└── [ ] Error handling middleware

Stage 3: Features
├── [ ] Feature A
├── [ ] Feature B
└── [ ] ...

Stage 4: Pre-Launch (MANDATORY)
├── [ ] Full security audit (see checklist)
├── [ ] Dependency audit (npm audit, etc.)
├── [ ] Penetration testing
├── [ ] Portability test (deploy on clean machine)
├── [ ] DEPLOYMENT.md written
├── [ ] All issues fixed or documented
└── [ ] Final validation
```

### API Service

```
Stage 1: Foundation
├── [ ] Config module + .env.example
├── [ ] Database + migrations (connection from env)
├── [ ] Auth middleware
└── [ ] Health check endpoint

Stage 2: Core Endpoints
├── [ ] Resource A (CRUD)
├── [ ] Resource B (CRUD)
└── [ ] Relationships

Stage 3: Advanced
├── [ ] Search/filtering
├── [ ] Pagination
└── [ ] Rate limiting

Stage 4: Pre-Launch (MANDATORY)
├── [ ] Full security audit
├── [ ] Dependency vulnerabilities checked
├── [ ] API penetration testing
├── [ ] Portability test (fresh environment)
├── [ ] DEPLOYMENT.md written
├── [ ] Rate limiting verified
└── [ ] Final validation
```

### DEPLOYMENT.md Template

```markdown
# Deployment Guide

## Requirements
- [Runtime] v[version]
- [Database] v[version]
- [Other dependencies]

## Environment Variables
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| PORT | No | 3000 | Server port |
| DATABASE_URL | Yes | - | Database connection string |
| ... | ... | ... | ... |

## Quick Start
1. Clone repository
2. Copy `.env.example` to `.env`
3. Edit `.env` with your values
4. Run `[install command]`
5. Run `[start command]`

## Production Deployment
[Platform-specific instructions]

## Troubleshooting
[Common issues and solutions]
```

---

## Documentation Standards

### File Header (Required)

```javascript
/**
 * @file filename.ext
 * @description Brief purpose
 * @created YYYY-MM-DD
 */
```

### Function Documentation (Required)

```javascript
/**
 * Brief description
 * @param {type} name - Description
 * @returns {type} Description
 */
```

### .EXPLAIN.md Files

Create for complex scripts/modules:

```markdown
# [Filename]

## Purpose
[What and why]

## Usage
[Code example]

## Key Functions
[List with brief descriptions]
```

---

## Pre-Launch Security Audit

### When to Run

**MANDATORY before any deployment or "project complete" status.**

Plan this phase from the start — it's not optional.

### Security Audit Checklist

#### 1. Code Review (Full Scan)
- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No hardcoded paths (use relative or configurable)
- [ ] No hardcoded credentials or connection strings
- [ ] No sensitive data in logs
- [ ] All user inputs validated and sanitized
- [ ] No debug/dev code left in production
- [ ] `.env.example` present with all required variables
- [ ] `.gitignore` excludes `.env` and sensitive files

#### 2. OWASP Top 10 Check
- [ ] **Injection** — SQL, NoSQL, OS command injection protected
- [ ] **Broken Auth** — Strong passwords, session management, MFA if needed
- [ ] **Sensitive Data Exposure** — Encryption at rest and in transit (HTTPS)
- [ ] **XXE** — XML parsing secured (if applicable)
- [ ] **Broken Access Control** — Authorization verified on all routes
- [ ] **Security Misconfiguration** — Default credentials removed, error messages generic
- [ ] **XSS** — Output encoding, CSP headers
- [ ] **Insecure Deserialization** — Untrusted data not deserialized
- [ ] **Vulnerable Components** — Dependencies updated, no known CVEs
- [ ] **Insufficient Logging** — Security events logged, logs protected

#### 3. Dependency Audit
```bash
# Run appropriate command for your stack:
npm audit                    # Node.js
pip-audit                    # Python
cargo audit                  # Rust
dotnet list package --vulnerable  # .NET
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
- https://snyk.io/vuln

#### 5. Basic Penetration Testing
- [ ] SQL injection attempts on all inputs
- [ ] XSS attempts on all outputs
- [ ] Auth bypass attempts (direct URL access, token manipulation)
- [ ] Rate limiting verified (brute force protection)
- [ ] File upload restrictions tested (if applicable)
- [ ] CORS policy verified

#### 6. Configuration Security
- [ ] HTTPS enforced
- [ ] Security headers present (HSTS, CSP, X-Frame-Options, etc.)
- [ ] Cookies secured (HttpOnly, Secure, SameSite)
- [ ] Error pages don't leak stack traces
- [ ] Admin interfaces protected/hidden

### Audit Report Template

```markdown
# Security Audit Report

**Project:** [Name]
**Date:** YYYY-MM-DD
**Audited by:** [Claude / Human / Both]

## Summary
- Critical issues: X
- High issues: X
- Medium issues: X
- Low issues: X

## Findings

### [CRITICAL/HIGH/MEDIUM/LOW] Issue Title
- **Location:** [File:line or endpoint]
- **Description:** [What's wrong]
- **Risk:** [Impact if exploited]
- **Fix:** [How to resolve]
- **Status:** [ ] Fixed / [ ] Accepted risk

## Dependency Audit Results
[Paste output]

## Checklist Completion
[Copy checklist with status]

## Conclusion
[ ] Ready for launch
[ ] Requires fixes before launch
```

### Post-Audit Actions

1. **Critical/High issues** → Fix immediately, re-test
2. **Medium issues** → Fix before launch or document accepted risk
3. **Low issues** → Add to backlog
4. **Re-run audit** after fixes

---

## Git Integration

### Branch Naming
`feature/session-XXX-brief-name`

### Commit Message
```
Session XXX: [Summary]

- Change 1
- Change 2
```

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

---

## File Standards

- **Encoding:** UTF-8 with LF line endings
- **Timestamps:** ISO 8601 (YYYY-MM-DD HH:mm)
- **Time format:** 24-hour

---

**Last Updated:** YYYY-MM-DD  
**Version:** 3.0.0
