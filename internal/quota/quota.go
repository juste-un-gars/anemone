// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package quota

import (
	"database/sql"
	"fmt"

	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/users"
)

// QuotaInfo contains quota and usage information for a user
type QuotaInfo struct {
	UserID          int
	Username        string
	QuotaTotalGB    int
	QuotaBackupGB   int
	UsedTotalMB     int64
	UsedBackupMB    int64
	UsedDataMB      int64
	UsedTotalGB     float64
	UsedBackupGB    float64
	UsedDataGB      float64
	PercentUsed     float64
	PercentBackup   float64
	AlertLevel      string // "none", "warning" (75%), "danger" (90%), "critical" (100%+)
	BackupAlertLevel string
}

// GetUserQuota retrieves quota information for a user
func GetUserQuota(db *sql.DB, userID int) (*QuotaInfo, error) {
	// Get user info
	user, err := users.GetByID(db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get all user shares
	userShares, err := shares.GetByUser(db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user shares: %w", err)
	}

	// Calculate usage
	var totalUsedMB int64
	var backupUsedMB int64
	var dataUsedMB int64

	for _, share := range userShares {
		sizeMB, err := share.GetSizeMB()
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to calculate size for share %s: %v\n", share.Name, err)
			continue
		}

		totalUsedMB += sizeMB
		if share.Name == "backup" || share.Name == "backup_"+user.Username {
			backupUsedMB += sizeMB
		} else {
			dataUsedMB += sizeMB
		}
	}

	// Convert to GB for display
	usedTotalGB := float64(totalUsedMB) / 1024.0
	usedBackupGB := float64(backupUsedMB) / 1024.0
	usedDataGB := float64(dataUsedMB) / 1024.0

	// Calculate percentages
	percentUsed := 0.0
	if user.QuotaTotalGB > 0 {
		percentUsed = (usedTotalGB / float64(user.QuotaTotalGB)) * 100.0
	}

	percentBackup := 0.0
	if user.QuotaBackupGB > 0 {
		percentBackup = (usedBackupGB / float64(user.QuotaBackupGB)) * 100.0
	}

	// Determine alert levels
	alertLevel := getAlertLevel(percentUsed)
	backupAlertLevel := getAlertLevel(percentBackup)

	return &QuotaInfo{
		UserID:           userID,
		Username:         user.Username,
		QuotaTotalGB:     user.QuotaTotalGB,
		QuotaBackupGB:    user.QuotaBackupGB,
		UsedTotalMB:      totalUsedMB,
		UsedBackupMB:     backupUsedMB,
		UsedDataMB:       dataUsedMB,
		UsedTotalGB:      usedTotalGB,
		UsedBackupGB:     usedBackupGB,
		UsedDataGB:       usedDataGB,
		PercentUsed:      percentUsed,
		PercentBackup:    percentBackup,
		AlertLevel:       alertLevel,
		BackupAlertLevel: backupAlertLevel,
	}, nil
}

// getAlertLevel determines the alert level based on percentage used
func getAlertLevel(percent float64) string {
	if percent >= 100.0 {
		return "critical"
	} else if percent >= 90.0 {
		return "danger"
	} else if percent >= 75.0 {
		return "warning"
	}
	return "none"
}

// IsQuotaExceeded checks if a user has exceeded their quota
func IsQuotaExceeded(db *sql.DB, userID int) (bool, error) {
	info, err := GetUserQuota(db, userID)
	if err != nil {
		return false, err
	}
	return info.PercentUsed >= 100.0, nil
}

// GetAlertColor returns the Tailwind color class for an alert level
func GetAlertColor(level string) string {
	switch level {
	case "critical":
		return "red"
	case "danger":
		return "orange"
	case "warning":
		return "yellow"
	default:
		return "green"
	}
}

// UpdateUserQuota updates quota limits for a user
func UpdateUserQuota(db *sql.DB, userID int, quotaTotalGB, quotaBackupGB int) error {
	query := `UPDATE users SET quota_total_gb = ?, quota_backup_gb = ? WHERE id = ?`
	_, err := db.Exec(query, quotaTotalGB, quotaBackupGB, userID)
	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}
	return nil
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
