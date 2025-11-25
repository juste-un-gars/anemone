# Changelog

All notable changes to Anemone will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/juste-un-gars/anemone/compare/v0.9.0-beta...HEAD
[0.9.0-beta]: https://github.com/juste-un-gars/anemone/releases/tag/v0.9.0-beta
