# Changelog

All notable changes to Anemone will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.15.0-beta] - 2026-02-08

### Changed

#### V2 UI Redesign Complete
- **Complete dark theme UI migration**: All 24 templates (16 main + 8 sub-pages) now use the v2 layout with dark theme, sidebar navigation, and CSS variable theming
- **8 sub-page forms migrated to v2**: Users add/quota/token/reset, Peers add/edit, Rclone edit, USB Backup edit
- **14 obsolete v1 templates removed**: All replaced admin and user templates deleted
- **Prototype cleanup**: Removed `cmd/v2preview/` standalone preview server and related files
- **Error rendering helpers**: Added `renderUsersAddError()` and `renderPeersAddError()` for consistent v2 form error re-rendering

### Technical
- Updated handlers: `handlers_admin_users.go`, `handlers_user.go`, `handlers_admin_peers.go`, `handlers_admin_rclone.go`, `handlers_admin_usb.go`
- All sub-page handlers now use `V2TemplateData` + `loadV2Page()` pattern
- Remaining v1 templates (intentional): auth/setup pages (7), backup POST error paths (5), rare pages (3)

## [0.13.6-beta] - 2026-02-08

### Fixed
- **USB Backup: false success on removed drive** - `IsMounted()` now checks `/proc/mounts` instead of just directory existence. Prevents false "success" status when a USB drive is physically removed without unmounting, and avoids writing backup data to the system disk by mistake.

## [0.13.5-beta] - 2026-01-31

### Added

#### SSH Key Generation for Rclone SFTP
- **Generate SSH key from UI**: One-click SSH key generation directly in Cloud Backup page
- **Display public key**: Copyable public key to add to remote servers' `authorized_keys`
- **Relative key paths**: Key paths stored as relative (e.g., `certs/rclone_key`), resolved at runtime for portability
- **Pre-filled key path**: Form auto-fills with generated key path when creating new SFTP destinations

### Documentation
- **New guide**: `docs/rclone-backup.md` - Complete guide for configuring rclone SFTP backup
  - Anemone configuration via web UI
  - Remote server setup (SSH user, authorized_keys)
  - Troubleshooting section

### Technical
- New file: `internal/rclone/sshkey.go` with GenerateSSHKey, GetSSHKeyInfo, ResolveKeyPath functions
- Modified `buildRemoteString()` to resolve relative key paths at runtime
- New routes: `/admin/rclone/key-info`, `/admin/rclone/generate-key`
- Updated `admin_rclone.html` with SSH key generation section
- i18n translations (FR + EN) for SSH key UI elements

## [0.13.4-beta] - 2026-01-31

### Added

#### Rclone Cloud Backup (SFTP)
- **New Cloud Backup module**: Backup user data to remote SFTP servers using rclone
- **Multiple SFTP destinations**: Configure multiple backup servers
- **SSH key or password auth**: Support both authentication methods
- **Connection testing**: Verify SFTP connection before syncing
- **Manual sync**: "Sync now" button for immediate backup
- **Automatic scheduling**: Interval, daily, weekly, or monthly schedules (like USB Backup)
- **Sync statistics**: Files/bytes synced, last sync time and status
- **Progress tracking**: Real-time progress display during sync

### Technical
- New package: `internal/rclone/` with rclone.go, sync.go, scheduler.go
- New database table: `rclone_backups` with SFTP config, scheduling, and status fields
- New admin page: `/admin/rclone` for cloud backup configuration
- Scheduler runs every minute, checks enabled backups
- i18n translations (FR + EN) for all rclone UI elements

## [0.13.3-beta] - 2026-01-30

### Added

#### WireGuard VPN Client Integration
- **New WireGuard management**: Configure VPN connection to access remote peers
- **Import .conf files**: Paste WireGuard configuration from your VPN provider
- **Connection control**: Connect/disconnect VPN directly from web interface
- **Auto-start option**: Automatically connect VPN when Anemone starts
- **Live status display**: Shows connection status, last handshake, and data transfer stats
- **Backup/Restore support**: WireGuard configuration included in server backups

### Technical
- New package: `internal/wireguard/` with config management, parser, and status
- New admin page: `/admin/wireguard` for VPN configuration
- New database table: `wireguard_config` for storing VPN settings
- Updated `install.sh`: Optional installation of `wireguard-tools`
- Added sudoers rules for `wg-quick up/down`, config file management
- i18n: Added French and English translations for WireGuard UI

## [0.13.2-beta] - 2026-01-30

### Changed

#### Code Refactoring
- **Split large files**: Refactored 4 files exceeding 800 lines to comply with CLAUDE.md guidelines
  - `sync.go` (1273→431 lines): Extracted `sync_incremental.go` (606) and `sync_archive.go` (279)
  - `handlers_admin_storage.go` (1152→246 lines): Extracted `handlers_admin_storage_zfs.go` (682) and `handlers_admin_storage_disk.go` (259)
  - `handlers_restore.go` (1002→794 lines): Extracted `handlers_restore_warning.go` (221)
  - `handlers_sync_api.go` (915→574 lines): Extracted `handlers_sync_api_read.go` (361)
- All source files now comply with the <800 lines guideline for better maintainability

## [0.13.1-beta] - 2026-01-30

### Added

#### Logging System
- **New logging infrastructure**: Structured logging with `log/slog` (Go 1.21+ standard library)
- **Log levels**: DEBUG, INFO, WARN, ERROR - configurable via UI or environment variable
- **Log rotation**: Daily log files with automatic cleanup (30 days or 200 MB max)
- **Log persistence**: Log level saved in database, persists across restarts
- **Admin UI**: New `/admin/logs` page to view/download logs and change log level
- **Environment override**: `ANEMONE_LOG_LEVEL` takes priority over database setting

### Changed
- Migrated ~40 files from standard `log` package to new `logger` package
- Logs now written to both stdout and file (`/srv/anemone/logs/anemone-YYYY-MM-DD.log`)
- Default log level is WARN (reduces noise in production)

### Technical
- New package: `internal/logger/` with `logger.go`, `rotation.go`, `logger_test.go`
- New config options: `ANEMONE_LOG_LEVEL`, `ANEMONE_LOG_DIR`
- New sysconfig functions: `GetLogLevel()`, `SetLogLevel()`
- New routes: `/admin/logs`, `/admin/logs/level`, `/admin/logs/download`

## [0.13.0-beta] - 2026-01-26

### Added

#### USB Backup Automatic Scheduling
- **Automatic sync scheduling**: Schedule USB backups to run automatically at configured intervals
- **Interval mode**: Every 15min, 30min, 1h, 2h, 4h, 8h, 12h, or 24h
- **Daily mode**: Every day at a specific time (HH:MM)
- **Weekly mode**: Every week on a specific day and time
- **Monthly mode**: Every month on a specific day (1-28) and time
- **Schedule configuration UI**: New section in USB backup edit form with enable/disable toggle, frequency selector, time picker, and day selectors
- **New template functions**: Added `deref` and `iterate` functions for schedule UI

### Changed
- New database columns for schedule: `sync_enabled`, `sync_frequency`, `sync_time`, `sync_day_of_week`, `sync_day_of_month`, `sync_interval_minutes`
- New scheduler runs every minute to check for enabled backups

## [0.12.0-beta] - 2026-01-26

### Added

#### USB Backup Refactoring
- **Backup type selection**: Choose between "Config only" or "Config + Data"
  - **Config only**: Backs up DB, certificates, smb.conf (~10 MB) - fits on any USB drive
  - **Config + Data**: Config plus selected user shares
- **Share selection**: Choose which shares to backup instead of all shares
- **Estimated size display**: Shows share sizes to help estimate required space

### Changed
- New database columns: `backup_type`, `selected_shares` in `usb_backups` table
- New `SyncConfig()` function for config-only backups
- `SyncAllShares()` now respects selected shares

## [0.11.9-beta] - 2026-01-26

### Fixed
- **USB drives not detected on NVMe systems**: Fixed detection that was hardcoded to exclude `/dev/sda*` assuming it's the system disk
  - Now dynamically detects the system disk via `findmnt /`
  - Any disk mounted in `/mnt/`, `/media/`, or `/run/media/` is now correctly detected

## [0.11.8-beta] - 2026-01-26

### Fixed
- **Format disk dialog missing options**: Added "Persistent mount" option to format dialog
  - Now includes all three options: Mount after format, Shared access, Persistent mount
  - All checked by default for convenience
- Note: Requires sudoers rule for `tee -a /etc/fstab` (added in install.sh, manual add needed for older installs)

## [0.11.7-beta] - 2026-01-26

### Added
- **Shared access option for disk mount**: New checkbox to allow all users read/write access to mounted disks
  - Available in both "Mount disk" and "Format disk" dialogs
  - Uses `umask=000` for FAT/exFAT, `chmod 777` for ext4/XFS
  - Checked by default for convenience

### Fixed
- **Persistent mount UID/GID hardcoded**: fstab entries now use actual user UID/GID instead of hardcoded 1000:1000
- **Trash listing showing parent directories**: Now shows actual deleted files with full relative path
  - Previously showed parent directories instead of files due to Samba keeptree=yes
  - Restore now works correctly to original location

## [0.11.5-beta] - 2026-01-25

### Added

#### Mount Disk Feature
- **Mount button for unmounted disks**: New "Mount" button for formatted but unmounted disks
- **Mount path selection dialog**: Dialog with validation for mount path selection
- **Persistent mount option**: Adds entry to /etc/fstab using UUID for automatic mounting on boot
- **FAT/exFAT support**: Proper uid/gid options for FAT filesystem mounting

### Changed
- **Combined columns in Physical Disks table**: Merged Filesystem and Status columns to reduce table width
  - Mounted disks show: mount point + (filesystem type)
  - Unmounted formatted disks show: filesystem badge + "(not mounted)"

### Fixed
- **Mount point directory cleanup**: Mount point directory now removed after unmount
- **Missing sudoers rules**: Added rules for mount with options, rmdir, and fstab management

## [0.11.4-beta] - 2026-01-25

### Added
- **Disk mount status display**: New "Status" column in Physical Disks table showing mount point with green icon
- **Unmount/Eject buttons**: Buttons to unmount or eject mounted disks directly from Storage page
- **Frontend mount path validation**: Validates mount path starts with /mnt/ or /media/ before submission

### Fixed
- **USB disk permissions after mount**: FAT32/exFAT disks now mounted with correct user ownership (uid/gid options)
- **ext4/XFS disk permissions**: Ownership set via chown after mounting
- **Missing auth check on unmount endpoint**: Added session verification to disk unmount handler
- **Missing sudoers for mount options**: Added `mount -o *` and `chown` permissions to sudoers

## [0.11.3-beta] - 2026-01-25

### Added
- **Mount after formatting**: Option to automatically mount disk after formatting with customizable mount path (default: /mnt/{diskname})
- **Eject disk button**: New eject button in USB Backup to safely unmount and eject USB drives
- **FAT32/exFAT tools in installer**: `install.sh` now installs `dosfstools` and `exfatprogs` automatically

### Fixed
- **Missing sudoers permissions**: Added mount, umount, eject, mkdir, mkfs.vfat, mkfs.exfat to sudoers configuration

## [0.11.2-beta] - 2026-01-25

### Changed
- **Disk formatting consolidated in Storage section**: All disk formatting (ext4, XFS, exFAT, FAT32) now in one place
- USB Backup section now links to Storage for formatting instead of duplicating the feature

### Added
- **exFAT and FAT32 options in Storage**: Can now format disks with Windows-compatible filesystems from the Storage page

## [0.11.1-beta] - 2026-01-25

### Fixed
- **Database locking causing slow page loads**: Added SQLite WAL mode and busy_timeout (5s) for better concurrency
- **Auto-update failing on existing tags**: Added `--force` flag to `git fetch --tags` to allow updating existing tags

## [0.11.0-beta] - 2026-01-25

### Added

#### USB Disk Formatting
- **Format USB drives from web UI**: New section in USB Backup page to format unmounted disks
- **FAT32 and exFAT support**: Windows-compatible formats for portable backups
- **Automatic detection of unmounted disks**: Detects USB drives that need formatting
- **Safe formatting**: Validates device path, checks if disk is in use before formatting

### Fixed
- **NVMe SMART data not displaying**: Fixed detection logic for NVMe drives which use a different health protocol than ATA/SATA drives
- NVMe drives now correctly show temperature, power-on hours, available spare, percentage used, and other health metrics

## [0.10.0-beta] - 2026-01-24

### Added

#### USB Backup Module
- **New `internal/usbbackup/` package**: Complete local backup solution for USB drives and external storage
- **Automatic drive detection**: Detects USB and external drives mounted in `/media/`, `/mnt/`, and removable devices
- **Encrypted backups**: AES-256-GCM encryption using existing crypto package
- **Manifest-based incremental sync**: Only transfers new/modified files, removes deleted files
- **Web UI integration**: Full management interface in admin dashboard
- **Status tracking**: Real-time progress, files/bytes synced, error reporting

#### Setup Wizard - Import Existing Installation
- **New wizard option** to recover Anemone after OS reinstallation
- Validates existing database at specified path
- Recreates system users and Samba accounts automatically
- Decrypts passwords using master_key from existing DB
- Regenerates smb.conf and reloads Samba
- Works like restore but without needing a backup file

#### Installation Script
- **Repair mode** in `install.sh` (option 2) for OS reinstallation recovery
- `simulate-reinstall.sh` test script for validating repair scenarios

### Changed
- **Setup detection refactored**: Uses `.needs-setup` marker file as single source of truth
- **Samba configuration**: Hide dot files (`.anemone/`, `.trash/`) from SMB shares

### Fixed
- **Systemd DATA_DIR bug**: Fixed hardcoded path in service file, now uses wizard-configured path
- **Data directory detection**: Improved auto-detection from `anemone.env` in repair scenarios
- **Storage options cleanup**: Removed redundant "existing ZFS pool" option from wizard

## [0.9.17-beta] - 2026-01-18

### Security
- **Fixed critical cross-user API vulnerability**: P2P sync API now validates user ownership before allowing file operations
- **Rate limiting**: Added rate limiting on login (5 attempts/15min) and password reset endpoints

### Added
- **API documentation**: Complete `docs/API.md` documenting all 55+ HTTP endpoints
- **Security documentation**: Comprehensive `docs/SECURITY.md` with architecture and best practices
- **Package documentation**: Added godoc comments to all 28 internal packages

### Changed
- **Refactored router.go**: Extracted 13 sync API handlers to `handlers_sync_api.go` (-18% code, 6,136 → 5,047 lines)
- **New btrfs package**: Centralized Btrfs utilities in `internal/btrfs/` (eliminates code duplication)
- **Standardized naming**: Renamed `scanBackupDirectory` functions to `scanBackupDir` for consistency

### Tests
- **37 new tests**: Added comprehensive test suites
  - `crypto_test.go`: 14 tests for encryption/hashing
  - `auth_test.go`: 14 tests for authentication/sessions
  - `users_test.go`: 5 tests for user management
  - `btrfs_test.go`: 4 tests for Btrfs utilities
- **Total test count**: 83 tests (up from ~46)

## [0.9.16-beta] - 2026-01-18

### Added

#### AnemoneSync Support
- **User share manifests**: Automatic generation of `.anemone/manifest.json` files in each user share (data and backup)
- **Manifest scheduler**: Background job generates manifests every 5 minutes with checksum caching for performance
- **AnemoneSync compatibility**: Manifest format designed for efficient file synchronization with AnemoneSync client
- **Atomic manifest writes**: Prevents partial reads during manifest updates

#### Manifest Features
- **SHA-256 checksums**: Each file entry includes a SHA-256 hash for integrity verification
- **Checksum caching**: Unchanged files (same size and mtime) reuse cached checksums for faster generation
- **Hidden file exclusion**: Automatically excludes hidden files, `.anemone/`, and `.trash/` directories
- **Progress logging**: Detailed logs showing manifest generation progress and statistics

### Documentation
- **USER_MANIFESTS.md**: Comprehensive documentation for the manifest system and AnemoneSync integration

## [0.9.15-beta] - 2026-01-13

### Security
- **Critical path traversal fixes**: Fixed 3 path traversal vulnerabilities in file download and extraction functions
- **Secure path validation**: Replaced `strings.HasPrefix()` with `filepath.Rel()` for proper path validation
- **Deprecated API removal**: Removed usage of deprecated `filepath.HasPrefix()` function

### Fixed
- **BuildManifest test**: Fixed test arguments for BuildManifest function

## [0.9.0-beta] - 2025-11-25

### Added

#### Core Features
- **Multi-user NAS with Btrfs quotas**: Create users with individual storage quotas enforced at filesystem level
- **P2P encrypted synchronization**: Peer-to-peer backup synchronization with AES-256-GCM encryption
- **Trash management**: Automated trash cleanup with configurable retention periods
- **File restoration**: Restore deleted files from local trash or remote peer backups
- **SMB integration**: Automatic Samba share creation and management per user
- **Complete backup system**: Full server backup/restore with encrypted database and configuration
- **Unlimited quotas**: Support for quota=0 to create users with unlimited storage

#### Web Interface
- **Fully bilingual interface**: Complete French and English translations for all pages
- **Language persistence**: User language preference saved automatically and persists across sessions
- **Admin dashboard**: User management, peer management, system settings, trash configuration
- **User dashboard**: Personal file statistics, backup status, language selection
- **Activation system**: Self-service user activation with encryption key generation and download
- **Restore interface**: Browse and restore files from local trash or remote backups with peer selection

#### Security
- **End-to-end encryption**: User data encrypted with individual encryption keys
- **Secure peer communication**: Password-protected peer-to-peer synchronization
- **Master key protection**: Database encryption with master key stored in secure configuration
- **HTTPS by default**: TLS certificate generation and configuration
- **Session management**: Secure cookie-based authentication

#### Administration
- **User management**: Create, edit, delete users with quotas and roles
- **Peer management**: Add, configure, test peer connections with automatic sync scheduling
- **Quota management**: Set or update user quotas with Btrfs subvolume enforcement
- **Trash settings**: Configure retention period with automatic cleanup
- **System backup**: Create and restore complete server backups
- **Activation tokens**: Generate time-limited activation links for new users

#### Automation
- **Automatic peer sync**: Configurable sync schedules (interval, daily, weekly, monthly)
- **Trash cleanup**: Scheduled deletion of files older than retention period
- **Backup scheduler**: Automated server backups with configurable frequency
- **Session cleanup**: Automatic removal of expired sessions

### Changed
- **Peer management simplified**: Removed redundant "Enable Synchronization" checkbox, keeping only "Enable automatic sync"
- **Restore UX improved**: Added peer selector to prevent duplicate restorations from multiple peers
- **Quota unlimited support**: Changed quota minimum from 1GB to 0GB with "0 = unlimited" help text
- **Language selector added**: Activation page now includes language dropdown for initial setup

### Fixed
- **Template compilation**: Fixed Go template parser error with JavaScript placeholder escaping in restore.html
- **Language persistence**: Fixed bug where language preference wasn't saved when using ?lang= parameter
- **Peer password backup**: Fixed double encryption issue - passwords now correctly decrypted before backup export
- **Manual restoration blocked**: Fixed issue where disabled peers couldn't be used for manual file restoration
- **Translation placeholders**: Replaced dict() function with correct key-value pairs for template parameter passing

### Security
- **Peer password handling**: Improved backup/restore process to properly handle encrypted peer passwords
- **Encryption key management**: Secure generation and storage of user encryption keys
- **Session security**: Implemented secure session handling with expiration

## Project Information

**Anemone** is a multi-user NAS with P2P encrypted synchronization designed for secure, distributed file storage and backup.

### Key Features
- Multi-user support with per-user Btrfs quotas
- P2P encrypted backup synchronization between servers
- Trash management with automatic cleanup
- File restoration from local or remote backups
- SMB/Samba integration for file sharing
- Complete server backup and disaster recovery
- Bilingual web interface (French/English)
- HTTPS by default with automatic TLS setup

### License
GNU Affero General Public License v3.0

### Author
juste-un-gars

### Repository
https://github.com/juste-un-gars/anemone

---

[Unreleased]: https://github.com/juste-un-gars/anemone/compare/v0.15.0-beta...HEAD
[0.15.0-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.6-beta...v0.15.0-beta
[0.13.6-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.5-beta...v0.13.6-beta
[0.13.5-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.4-beta...v0.13.5-beta
[0.13.4-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.3-beta...v0.13.4-beta
[0.13.3-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.2-beta...v0.13.3-beta
[0.13.2-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.1-beta...v0.13.2-beta
[0.13.1-beta]: https://github.com/juste-un-gars/anemone/compare/v0.13.0-beta...v0.13.1-beta
[0.13.0-beta]: https://github.com/juste-un-gars/anemone/compare/v0.12.0-beta...v0.13.0-beta
[0.12.0-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.9-beta...v0.12.0-beta
[0.11.9-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.8-beta...v0.11.9-beta
[0.11.8-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.7-beta...v0.11.8-beta
[0.11.7-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.5-beta...v0.11.7-beta
[0.11.5-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.4-beta...v0.11.5-beta
[0.11.4-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.3-beta...v0.11.4-beta
[0.11.3-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.2-beta...v0.11.3-beta
[0.11.2-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.1-beta...v0.11.2-beta
[0.11.1-beta]: https://github.com/juste-un-gars/anemone/compare/v0.11.0-beta...v0.11.1-beta
[0.11.0-beta]: https://github.com/juste-un-gars/anemone/compare/v0.10.0-beta...v0.11.0-beta
[0.10.0-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.10.0-beta
[0.9.17-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.17-beta
[0.9.16-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.16-beta
[0.9.15-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.15-beta
[0.9.0-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.0-beta
