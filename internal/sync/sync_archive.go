// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file contains archive-based sync (legacy) and tar utilities.

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
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

// SyncShare synchronizes a share to a peer using HTTPS with encryption
// Creates tar.gz archive, encrypts it with user's key, and sends to peer
// NOTE: This is the legacy full-archive sync method. Prefer SyncShareIncremental for production use.
func SyncShare(db *sql.DB, req *SyncRequest) error {
	// Create sync log entry
	logID, err := CreateSyncLog(db, req.UserID, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// Get user's encryption key
	encryptionKey, err := GetUserEncryptionKey(db, req.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get encryption key: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Create tar.gz archive of the share directory
	var tarBuf bytes.Buffer
	fileCount, totalSize, err := createTarGz(&tarBuf, req.SharePath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create archive: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Encrypt the archive
	var encryptedBuf bytes.Buffer
	if err := crypto.EncryptStream(&tarBuf, &encryptedBuf, encryptionKey); err != nil {
		errMsg := fmt.Sprintf("Failed to encrypt archive: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Send encrypted archive to peer via HTTP POST
	peerURL := fmt.Sprintf("https://%s:%d/api/sync/receive", req.PeerAddress, req.PeerPort)

	// Create multipart form with share info
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add metadata fields
	writer.WriteField("share_id", fmt.Sprintf("%d", req.ShareID))
	writer.WriteField("user_id", fmt.Sprintf("%d", req.UserID))
	writer.WriteField("encrypted", "true") // Flag to indicate encryption
	// Extract share name from path (last directory)
	shareName := filepath.Base(filepath.Dir(req.SharePath))
	writer.WriteField("share_name", shareName)

	// Add encrypted archive file
	part, err := writer.CreateFormFile("archive", "share.tar.gz.enc")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create form file: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	io.Copy(part, &encryptedBuf)
	writer.Close()

	// Create HTTP client with optimized settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ClientSessionCache: tls.NewLRUClientSessionCache(32),
		},
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     120 * time.Second,
		DisableCompression:  true,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Minute, // 10 min timeout for large transfers
	}

	// Send POST request
	resp, err := client.Post(peerURL, writer.FormDataContentType(), &requestBody)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send to peer: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("Peer returned error %d: %s", resp.StatusCode, string(body))
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
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
		// Use filepath.Rel instead of deprecated filepath.HasPrefix
		absDestDir, err := filepath.Abs(destDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		absTargetPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute target path: %w", err)
		}
		relPath, err := filepath.Rel(absDestDir, absTargetPath)
		if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			return fmt.Errorf("illegal file path (path traversal detected): %s", header.Name)
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
