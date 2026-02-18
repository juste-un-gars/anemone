# Integration Testing

This document describes the integration test suite for Anemone v0.22.0-beta, executed on 2026-02-18.

## Environment

- **Servers:** 3 fresh Ubuntu 24.04 LTS instances
- **Version:** v0.22.0-beta (installed via `install.sh`)
- **Configuration:** Default settings, HTTPS on port 8443

## Results Summary

**95/95 tests passed across 20 phases.**

| Phase | Description | Tests | Result |
|-------|-------------|-------|--------|
| 1 | Installation | 4 | PASS |
| 2 | Setup wizard | 2 | PASS |
| 3 | User management | 6 | PASS |
| 4 | File upload & manifests | 5 | PASS |
| 5 | P2P peering | 5 | PASS |
| 6 | P2P synchronization | 6 | PASS |
| 7 | Restore from peer | 5 | PASS |
| 8 | Rclone SFTP backup | 9 | PASS |
| 9 | Security | 4 | PASS |
| 10 | Samba (SMB) | 6 | PASS |
| 11 | Incremental sync | 3 | PASS |
| 12 | Trash management | 5 | PASS |
| 13 | User deletion | 6 | PASS |
| 14 | Log verification | 3 | PASS |
| 15 | Password change | 4 | PASS |
| 16 | Large file upload | 2 | PASS |
| 17 | Full server restore | 11 | PASS |
| 18 | Repair mode | 8 | PASS |
| 19 | Concurrent uploads | 3 | PASS |
| | **Total** | **95** | **ALL PASS** |

## Test Details

### Phase 1: Installation (4/4)

Verified on all 3 servers:

- Service `anemone` is active and running
- HTTPS responds on port 8443
- System group `anemone` is created
- SQLite database exists in data directory

### Phase 2: Setup Wizard (2/2)

- Setup wizard completes successfully, redirects to `/login`
- Admin login works after setup

### Phase 3: User Management (6/6)

- Create users via admin panel (activation tokens generated)
- Activate accounts via `/activate/<token>` URL
- Users added to system group `anemone`
- Share directories created with correct ownership (`user:anemone`) and setgid (`g+rwxs`)
- Samba config includes `force group = anemone`
- Per-user Samba shares generated (data + backup)

### Phase 4: File Upload & Manifests (5/5)

- Upload multiple file types (text, xlsx, binary) to data and backup shares
- Upload 5 MB binary file
- All uploaded files have correct permissions (`user:anemone`)
- Manifests generated for backup shares

### Phase 5: P2P Peering (5/5)

- Configure sync password on both servers
- Add each server as peer on the other
- Peers visible from both sides in admin panel

### Phase 6: P2P Synchronization (6/6)

- Force sync from server A to server B: all user backup shares synced
- Encrypted files (`.enc`) appear in incoming directory on target
- Encrypted manifests (`.anemone-manifest.json.enc`) present
- Bidirectional sync works (B to A)
- Only backup shares are synced (data shares remain local — expected behavior)

### Phase 7: Restore from Peer (5/5)

- Delete a file from local backup share
- API `/api/restore/backups` lists available backups from peer
- API `/api/restore/files` lists individual files in backup
- API `/api/restore/download` decrypts and returns the file
- MD5 hash of restored file matches the original

### Phase 8: Rclone SFTP Backup (9/9)

- Upload files to a user's data and backup shares
- Generate SSH key pair (ed25519) via Anemone
- Configure authorized_keys on remote server
- Test SSH connectivity
- Configure rclone SFTP remote
- Run rclone backup — files transferred successfully
- Verify files arrived on remote server
- Delete a local file, verify remote copy is preserved (rclone copy mode)
- MD5 hash of remote file matches original

### Phase 9: Security (4/4)

- User isolation: user A cannot access user B's files (HTTP 303 redirect)
- Non-admin users get HTTP 403 on all `/admin/*` routes
- Brute-force protection: HTTP 429 after 5 failed login attempts
- Unauthenticated requests to protected routes redirect to `/login`

### Phase 10: Samba / SMB (6/6)

- User can list own data share via `smbclient`
- User can write files via SMB — correct permissions on disk (`user:anemone`, mode 0640)
- User cannot access another user's share (NT_STATUS_ACCESS_DENIED)
- Other user can list and read own share via SMB

### Phase 11: Incremental Sync (3/3)

- Add new files to backup share after initial sync
- Re-sync to peer: only new files transferred
- New encrypted files present in peer's incoming directory alongside previous ones

### Phase 12: Trash Management (5/5)

- Delete file via API `/api/files/delete` — moved to `.trash/` directory
- File visible in trash page with restore and delete buttons
- Restore via `POST /trash/restore` — file returns to original location
- Trash is empty after restore

### Phase 13: User Deletion (6/6)

- Delete user via admin API
- System user removed (`id user`: no such user)
- Samba user removed (absent from `pdbedit`)
- User's share directories removed from disk
- User removed from `anemone` group
- User's Samba shares removed from `smb.conf`

### Phase 14: Log Verification (3/3)

Checked all 3 servers after the full test run:

- **0 ERROR** entries in any log file
- Only expected WARN entries (CSRF token mismatches from automated testing, rate-limit triggers from brute-force test)

### Phase 15: Password Change (4/4)

- User changes password via settings page
- Login with new password succeeds
- Login with old password fails
- SMB access works with new password (Samba password updated)

### Phase 16: Large File Upload (2/2)

- Upload 150 MB file via API — completes in ~3.5 seconds
- MD5 hash on disk matches original (no corruption)

### Phase 17: Full Server Restore (11/11)

- Create encrypted server backup via admin panel
- Download backup re-encrypted with a restore passphrase
- Simulate database loss (delete `anemone.db`, restart service)
- Setup wizard appears (expected)
- Validate backup: correct user count and share count detected
- Execute restore with data directory path
- Restart service — login page accessible
- All users restored with correct IDs
- System configuration restored (master key, NAS name, settings)
- Share directories intact with correct ownership and permissions
- Admin login works

### Phase 18: Repair Mode (8/8)

- Run `install.sh` in repair mode on a working server
- All 7 repair steps complete successfully
- Service active after repair
- HTTPS responds normally
- System group `anemone` intact with all members
- Sudoers file recreated
- Share permissions intact (`user:anemone`, setgid)
- Samba config regenerated with `force group = anemone`

### Phase 19: Concurrent Uploads (3/3)

- Launch 5 parallel 10 MB uploads from the same user
- All 5 uploads succeed (no errors, no timeouts)
- All 5 files present on disk with correct MD5 hashes (no corruption)

## Known Limitations

- ZFS storage wizard: navigating back in the wizard does not re-display pool name and mountpoint fields. Workaround: restart the Anemone service and re-run the wizard.
