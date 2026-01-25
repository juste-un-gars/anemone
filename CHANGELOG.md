# Changelog

All notable changes to Anemone will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/juste-un-gars/anemone/compare/v0.10.0-beta...HEAD
[0.10.0-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.10.0-beta
[0.9.17-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.17-beta
[0.9.16-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.16-beta
[0.9.15-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.15-beta
[0.9.0-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.0-beta
