# Anemone API Documentation

This document describes all HTTP endpoints provided by the Anemone web interface and P2P synchronization API.

**Base URL:** `https://your-server:8443` (HTTPS) or `http://your-server:8080` (HTTP, if enabled)

---

## Table of Contents

1. [Public Endpoints](#public-endpoints)
2. [Authentication](#authentication)
3. [User Endpoints](#user-endpoints)
4. [Admin Endpoints](#admin-endpoints)
5. [P2P Sync API](#p2p-sync-api)
6. [Restore API](#restore-api)

---

## Public Endpoints

### Health Check
```
GET /health
```
Returns server health status. Used for monitoring and load balancers.

**Response:** `200 OK` with `OK` body

---

### Initial Setup
```
GET /setup
POST /setup/confirm
```
First-time server setup to create admin account. Only available when no users exist.

**POST Parameters:**
- `username` - Admin username
- `password` - Admin password
- `password_confirm` - Password confirmation

---

## Authentication

### Login
```
GET /login
POST /login
```
User authentication page and form submission.

**POST Parameters:**
- `username` - Username
- `password` - Password

**Rate Limiting:** 5 attempts per 15 minutes per IP

---

### Logout
```
GET /logout
```
Terminates user session and redirects to login.

---

### Account Activation
```
GET /activate/{token}
POST /activate/confirm
```
Activates a new user account using email token.

**POST Parameters:**
- `token` - Activation token
- `password` - New password
- `password_confirm` - Password confirmation

---

### Password Reset
```
GET /reset-password
POST /reset-password
POST /reset-password/confirm
```
Request and complete password reset.

**POST /reset-password Parameters:**
- `username` - Username to reset

**POST /reset-password/confirm Parameters:**
- `token` - Reset token
- `password` - New password
- `password_confirm` - Password confirmation

**Rate Limiting:** 5 attempts per 15 minutes per IP

---

## User Endpoints

All user endpoints require authentication via session cookie.

### Dashboard
```
GET /dashboard
```
Main user dashboard showing shares and storage usage.

---

### User Settings
```
GET /settings
```
User settings page.

---

### Change Language
```
POST /settings/language
```
Change user interface language.

**Parameters:**
- `language` - Language code (`fr` or `en`)

---

### Change Password
```
POST /settings/password
```
Change user password.

**Parameters:**
- `current_password` - Current password
- `new_password` - New password
- `confirm_password` - Password confirmation

---

### Trash Management
```
GET /trash
GET /trash/{share}
POST /trash/{share}/restore
POST /trash/{share}/delete
POST /trash/{share}/empty
```
View and manage deleted files.

**Actions:**
- `restore` - Restore file from trash (param: `file`)
- `delete` - Permanently delete file (param: `file`)
- `empty` - Empty all trash for share

---

### Restore Warning
```
GET /restore-warning
POST /restore-warning/acknowledge
POST /restore-warning/bulk
```
Displayed when user has pending file restorations from remote backups.

---

### Sync Share Toggle
```
POST /sync/share/{id}
```
Enable or disable P2P synchronization for a specific share.

**Parameters:**
- `enabled` - `true` or `false`

---

## Admin Endpoints

All admin endpoints require admin role authentication.

### User Management
```
GET /admin/users
GET /admin/users/add
POST /admin/users/add
GET /admin/users/{id}
POST /admin/users/{id}/edit
POST /admin/users/{id}/delete
POST /admin/users/{id}/reset-password
```
Manage user accounts.

**Add User Parameters:**
- `username` - Username (alphanumeric, underscore, hyphen only)
- `email` - Email address
- `is_admin` - Admin role (`true`/`false`)
- `quota_gb` - Storage quota in GB

**Edit User Parameters:**
- `email` - Email address
- `is_admin` - Admin role
- `quota_gb` - Storage quota in GB
- `enabled` - Account enabled status

---

### Peer Management
```
GET /admin/peers
GET /admin/peers/add
POST /admin/peers/add
GET /admin/peers/{id}
POST /admin/peers/{id}/edit
POST /admin/peers/{id}/delete
POST /admin/peers/{id}/test
```
Manage P2P peer connections.

**Add/Edit Peer Parameters:**
- `name` - Peer display name
- `address` - Hostname or IP
- `port` - HTTPS port (default: 8443)
- `password` - Sync authentication password
- `sync_enabled` - Enable automatic sync
- `sync_frequency` - Frequency (`daily`, `weekly`, `monthly`, `interval`)
- `sync_day_of_week` - Day for weekly sync (0-6)
- `sync_day_of_month` - Day for monthly sync (1-28)
- `sync_interval_hours` - Hours between syncs for interval mode
- `sync_timeout_minutes` - Sync timeout in minutes

---

### System Settings
```
GET /admin/settings
POST /admin/settings/sync-password
POST /admin/settings/trash
```
Configure system-wide settings.

**Sync Password Parameters:**
- `password` - New sync authentication password

**Trash Settings Parameters:**
- `retention_days` - Days to keep deleted files (1-365)

---

### Synchronization
```
GET /admin/sync
POST /admin/sync/config
POST /admin/sync/force
```
View sync status and trigger manual synchronization.

**Config Parameters:**
- `enabled` - Enable automatic sync
- `interval` - Sync interval

**Force Sync Parameters:**
- `peer_id` - Peer ID to sync with

---

### Incoming Backups
```
GET /admin/incoming
POST /admin/incoming/delete
```
View and manage backups received from remote peers.

**Delete Parameters:**
- `source_server` - Source server name
- `user_id` - User ID
- `share_name` - Share name

---

### Server Backup
```
GET /admin/backup
POST /admin/backup/create
GET /admin/backup/download
POST /admin/backup/delete
```
Export and import server configuration.

**Create Parameters:**
- `password` - Encryption password

**Download Parameters:**
- `filename` - Backup filename

**Delete Parameters:**
- `filename` - Backup filename

---

### Restore Users from Remote
```
GET /admin/restore-users
POST /admin/restore-users/restore
```
Restore user accounts from remote peer backups.

---

### Share Management
```
GET /admin/shares
```
View all user shares across the system.

---

### System Updates
```
GET /admin/system/update
POST /admin/system/update/check
POST /admin/system/update/install
```
Check for and install system updates from GitHub releases.

---

## P2P Sync API

All P2P Sync API endpoints require sync authentication via Basic Auth.
Authentication uses the sync password configured in admin settings.

**Authentication Header:**
```
Authorization: Basic base64(sync:password)
```

### Receive Backup Archive
```
POST /api/sync/receive
Content-Type: multipart/form-data
```
Receives and extracts an encrypted backup archive from a peer.

**Form Parameters:**
- `user_id` - User ID
- `share_name` - Share name
- `encryption_key` - Encryption key for decryption
- `archive` - Encrypted tar.gz archive file

**Response:** `200 OK` or error

---

### Manifest Operations
```
GET /api/sync/manifest?user_id={id}&share_name={name}
PUT /api/sync/manifest
```
Get or update the file manifest for a share backup.

**PUT Body (JSON):**
```json
{
  "user_id": 1,
  "share_name": "documents",
  "files": [
    {
      "path": "file.txt",
      "size": 1024,
      "mod_time": "2025-01-18T10:00:00Z",
      "checksum": "sha256hash"
    }
  ]
}
```

---

### Source Info Update
```
PUT /api/sync/source-info
```
Updates source server information for a backup.

**Body (JSON):**
```json
{
  "user_id": 1,
  "share_name": "documents",
  "source_server": "server-name"
}
```

---

### Single File Operations
```
POST /api/sync/file
DELETE /api/sync/file
```
Upload or delete a single encrypted file.

**POST Form Parameters:**
- `user_id` - User ID
- `share_name` - Share name
- `path` - File path within share
- `encryption_key` - Encryption key
- `file` - Encrypted file content

**DELETE Query Parameters:**
- `user_id` - User ID
- `share_name` - Share name
- `path` - File path to delete

---

### List Physical Files
```
GET /api/sync/list-physical-files?user_id={id}&share_name={name}
```
Lists all physical files in a backup directory.

**Response (JSON):**
```json
{
  "files": [
    {
      "path": "file.txt",
      "size": 1024,
      "mod_time": "2025-01-18T10:00:00Z"
    }
  ]
}
```

---

### List User Backups
```
GET /api/sync/list-user-backups?user_id={id}
```
Lists all backup shares for a user.

**Response (JSON):**
```json
{
  "backups": [
    {
      "share_name": "documents",
      "file_count": 100,
      "total_size": 1048576
    }
  ]
}
```

---

### Download Encrypted Manifest
```
GET /api/sync/download-encrypted-manifest?user_id={id}&share_name={name}&encryption_key={key}
```
Downloads the encrypted manifest file for a backup.

**Response:** Binary encrypted manifest file

---

### Download Encrypted File
```
GET /api/sync/download-encrypted-file?user_id={id}&share_name={name}&path={path}&encryption_key={key}
```
Downloads a single encrypted file from a backup.

**Response:** Binary encrypted file content

---

### Delete User Backup
```
DELETE /api/sync/delete-user-backup?user_id={id}&share_name={name}
```
Deletes an entire user backup share.

**Response:** `200 OK` or error

---

## Restore API

User-facing API for restoring files from remote backups. Requires user authentication.

### List Available Backups
```
GET /api/restore/backups
```
Lists all backups available for the current user.

**Response (JSON):**
```json
{
  "backups": [
    {
      "source_server": "remote-server",
      "share_name": "documents",
      "file_count": 100,
      "total_size": 1048576,
      "last_modified": "2025-01-18T10:00:00Z"
    }
  ]
}
```

---

### List Files in Backup
```
GET /api/restore/files?source_server={server}&share_name={name}&path={path}
```
Lists files and directories in a backup.

**Query Parameters:**
- `source_server` - Source server name
- `share_name` - Share name
- `path` - Directory path (optional, defaults to root)

**Response (JSON):**
```json
{
  "files": [
    {
      "name": "file.txt",
      "path": "documents/file.txt",
      "size": 1024,
      "mod_time": "2025-01-18T10:00:00Z",
      "is_dir": false
    }
  ]
}
```

---

### Download Single File
```
GET /api/restore/download?source_server={server}&share_name={name}&path={path}
```
Downloads and decrypts a single file from backup.

**Response:** Decrypted file content with appropriate Content-Type

---

### Download Multiple Files
```
POST /api/restore/download-multiple
Content-Type: application/json
```
Downloads multiple files as a ZIP archive.

**Body (JSON):**
```json
{
  "source_server": "remote-server",
  "share_name": "documents",
  "paths": ["file1.txt", "folder/file2.txt"]
}
```

**Response:** ZIP archive containing decrypted files

---

## Error Responses

All endpoints return appropriate HTTP status codes:

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found |
| 405 | Method Not Allowed |
| 429 | Too Many Requests - Rate limited |
| 500 | Internal Server Error |

Error responses include a descriptive message in the body.

---

## Security Notes

1. **HTTPS Required:** All production deployments should use HTTPS (port 8443)
2. **Rate Limiting:** Login and password reset endpoints are rate-limited
3. **Sync Authentication:** P2P sync uses separate password-based auth
4. **Encryption:** All backup data is encrypted with AES-256-GCM
5. **Session Management:** Web sessions use secure, HttpOnly cookies

---

**Last Updated:** 2025-01-18
**Version:** 0.9.16-beta
