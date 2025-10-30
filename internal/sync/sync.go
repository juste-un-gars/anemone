// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package sync

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SyncLog represents a synchronization log entry
type SyncLog struct {
	ID           int
	UserID       int
	PeerID       int
	StartedAt    time.Time
	CompletedAt  *time.Time
	Status       string // "running", "success", "error"
	FilesSynced  int
	BytesSynced  int64
	ErrorMessage string
}

// SyncRequest represents a synchronization request
type SyncRequest struct {
	ShareID  int
	PeerID   int
	UserID   int
	SharePath string
	PeerAddress string
	PeerPort    int
}

// CreateSyncLog creates a new sync log entry and returns its ID
func CreateSyncLog(db *sql.DB, userID, peerID int) (int, error) {
	query := `INSERT INTO sync_log (user_id, peer_id, started_at, status)
	          VALUES (?, ?, CURRENT_TIMESTAMP, 'running')`

	result, err := db.Exec(query, userID, peerID)
	if err != nil {
		return 0, fmt.Errorf("failed to create sync log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get sync log ID: %w", err)
	}

	return int(id), nil
}

// UpdateSyncLog updates a sync log entry with completion details
func UpdateSyncLog(db *sql.DB, logID int, status string, filesSynced int, bytesSynced int64, errorMsg string) error {
	query := `UPDATE sync_log
	          SET completed_at = CURRENT_TIMESTAMP, status = ?, files_synced = ?, bytes_synced = ?, error_message = ?
	          WHERE id = ?`

	_, err := db.Exec(query, status, filesSynced, bytesSynced, errorMsg, logID)
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	return nil
}

// GetLastSyncByUser retrieves the last sync log for a user
func GetLastSyncByUser(db *sql.DB, userID int) (*SyncLog, error) {
	query := `SELECT id, user_id, peer_id, started_at, completed_at, status, files_synced, bytes_synced, error_message
	          FROM sync_log
	          WHERE user_id = ?
	          ORDER BY started_at DESC
	          LIMIT 1`

	log := &SyncLog{}
	err := db.QueryRow(query, userID).Scan(
		&log.ID, &log.UserID, &log.PeerID, &log.StartedAt, &log.CompletedAt,
		&log.Status, &log.FilesSynced, &log.BytesSynced, &log.ErrorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No sync yet
		}
		return nil, fmt.Errorf("failed to get last sync: %w", err)
	}

	return log, nil
}

// GetSyncLogs retrieves sync logs for a user with optional limit
func GetSyncLogs(db *sql.DB, userID int, limit int) ([]*SyncLog, error) {
	query := `SELECT id, user_id, peer_id, started_at, completed_at, status, files_synced, bytes_synced, error_message
	          FROM sync_log
	          WHERE user_id = ?
	          ORDER BY started_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sync logs: %w", err)
	}
	defer rows.Close()

	var logs []*SyncLog
	for rows.Next() {
		log := &SyncLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.PeerID, &log.StartedAt, &log.CompletedAt,
			&log.Status, &log.FilesSynced, &log.BytesSynced, &log.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// SyncShare synchronizes a share to a peer using HTTP/HTTPS API
// This is a simple implementation for testing: creates tar archive and sends to peer
// Will be replaced with rclone + encryption for production
func SyncShare(db *sql.DB, req *SyncRequest) error {
	// Create sync log entry
	logID, err := CreateSyncLog(db, req.UserID, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// Create tar.gz archive of the share directory
	var buf bytes.Buffer
	fileCount, totalSize, err := createTarGz(&buf, req.SharePath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create archive: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Send archive to peer via HTTP POST
	peerURL := fmt.Sprintf("https://%s:%d/api/sync/receive", req.PeerAddress, req.PeerPort)

	// Create multipart form with share info
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add metadata fields
	writer.WriteField("share_id", fmt.Sprintf("%d", req.ShareID))
	writer.WriteField("user_id", fmt.Sprintf("%d", req.UserID))
	// Extract share name from path (last directory)
	shareName := filepath.Base(filepath.Dir(req.SharePath))
	writer.WriteField("share_name", shareName)

	// Add archive file
	part, err := writer.CreateFormFile("archive", "share.tar.gz")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create form file: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}
	io.Copy(part, &buf)
	writer.Close()

	// Create HTTP client that accepts self-signed certs
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Minute, // 5 min timeout for large transfers
	}

	// Send POST request
	resp, err := client.Post(peerURL, writer.FormDataContentType(), &requestBody)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send to peer: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("Peer returned error %d: %s", resp.StatusCode, string(body))
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Update log with success
	err = UpdateSyncLog(db, logID, "success", fileCount, totalSize, "")
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	// Update peer's last_sync timestamp
	updatePeerQuery := `UPDATE peers SET last_sync = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = db.Exec(updatePeerQuery, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to update peer last_sync: %w", err)
	}

	return nil
}

// createTarGz creates a tar.gz archive of a directory
// Returns: fileCount, totalSize (bytes), error
func createTarGz(buf *bytes.Buffer, sourceDir string) (int, int64, error) {
	gzipWriter := gzip.NewWriter(buf)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	fileCount := 0
	var totalSize int64

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update header name to be relative to source dir
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If not a regular file, skip content
		if !info.Mode().IsRegular() {
			return nil
		}

		// Write file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		written, err := io.Copy(tarWriter, file)
		if err != nil {
			return err
		}

		fileCount++
		totalSize += written

		return nil
	})

	return fileCount, totalSize, err
}

// ExtractTarGz extracts a tar.gz archive to a destination directory
func ExtractTarGz(reader io.Reader, destDir string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract each file
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct full path
		targetPath := filepath.Join(destDir, header.Name)

		// Check for path traversal attacks
		if !filepath.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// Copy file content
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", targetPath, err)
			}

		default:
			// Skip other types (symlinks, etc.) for now
			continue
		}
	}

	return nil
}
