# Anemone - File Structure Documentation

This document provides a comprehensive overview of all files in the Anemone project.

---

## 📦 Main Executables (`cmd/`)

### Core Application
- **`cmd/anemone/main.go`** - Main application entry point, starts the web server and services
- **`cmd/anemone-dfree/main.go`** - Samba disk-free reporting tool for quota enforcement
- **`cmd/anemone-smbgen/main.go`** - Generates Samba configuration files for shares

### Encryption & Security Tools
- **`cmd/anemone-decrypt/main.go`** - Decrypts encrypted backup files
- **`cmd/anemone-decrypt-password/main.go`** - Decrypts peer passwords
- **`cmd/anemone-encrypt-peer-password/main.go`** - Encrypts peer passwords for storage
- **`cmd/anemone-reencrypt-key/main.go`** - Re-encrypts user encryption keys with new master key

### Database & Restore Tools
- **`cmd/anemone-migrate/main.go`** - Database migration tool
- **`cmd/anemone-restore-decrypt/main.go`** - Decrypts and restores files from backups

---

## 🏗️ Core Packages (`internal/`)

### Authentication & Security
- **`internal/auth/middleware.go`** - HTTP middleware for authentication
- **`internal/auth/session.go`** - Session management and user sessions
- **`internal/crypto/crypto.go`** - AES-256-GCM encryption/decryption functions
- **`internal/tls/autocert.go`** - Automatic TLS certificate generation

### User Management
- **`internal/users/users.go`** - User CRUD operations and management
- **`internal/activation/tokens.go`** - User activation token generation and validation
- **`internal/reset/reset.go`** - Password reset token management

### Storage & Quota
- **`internal/shares/shares.go`** - Share creation and management
- **`internal/quota/quota.go`** - Disk quota calculation and reporting
- **`internal/quota/enforcement.go`** - Btrfs quota enforcement
- **`internal/smb/smb.go`** - Samba share creation and configuration

### Backup & Restore
- **`internal/backup/backup.go`** - User file backup to peers
- **`internal/restore/restore.go`** - File restoration from backups
- **`internal/bulkrestore/bulkrestore.go`** - Bulk file restoration from backups
- **`internal/incoming/incoming.go`** - Incoming backup reception from peers
- **`internal/serverbackup/serverbackup.go`** - Complete server backup and restore
- **`internal/trash/trash.go`** - Trash/recycle bin management
- **`internal/trash/scheduler.go`** - Automatic trash cleanup scheduler

### P2P Synchronization
- **`internal/sync/sync.go`** - P2P synchronization engine
- **`internal/sync/manifest.go`** - File manifest creation and comparison
- **`internal/sync/manifest_test.go`** - Unit tests for manifest functions
- **`internal/syncauth/syncauth.go`** - Sync endpoint authentication
- **`internal/syncconfig/syncconfig.go`** - Sync configuration management
- **`internal/peers/peers.go`** - Peer server management

### System Configuration
- **`internal/config/config.go`** - Application configuration
- **`internal/sysconfig/sysconfig.go`** - System-wide settings
- **`internal/database/database.go`** - SQLite database connection
- **`internal/database/migrations.go`** - Database schema migrations

### Web Interface
- **`internal/web/router.go`** - HTTP router and all request handlers

### Updates & Scheduling
- **`internal/updater/updater.go`** - GitHub release checking
- **`internal/updater/database.go`** - Update info database storage
- **`internal/updater/install.go`** - Automatic update installation
- **`internal/updater/scheduler.go`** - Automatic update check scheduler
- **`internal/scheduler/scheduler.go`** - Cron scheduler for periodic tasks

### Internationalization
- **`internal/i18n/i18n.go`** - Translation system
- **`internal/i18n/locales/en.json`** - English translations
- **`internal/i18n/locales/fr.json`** - French translations
- **`internal/i18n/locales/README.md`** - Translation documentation

---

## 🌐 Web Templates (`web/templates/`)

### Initial Setup
- **`web/templates/setup.html`** - Initial system setup page
- **`web/templates/setup_success.html`** - Setup completion confirmation

### Authentication
- **`web/templates/login.html`** - User login page
- **`web/templates/activate.html`** - User activation page
- **`web/templates/activate_success.html`** - Activation success confirmation
- **`web/templates/reset_password.html`** - Password reset page

### User Dashboard
- **`web/templates/dashboard_user.html`** - Regular user dashboard
- **`web/templates/settings.html`** - User settings page
- **`web/templates/trash.html`** - User trash/recycle bin
- **`web/templates/restore.html`** - File restoration interface
- **`web/templates/restore_warning.html`** - Restore operation warning

### Admin Dashboard
- **`web/templates/dashboard_admin.html`** - Administrator dashboard

### Admin - User Management
- **`web/templates/admin_users.html`** - User list and management
- **`web/templates/admin_users_add.html`** - Create new user
- **`web/templates/admin_users_quota.html`** - User quota management
- **`web/templates/admin_users_token.html`** - User activation token display
- **`web/templates/admin_users_reset_token.html`** - Password reset token generation

### Admin - Storage & Shares
- **`web/templates/admin_shares.html`** - Share management

### Admin - Backup & Restore
- **`web/templates/admin_backup.html`** - Backup configuration
- **`web/templates/admin_backup_export.html`** - Server backup export
- **`web/templates/admin_incoming.html`** - Incoming backup management
- **`web/templates/admin_restore_users.html`** - Bulk user restoration

### Admin - Sync & Peers
- **`web/templates/admin_peers.html`** - Peer server list
- **`web/templates/admin_peers_add.html`** - Add new peer server
- **`web/templates/admin_peers_edit.html`** - Edit peer configuration
- **`web/templates/admin_sync.html`** - Synchronization configuration

### Admin - System
- **`web/templates/admin_settings.html`** - System settings
- **`web/templates/admin_settings_trash.html`** - Trash retention settings
- **`web/templates/admin_system_update.html`** - System update interface

---

## 🔧 Scripts

### Installation & Configuration
- **`install.sh`** - Main installation script
- **`scripts/configure-smb-reload.sh`** - Samba configuration reload script
- **`scripts/auto-update.sh`** - Automatic update script
- **`scripts/README.md`** - Scripts documentation

### Wrappers & Utilities
- **`dfree-wrapper.sh`** - Wrapper script for anemone-dfree (used by Samba)
- **`restore_server.sh`** - Server restoration script

---

## 📚 Documentation

### User Documentation
- **`README.md`** - Project overview and main documentation
- **`QUICKSTART.md`** - Quick start guide
- **`CHANGELOG.md`** - Version history and changes

### Development Documentation
- **`FILES.md`** - This file - complete file structure documentation
- **`SESSION_STATE.md`** - Current development session state
- **`CHECKFILES.md`** - File verification checklist
- **`TESTS_ANEMONE.md`** - Testing documentation

### Security & Audit
- **`SECURITY_AUDIT.md`** - Security audit results

### Session Archives
- **`SESSIONS_ARCHIVE.md`** - Archived development sessions
- **`SESSION_STATE_ARCHIVE.md`** - Archived session states (various versions)
- **`SESSION_STATE_ARCHIVE_SESSIONS_8_11.md`** - Sessions 8-11 archive
- **`SESSION_STATE_ARCHIVE_SESSIONS_12_16.md`** - Sessions 12-16 archive
- **`SESSION_STATE_ARCHIVE_SESSIONS_13_19.md`** - Sessions 13-19 archive
- **`SESSION_STATE_ARCHIVE_SESSIONS_17_18_19.md`** - Sessions 17-19 archive
- **`SESSION_STATE_ARCHIVE_SESSIONS_20_24.md`** - Sessions 20-24 archive
- **`SESSION_STATE_ARCHIVE_SESSIONS_27_30.md`** - Sessions 27-30 archive
- **`SESSION_STATE_ARCHIVE_31_32_33_34.md`** - Sessions 31-34 archive

---

## 🗂️ Other

### Configuration
- **`.claude/settings.local.json`** - Claude Code local settings

### Temporary/Audit
- **`_audit_temp/`** - Temporary audit files (deprecated)
  - `_audit_temp/cmd/test-manifest/main.go`
  - `_audit_temp/README.md`
  - `_audit_temp/web/templates/base.html`

---

## 📊 Statistics

- **Total Go source files**: ~45
- **Total HTML templates**: 31
- **Total shell scripts**: 5
- **Total documentation files**: ~15

---

*Last updated: 2025-12-09 - Session 40*
