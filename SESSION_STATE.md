# Anemone - Session State

**Current Version:** v0.9.23-beta
**Last Updated:** 2026-01-20

---

## Current Session

**Session 57** - Storage Management (Phase 2-3 - Full ZFS & Disk Operations)
- **Status:** Completed
- **Date:** 2026-01-20
- **Commits:** `f7682d8`, `11e5295`, `42acc84`

### Completed (Session 57)

#### Feature: Storage Management Phase 2 - Scrub & SMART Details
- [x] Added pool scrub functionality (`zpool scrub`)
- [x] Added SMART details modal with full attribute table
- [x] Fixed UNKNOWN health display for disks without SMART

#### Feature: Storage Management Phase 3 - Full ZFS & Disk Operations
- [x] **Security**: Password verification endpoint with rate limiting (5 attempts/minute)
- [x] Single-use tokens with 5-minute TTL for destructive operations
- [x] **ZFS Pool operations**: create, destroy, export, import, add vdev, replace disk
- [x] **ZFS Dataset operations**: create, delete, set properties (compression, quota, mountpoint)
- [x] **ZFS Snapshot operations**: create, list, delete, rollback, clone
- [x] **Disk formatting**: format ext4/xfs with labels, quick/full wipe
- [x] Input validation to prevent command injection
- [x] 77 new translations for FR and EN
- [x] Sudoers permissions for mkfs, wipefs, parted, dd

### Files Created (Session 57)
- `internal/adminverify/adminverify.go` - Password verification with rate limiting
- `internal/storage/zfs_pool.go` - ZFS pool operations
- `internal/storage/zfs_dataset.go` - Dataset operations
- `internal/storage/zfs_snapshot.go` - Snapshot operations
- `internal/storage/disk_format.go` - Disk formatting and wiping

### Files Modified (Session 57)
- `internal/web/handlers_admin_storage.go` - Added 25+ new API handlers
- `internal/web/router.go` - Added 18 new storage routes
- `web/templates/admin_storage.html` - Complete rewrite with tabbed UI
- `internal/i18n/locales/fr.json` - 77 new storage translations
- `internal/i18n/locales/en.json` - 77 new storage translations
- `install.sh` - Added sudoers for disk formatting commands

---

## Previous Session

**Session 56** - Storage Management (Phase 1 - Read-Only)
- **Status:** Completed
- **Date:** 2026-01-20
- **Commits:** `9fcf8ac`, `defd833`, `ee4ce60`

### Completed (Session 56)

#### Feature: Storage Management Page (Phase 1 - Read-Only)
- [x] Updated install.sh with smartmontools and ZFS utilities
- [x] Added sudoers permissions for smartctl, zpool, zfs
- [x] Created internal/storage/ package:
  - `storage.go` - Types and StorageOverview function
  - `lsblk.go` - List physical disks via lsblk
  - `smart.go` - SMART health monitoring via smartctl
  - `zfs.go` - ZFS pools status via zpool/zfs
- [x] Created /admin/storage page handler and API endpoint
- [x] Created admin_storage.html template with:
  - Overview cards (disk count, health, pools, capacity)
  - Physical disks table (SMART health, temp, power-on hours)
  - ZFS pools section (vdevs, capacity bars, scan status)
- [x] Added storage widget to admin dashboard
- [x] Added FR/EN translations

#### Bug Fixes
- [x] Fixed lsblk JSON parsing - size/rota returned as native types not strings
- [x] Added flexInt/flexBool types to handle both old and new lsblk versions

---

## Previous Session

**Session 55** - Bug Fixes (Speed, Empty Dirs, Datetime Parsing)
- **Status:** Completed
- **Date:** 2026-01-19
- **Commits:** `785a909`, `535cbbd`, `ca9c968`

### Completed (Session 55)

#### Bug Fix: Robust Datetime Parsing for SQLite (v0.9.23-beta)
- [x] SQLite driver returns RFC3339 format (`2006-01-02T15:04:05Z`), not expected `2006-01-02 15:04:05`
- [x] Added multi-format parsing function trying 10 different datetime formats
- [x] Dates and speed now display correctly in sync reports

#### Bug Fix: Sync Speed Display on Peers Page (v0.9.21-beta)
- [x] Added speed column to sync report table on `/admin/peers` page
- [x] Speed calculation was in `handlers_admin_sync.go` but missing from `handlers_admin_peers.go`
- [x] Added `CompletedAt` and `Speed` fields to `RecentSync` struct
- [x] Parse SQLite datetime strings using `sql.NullString`
- [x] Calculate speed from `bytes_synced / duration`
- [x] Display speed in MB/s, KB/s, or B/s (or "-" if unavailable)

#### Bug Fix: Empty Directories Not Cleaned After Sync Delete (v0.9.22-beta)
- [x] When files deleted on AN1, directories left empty on AN2 after sync
- [x] Added `cleanEmptyParentDirs()` function in `handlers_sync_api.go`
- [x] Called after file deletion to walk up and remove empty parent dirs
- [x] Stops at backup root directory or when directory is not empty

---

## Recent Sessions

| # | Name | Date | Status |
|---|------|------|--------|
| 57 | Storage Management (Phase 2-3) | 2026-01-20 | Completed |
| 56 | Storage Management (Phase 1) | 2026-01-20 | Completed |
| 55 | Bug Fixes (Speed, Empty Dirs, Datetime) | 2026-01-19 | Completed |
| 54 | Bug Fixes & Release Management | 2026-01-18 | Completed |
| 53 | Performance & Real-time Manifests | 2025-01-18 | Completed |
| 52 | Security Audit Phases 1-5 | 2025-01-18 | Completed |
| 51 | User Share Manifests | 2025-01-18 | Completed |
| 37-39 | Security Audit & Fixes | 2024-12 | Completed |
| 31-34 | Update System | 2024-11 | Completed |
| 27-30 | Restore Interface | 2024-11 | Completed |
| 26 | Internationalization FR/EN | 2024-11-20 | Completed |
| 20-24 | P2P Sync & Scheduler | 2024-11 | Completed |
| 17-19 | Trash & Quotas | 2024-11 | Completed |
| 12-16 | SMB Automation | 2024-11 | Completed |
| 8-11 | P2P Foundation | 2024-11 | Completed |
| 1-7 | Initial Setup & Auth | 2024-10 | Completed |

---

## Session Archives

All detailed session files are in `.claude/sessions/`:

- `SESSION_052_security_audit.md` - Current audit session
- `SESSION_051_user_manifests.md` - User manifests
- `SESSION_STATE_ARCHIVE.md` - Sessions 1-7
- `SESSION_STATE_ARCHIVE_SESSIONS_8_11.md` - P2P Foundation
- `SESSION_STATE_ARCHIVE_SESSIONS_12_16.md` - SMB Automation
- `SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md` - Trash & Quotas
- `SESSION_STATE_ARCHIVE_SESSIONS_20_24.md` - P2P Sync & Scheduler
- `SESSIONS_ARCHIVE.md` - Session 26 (i18n)
- `SESSION_STATE_ARCHIVE_SESSIONS_27_30.md` - Restore Interface
- `SESSION_STATE_ARCHIVE_31_32_33_34.md` - Update System
- `SESSION_STATE_ARCHIVE_SESSIONS_37_39.md` - Security Audit

---

## Quick Links

- **[CLAUDE.md](CLAUDE.md)** - Project context & guidelines
- **[.claude/REFERENCE.md](.claude/REFERENCE.md)** - Quick reference
- **[README.md](README.md)** - Installation guide
- **[CHANGELOG.md](CHANGELOG.md)** - Version history
- **[docs/SECURITY.md](docs/SECURITY.md)** - Security documentation
- **[docs/API.md](docs/API.md)** - API documentation

---

## Next Steps

### Storage Management - Future Enhancements
- [ ] Add disk SMART test scheduling
- [ ] ZFS pool auto-import on boot
- [ ] Email alerts for disk health warnings
- [ ] Quota management per dataset

### Future Features
- [ ] Audit trail and logging system
- [ ] Notification system (webhooks, email)
