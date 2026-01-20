package storage

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// IsZFSAvailable checks if zpool command is available
func IsZFSAvailable() bool {
	_, err := exec.LookPath("zpool")
	return err == nil
}

// ListZFSPools returns all ZFS pools with their status
func ListZFSPools() ([]ZFSPool, error) {
	if !IsZFSAvailable() {
		return nil, nil
	}

	// Get list of pools
	cmd := exec.Command("sudo", "zpool", "list", "-H", "-p",
		"-o", "name,size,alloc,free,frag,cap,dedup,health")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pools []ZFSPool
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		pool := ZFSPool{
			Name:  fields[0],
			State: fields[7],
		}

		// Parse sizes (in bytes with -p flag)
		if size, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			pool.Size = size
			pool.SizeHuman = FormatBytes(size)
		}
		if alloc, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
			pool.Allocated = alloc
			pool.AllocHuman = FormatBytes(alloc)
		}
		if free, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
			pool.Free = free
			pool.FreeHuman = FormatBytes(free)
		}

		// Fragmentation and capacity (percentage)
		if frag, err := strconv.Atoi(fields[4]); err == nil {
			pool.Fragmentation = frag
		}
		if cap, err := strconv.Atoi(fields[5]); err == nil {
			pool.Capacity = cap
		}

		// Calculate used percentage
		if pool.Size > 0 {
			pool.UsedPercent = float64(pool.Allocated) / float64(pool.Size) * 100
		}

		// Deduplication ratio
		if dedup, err := strconv.ParseFloat(strings.TrimSuffix(fields[6], "x"), 64); err == nil {
			pool.Dedup = dedup
		}

		// Map health status
		pool.Health = mapZFSHealth(pool.State)

		// Get detailed vdev info
		vdevs, err := getPoolVDevs(pool.Name)
		if err == nil {
			pool.VDevs = vdevs
		}

		// Get scan status
		pool.ScanStatus = getPoolScanStatus(pool.Name)

		// Get errors
		pool.Errors = getPoolErrors(pool.Name)

		// Get datasets
		datasets, err := getPoolDatasets(pool.Name)
		if err == nil {
			pool.Datasets = datasets
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

// mapZFSHealth converts ZFS state to HealthStatus
func mapZFSHealth(state string) HealthStatus {
	switch strings.ToUpper(state) {
	case "ONLINE":
		return HealthOK
	case "DEGRADED":
		return HealthWarning
	case "FAULTED", "OFFLINE", "REMOVED", "UNAVAIL":
		return HealthCritical
	default:
		return HealthUnknown
	}
}

// getPoolVDevs retrieves vdev structure for a pool
func getPoolVDevs(poolName string) ([]ZFSVDev, error) {
	cmd := exec.Command("sudo", "zpool", "status", poolName)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var vdevs []ZFSVDev
	var currentVDev *ZFSVDev

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	inConfig := false

	// Regex for parsing vdev lines
	diskRegex := regexp.MustCompile(`^\s+([\w\-]+)\s+(ONLINE|OFFLINE|DEGRADED|FAULTED|UNAVAIL|REMOVED)\s+(\d+)\s+(\d+)\s+(\d+)`)
	vdevRegex := regexp.MustCompile(`^\s+(mirror-\d+|raidz\d?-\d+|spare-\d+|log|cache|special)\s+(ONLINE|OFFLINE|DEGRADED|FAULTED)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Start parsing after "config:" section
		if strings.Contains(line, "config:") {
			inConfig = true
			continue
		}

		if !inConfig {
			continue
		}

		// Stop at errors section
		if strings.Contains(line, "errors:") {
			break
		}

		// Skip pool name line and headers
		if strings.Contains(line, poolName) && strings.Contains(line, "ONLINE") {
			continue
		}
		if strings.Contains(line, "NAME") && strings.Contains(line, "STATE") {
			continue
		}

		// Check for vdev line (mirror-0, raidz1-0, etc.)
		if matches := vdevRegex.FindStringSubmatch(line); matches != nil {
			if currentVDev != nil {
				vdevs = append(vdevs, *currentVDev)
			}
			currentVDev = &ZFSVDev{
				Name:   matches[1],
				Type:   extractVDevType(matches[1]),
				State:  matches[2],
				Health: mapZFSHealth(matches[2]),
			}
			continue
		}

		// Check for disk line
		if matches := diskRegex.FindStringSubmatch(line); matches != nil {
			disk := ZFSDisk{
				Name:   matches[1],
				Path:   "/dev/" + matches[1],
				State:  matches[2],
				Health: mapZFSHealth(matches[2]),
			}
			if read, err := strconv.ParseUint(matches[3], 10, 64); err == nil {
				disk.Read = read
			}
			if write, err := strconv.ParseUint(matches[4], 10, 64); err == nil {
				disk.Write = write
			}
			if cksum, err := strconv.ParseUint(matches[5], 10, 64); err == nil {
				disk.Cksum = cksum
			}

			if currentVDev != nil {
				currentVDev.Disks = append(currentVDev.Disks, disk)
			} else {
				// Single disk pool (no vdev grouping)
				vdevs = append(vdevs, ZFSVDev{
					Name:   disk.Name,
					Type:   "disk",
					State:  disk.State,
					Health: disk.Health,
					Disks:  []ZFSDisk{disk},
				})
			}
		}
	}

	// Add last vdev
	if currentVDev != nil {
		vdevs = append(vdevs, *currentVDev)
	}

	return vdevs, nil
}

// extractVDevType extracts the vdev type from its name
func extractVDevType(name string) string {
	if strings.HasPrefix(name, "mirror") {
		return "mirror"
	}
	if strings.HasPrefix(name, "raidz3") {
		return "raidz3"
	}
	if strings.HasPrefix(name, "raidz2") {
		return "raidz2"
	}
	if strings.HasPrefix(name, "raidz") {
		return "raidz1"
	}
	if strings.HasPrefix(name, "spare") {
		return "spare"
	}
	if name == "log" {
		return "log"
	}
	if name == "cache" {
		return "cache"
	}
	if name == "special" {
		return "special"
	}
	return "disk"
}

// getPoolScanStatus retrieves the last scrub/resilver status
func getPoolScanStatus(poolName string) string {
	cmd := exec.Command("sudo", "zpool", "status", poolName)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "scan:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "  scan:"))
		}
	}

	return "none requested"
}

// getPoolErrors retrieves error summary for a pool
func getPoolErrors(poolName string) string {
	cmd := exec.Command("sudo", "zpool", "status", poolName)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "errors:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "errors:"))
		}
	}

	return "No known data errors"
}

// getPoolDatasets retrieves datasets for a pool
func getPoolDatasets(poolName string) ([]ZFSDataset, error) {
	cmd := exec.Command("sudo", "zfs", "list", "-H", "-p", "-r",
		"-o", "name,type,used,avail,refer,mountpoint,compression,compressratio",
		poolName)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var datasets []ZFSDataset
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		dataset := ZFSDataset{
			Name:        fields[0],
			Type:        fields[1],
			Mountpoint:  fields[5],
			Compression: fields[6],
		}

		// Parse sizes
		if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
			dataset.Used = used
			dataset.UsedHuman = FormatBytes(used)
		}
		if avail, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
			dataset.Available = avail
			dataset.AvailHuman = FormatBytes(avail)
		}
		if refer, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
			dataset.Refer = refer
		}

		// Compression ratio
		if ratio, err := strconv.ParseFloat(strings.TrimSuffix(fields[7], "x"), 64); err == nil {
			dataset.CompRatio = ratio
		}

		datasets = append(datasets, dataset)
	}

	return datasets, nil
}

// GetZFSPool retrieves a single pool by name
func GetZFSPool(name string) (*ZFSPool, error) {
	pools, err := ListZFSPools()
	if err != nil {
		return nil, err
	}

	for _, pool := range pools {
		if pool.Name == name {
			return &pool, nil
		}
	}

	return nil, nil
}
