# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**📚 Key Documentation Files:**
- **[CLAUDE.md](CLAUDE.md)** - Project overview, objectives, architecture, session management
- **[SESSION_STATE.md](SESSION_STATE.md)** - Current session status and recent work
- **[.claude/REFERENCE.md](.claude/REFERENCE.md)** - Quick reference: URLs, credentials, cmdlets
- **[README.md](README.md)** - Installation and setup guide

---

## Project Context

**Project Name:** [To be filled]  
**Tech Stack:** [To be filled]  
**Primary Language(s):** [To be filled]  
**Key Dependencies:** [To be filled]  
**Architecture Pattern:** [To be filled]  
**Development Environment:** [To be filled]

---

## File Encoding Standards

- **All files:** UTF-8 with LF (Unix) line endings
- **Timestamps:** ISO 8601 (YYYY-MM-DD HH:mm)
- **Time format:** 24-hour (HH:mm)

---

## Claude Code Session Management

### 🚀 Quick Start (TL;DR)

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
- path/file.js - [Description]
### Updated
- path/file.js - [Changes]

## Documentation Created/Updated
- [ ] [file].EXPLAIN.md - Created/Updated
- Files documented: X/Y (Z%)

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
4. Document all code (headers, functions, .EXPLAIN.md)
5. Never lose context between messages
6. Auto-save progress every 10-15 minutes
7. Verify documentation before marking tasks complete

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
**Document:** "document files", "create EXPLAIN files"  
**Audit:** "check documentation", "audit docs"

---

## Documentation Standards

### Overview
**Every code file MUST have complete documentation before task is marked complete.**

### Required Documentation Elements

#### 1. File Header (All Files)
```language
/**
 * @file filename.ext
 * @description Brief file purpose
 * @session SESSION_XXX
 * @created YYYY-MM-DD
 * @author [name/team]
 */
```

#### 2. Function Documentation
```javascript
/**
 * Brief function description
 * 
 * @param {type} paramName - Parameter description
 * @returns {type} Return description
 * @throws {Error} Error conditions
 * @example
 * functionName(arg) // => result
 * @session SESSION_XXX
 */
```

#### 3. .EXPLAIN.md Files (Scripts/Modules)
**Create for:** All scripts, complex modules, utilities

**Template:**
```markdown
# [Filename] - Explanation

## Purpose
[What this does]

## How It Works
[Step-by-step explanation]

## Usage
```language
[Code examples]
```

## Key Functions
### functionName()
[Description, params, returns]

## Error Handling
[Common issues and solutions]

## Session History
- SESSION_XXX: Created
```

---

### Documentation Checklist
Before marking any task complete, verify:
- [ ] File header present
- [ ] All functions documented (description, params, returns, examples)
- [ ] All classes documented (properties, methods, usage)
- [ ] Complex sections explained
- [ ] Inline comments for non-obvious logic
- [ ] .EXPLAIN.md created/updated (for scripts)
- [ ] Error cases documented

---

## Git Workflow Integration

### Branch Naming
**Format:** `feature/session-XXX-brief-description`  
**Examples:** `feature/session-025-user-auth`, `bugfix/session-027-memory-leak`

### Commit Messages
```
Session XXX: [Brief summary]

[Details]

Changes:
- Change 1
- Change 2

Documentation:
- Updated [file].EXPLAIN.md

Session: SESSION_XXX
```

### Tagging Completed Sessions
```bash
git tag -a session-XXX-complete -m "Session XXX: [Feature] - Complete"
git push origin session-XXX-complete
```

### Session Files in Git
Commit session files when:
- Starting new session
- Completing session
- End of work day
- Before switching branches

---

## Multi-Developer Guidelines

### Session Locking
```bash
# Start work
echo "Locked by: $(whoami) at $(date)" > .claude/sessions/.session-XXX.lock

# Finish work
rm .claude/sessions/.session-XXX.lock
```

### Before Creating New Session
1. `git pull origin main`
2. Read SESSION_STATE.md for current number
3. Check lock files: `ls .claude/sessions/.session-*.lock`
4. Create next sequential number

### Handoff Protocol
1. Update session with comprehensive notes
2. Commit and push
3. Tag developer in session file
4. Notify via team channel

---

## FAQ

### Session Management
**Q: Can I rename a session?**  
A: Yes, update all references in SESSION_STATE.md. Use `git mv`.

**Q: Delete old sessions?**  
A: Archive first in `.claude/sessions/archive/YYYY/`. Keep 3+ months.

**Q: Search sessions?**  
A: `grep -r "term" .claude/sessions/`

### Documentation
**Q: Document everything?**  
A: Yes. Mandatory for all code. Saves time long-term.

**Q: Experimenting?**  
A: Minimum: file header + function descriptions. Skip .EXPLAIN.md if temporary.

**Q: How detailed should .EXPLAIN.md be?**  
A: Detailed enough for unfamiliar developers. Include examples and common issues.

### Git
**Q: Commit session files?**  
A: Yes, always. Valuable project history.

**Q: Multiple sessions simultaneously?**  
A: No. Finish one before starting another. Use branches for context-switch.

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
"check documentation"         # Verify doc coverage
"update session"              # Update session file
```

### Documentation
```bash
"document all files in src/"  # Document directory
"create missing EXPLAIN files" # Generate .EXPLAIN.md
"audit documentation"         # Check completeness
```

---

## Additional Resources

- **Quick Start:** [.claude/QUICKSTART.md](.claude/QUICKSTART.md)
- **Reference:** [.claude/REFERENCE.md](.claude/REFERENCE.md)
- **Templates:** [.claude/templates/](.claude/templates/)
- **Examples:** [.claude/sessions/examples/](.claude/sessions/examples/)

---

**Last Updated:** YYYY-MM-DD  
**Version:** 2.0.0
