package storage

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// smartctlOutput represents the JSON output from smartctl
type smartctlOutput struct {
	SmartCtl struct {
		ExitStatus int `json:"exit_status"`
	} `json:"smartctl"`
	Device struct {
		Name     string `json:"name"`
		InfoName string `json:"info_name"`
		Type     string `json:"type"`
		Protocol string `json:"protocol"`
	} `json:"device"`
	SmartSupport struct {
		Available bool `json:"available"`
		Enabled   bool `json:"enabled"`
	} `json:"smart_support"`
	ModelName    string `json:"model_name"`
	SerialNumber string `json:"serial_number"`
	SmartStatus  struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`
	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	PowerCycleCount int `json:"power_cycle_count"`
	ATASmartAttributes struct {
		Table []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Value      int    `json:"value"`
			Worst      int    `json:"worst"`
			Thresh     int    `json:"thresh"`
			WhenFailed string `json:"when_failed"`
			Flags      struct {
				Value         int    `json:"value"`
				String        string `json:"string"`
				Prefailure    bool   `json:"prefailure"`
				UpdatedOnline bool   `json:"updated_online"`
				Performance   bool   `json:"performance"`
				ErrorRate     bool   `json:"error_rate"`
				EventCount    bool   `json:"event_count"`
				AutoKeep      bool   `json:"auto_keep"`
			} `json:"flags"`
			Raw struct {
				Value  int    `json:"value"`
				String string `json:"string"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	NVMeSmartHealthInfo struct {
		Temperature         int `json:"temperature"`
		AvailableSpare      int `json:"available_spare"`
		AvailableSpareThresh int `json:"available_spare_threshold"`
		PercentageUsed      int `json:"percentage_used"`
		DataUnitsRead       int64 `json:"data_units_read"`
		DataUnitsWritten    int64 `json:"data_units_written"`
		HostReads           int64 `json:"host_reads"`
		HostWrites          int64 `json:"host_writes"`
		PowerCycles         int `json:"power_cycles"`
		PowerOnHours        int `json:"power_on_hours"`
		UnsafeShutdowns     int `json:"unsafe_shutdowns"`
		MediaErrors         int `json:"media_errors"`
		NumErrLogEntries    int `json:"num_err_log_entries"`
	} `json:"nvme_smart_health_information_log"`
}

// SMART attribute IDs we care about
const (
	attrReallocatedSectors  = 5
	attrPowerOnHours        = 9
	attrPowerCycleCount     = 12
	attrTemperature         = 194
	attrTemperatureAlt      = 190
	attrCurrentPendingSector = 197
	attrUncorrectableSector = 198
)

// IsSmartAvailable checks if smartctl is installed
func IsSmartAvailable() bool {
	_, err := exec.LookPath("smartctl")
	return err == nil
}

// GetSMARTInfo retrieves SMART data for a disk
func GetSMARTInfo(devicePath string) (*SMARTInfo, error) {
	if !IsSmartAvailable() {
		return &SMARTInfo{Available: false}, nil
	}

	// Run smartctl with JSON output (requires sudo)
	cmd := exec.Command("sudo", "smartctl", "-a", "-j", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// smartctl may return non-zero even with valid data
		// Check if we got any output
		if len(output) == 0 {
			return &SMARTInfo{Available: false}, nil
		}
	}

	var smart smartctlOutput
	if err := json.Unmarshal(output, &smart); err != nil {
		return &SMARTInfo{Available: false}, nil
	}

	// Check if SMART is actually supported by this device
	// exit_status bit 1 (value 2) = device open failed
	// exit_status bit 2 (value 4) = SMART/ATA command failed
	// Also check smart_support.available field
	if !smart.SmartSupport.Available || (smart.SmartCtl.ExitStatus&6) != 0 {
		return &SMARTInfo{Available: false}, nil
	}

	info := &SMARTInfo{
		Available:   true,
		Healthy:     smart.SmartStatus.Passed,
		LastChecked: time.Now(),
	}

	// Handle NVMe drives differently
	if smart.Device.Protocol == "NVMe" || strings.Contains(devicePath, "nvme") {
		info.Temperature = smart.NVMeSmartHealthInfo.Temperature
		info.PowerOnHours = smart.NVMeSmartHealthInfo.PowerOnHours
		info.PowerCycleCount = smart.NVMeSmartHealthInfo.PowerCycles
		// NVMe media errors are significant
		if smart.NVMeSmartHealthInfo.MediaErrors > 0 {
			info.Healthy = false
		}
		return info, nil
	}

	// Handle SATA/SAS drives with ATA attributes
	info.Temperature = smart.Temperature.Current
	info.PowerOnHours = smart.PowerOnTime.Hours
	info.PowerCycleCount = smart.PowerCycleCount

	// Parse ATA SMART attributes
	for _, attr := range smart.ATASmartAttributes.Table {
		smartAttr := SMARTAttribute{
			ID:        attr.ID,
			Name:      attr.Name,
			Value:     attr.Value,
			Worst:     attr.Worst,
			Threshold: attr.Thresh,
			RawValue:  attr.Raw.String,
			Status:    "ok",
		}

		// Check if attribute is failing or near threshold
		if attr.Value <= attr.Thresh && attr.Thresh > 0 {
			smartAttr.Status = "failing"
		} else if attr.Value <= attr.Thresh+10 && attr.Thresh > 0 {
			smartAttr.Status = "warning"
		}

		// Extract key values
		switch attr.ID {
		case attrReallocatedSectors:
			info.ReallocatedSectors = attr.Raw.Value
		case attrCurrentPendingSector:
			info.PendingSectors = attr.Raw.Value
		case attrUncorrectableSector:
			info.UncorrectableSectors = attr.Raw.Value
		case attrTemperature, attrTemperatureAlt:
			if info.Temperature == 0 {
				info.Temperature = attr.Raw.Value
			}
		case attrPowerOnHours:
			if info.PowerOnHours == 0 {
				info.PowerOnHours = attr.Raw.Value
			}
		case attrPowerCycleCount:
			if info.PowerCycleCount == 0 {
				info.PowerCycleCount = attr.Raw.Value
			}
		}

		info.Attributes = append(info.Attributes, smartAttr)
	}

	// Mark as unhealthy if critical attributes are bad
	if info.ReallocatedSectors > 100 || info.PendingSectors > 0 || info.UncorrectableSectors > 0 {
		info.Healthy = false
	}

	return info, nil
}

// GetDiskHealth determines health status from SMART info
func GetDiskHealth(smart *SMARTInfo) HealthStatus {
	if smart == nil || !smart.Available {
		return HealthUnknown
	}

	if !smart.Healthy {
		return HealthCritical
	}

	// Check for warning signs
	if smart.ReallocatedSectors > 10 || smart.Temperature > 55 {
		return HealthWarning
	}

	// Check attributes for warnings
	for _, attr := range smart.Attributes {
		if attr.Status == "warning" {
			return HealthWarning
		}
		if attr.Status == "failing" {
			return HealthCritical
		}
	}

	return HealthOK
}

// GetDisksWithSMART returns all disks enriched with SMART data
func GetDisksWithSMART() ([]Disk, error) {
	disks, err := ListDisks()
	if err != nil {
		return nil, err
	}

	for i := range disks {
		smart, err := GetSMARTInfo(disks[i].Path)
		if err == nil && smart.Available {
			disks[i].SMARTData = smart
			disks[i].Health = GetDiskHealth(smart)
			disks[i].Temperature = smart.Temperature
			disks[i].PowerOnHours = smart.PowerOnHours
		}
	}

	return disks, nil
}

// ParseSMARTTemperature extracts temperature from various formats
func ParseSMARTTemperature(raw string) int {
	// Handle formats like "36 (Min/Max 20/47)"
	parts := strings.Split(raw, " ")
	if len(parts) > 0 {
		if temp, err := strconv.Atoi(parts[0]); err == nil {
			return temp
		}
	}
	return -1
}
