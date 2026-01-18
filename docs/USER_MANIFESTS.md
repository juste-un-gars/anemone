# User Share Manifests for AnemoneSync

This document describes the user share manifest system introduced in Anemone v0.9.16-beta. This feature enables efficient file synchronization with external sync clients like AnemoneSync.

## Overview

Anemone now generates manifest files in each user share directory. These manifests contain metadata about all files in the share, allowing sync clients to quickly detect changes without performing a full SMB directory scan.

### Key Benefits

- **Fast change detection**: Sync clients can compare local files against the manifest instead of scanning the entire share via SMB
- **Reduced network traffic**: Only metadata is transferred, not file content
- **Incremental sync support**: Manifests enable efficient delta synchronization
- **Checksum-based integrity**: SHA-256 hashes ensure file integrity verification

## Manifest Location

Manifests are stored in a hidden `.anemone` directory at the root of each user share:

```
/srv/anemone/shares/
├── alice/
│   ├── backup/
│   │   └── .anemone/
│   │       └── manifest.json    ← Manifest for backup_alice
│   └── data/
│       └── .anemone/
│           └── manifest.json    ← Manifest for data_alice
├── bob/
│   ├── backup/
│   │   └── .anemone/
│   │       └── manifest.json
│   └── data/
│       └── .anemone/
│           └── manifest.json
```

From an SMB client perspective:
```
\\server\data_alice\.anemone\manifest.json
\\server\backup_alice\.anemone\manifest.json
```

## Manifest Format

The manifest is a JSON file with the following structure:

```json
{
  "version": 1,
  "generated_at": "2026-01-18T10:30:00Z",
  "share_name": "data_alice",
  "share_type": "data",
  "username": "alice",
  "file_count": 1234,
  "total_size": 5368709120,
  "files": [
    {
      "path": "Documents/report.pdf",
      "size": 1048576,
      "mtime": 1737193800,
      "hash": "sha256:a1b2c3d4e5f6g7h8..."
    },
    {
      "path": "Images/photo.jpg",
      "size": 2097152,
      "mtime": 1737193500,
      "hash": "sha256:e5f6g7h8i9j0k1l2..."
    }
  ]
}
```

### Field Descriptions

#### Root Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | integer | Manifest format version (currently 1) |
| `generated_at` | string | ISO 8601 UTC timestamp of generation |
| `share_name` | string | Name of the share (e.g., "data_alice") |
| `share_type` | string | Either "data" or "backup" |
| `username` | string | Owner of the share |
| `file_count` | integer | Total number of files in manifest |
| `total_size` | integer | Total size of all files in bytes |
| `files` | array | Array of file entries |

#### File Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Relative path from share root (forward slashes) |
| `size` | integer | File size in bytes |
| `mtime` | integer | Unix timestamp of last modification |
| `hash` | string | SHA-256 hash in format "sha256:hexdigest" |

## Update Schedule

Manifests are automatically updated by a background scheduler:

- **Default interval**: Every 5 minutes
- **Initial generation**: 30 seconds after server startup
- **Checksum optimization**: Unchanged files (same size and mtime) reuse cached checksums

## Using Manifests in AnemoneSync

### Reading the Manifest

```go
// Example: Reading manifest from SMB share
manifestPath := ".anemone/manifest.json"
manifestData, err := smbClient.ReadFile(manifestPath)
if err != nil {
    // Fallback to classic SMB scan (compatible with non-Anemone servers)
    return scanRemoteRecursive(...)
}

var manifest UserManifest
if err := json.Unmarshal(manifestData, &manifest); err != nil {
    return fmt.Errorf("failed to parse manifest: %w", err)
}

// Use manifest for change detection
for _, remoteFile := range manifest.Files {
    localFile, exists := localIndex[remoteFile.Path]
    if !exists {
        // File needs to be downloaded
        downloadQueue = append(downloadQueue, remoteFile.Path)
    } else if localFile.Hash != remoteFile.Hash {
        // File has changed
        updateQueue = append(updateQueue, remoteFile.Path)
    }
}
```

### Change Detection Algorithm

1. **Read remote manifest** from `.anemone/manifest.json`
2. **Build local file index** with paths, sizes, mtimes, and hashes
3. **Compare entries**:
   - Remote file not in local → Download
   - Local file not in remote → Delete (if bidirectional sync)
   - Hash mismatch → Update
   - Same hash → Skip

### Fallback Strategy

If the manifest is unavailable or outdated, AnemoneSync should fall back to traditional SMB directory scanning:

```go
func syncWithFallback(share string) error {
    manifest, err := readManifest(share)
    if err != nil {
        log.Printf("Manifest unavailable, using SMB scan: %v", err)
        return syncViaSMBScan(share)
    }

    // Check if manifest is too old (e.g., > 10 minutes)
    if time.Since(manifest.GeneratedAt) > 10*time.Minute {
        log.Printf("Manifest is stale, refreshing via SMB scan")
        return syncViaSMBScan(share)
    }

    return syncViaManifest(manifest)
}
```

## Excluded Files

The following are automatically excluded from manifests:

- **Hidden files and directories**: Anything starting with `.` (e.g., `.DS_Store`, `.git/`)
- **Manifest directory**: `.anemone/` itself
- **Trash directory**: `.trash/`
- **Non-regular files**: Symlinks, pipes, devices, etc.

## Security Considerations

1. **No sensitive data**: Manifests only contain file metadata (paths, sizes, times, hashes), not file contents
2. **Hidden directory**: The `.anemone` directory is hidden by convention
3. **Permissions**: Manifest files are created with 0644 permissions (readable by all, writable by owner)
4. **Atomic writes**: Manifests are written atomically (temp file + rename) to prevent partial reads

## Troubleshooting

### Manifest not generated

Check the Anemone logs for manifest scheduler messages:
```
📋 Starting user manifest scheduler (every 5 minutes)...
✅ User manifest scheduler started
📋 Running initial manifest generation...
✅ Manifest generation complete in 2.5s: 4 shares processed (15234 files, 12.5 GB)
```

### Manifest is outdated

The scheduler runs every 5 minutes by default. If you need immediate updates, consider:
1. Restarting the Anemone service (triggers immediate generation)
2. Waiting for the next scheduled run

### Checksum calculation is slow

For large shares, initial manifest generation may take time as all file checksums are calculated. Subsequent runs will reuse cached checksums for unchanged files.

## Future Enhancements

- **Real-time updates**: inotify/fswatch integration for immediate manifest updates
- **Configurable interval**: Web UI setting to adjust update frequency
- **Partial manifests**: Support for directory-level manifests for very large shares
- **Compression**: Optional gzip compression for manifest files

## Version History

- **v0.9.16-beta**: Initial implementation of user share manifests
