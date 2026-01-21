# Storage Setup with ZFS

This guide explains how to prepare your storage before installing Anemone.

## Why Use ZFS?

**Strongly recommended** - Configure a ZFS pool **before** installing Anemone.

### Benefits

- **Data protection**: Checksums on all data detect and prevent corruption
- **Built-in RAID**: Mirror, RaidZ, RaidZ2 for redundancy
- **Instant snapshots**: Point-in-time backups without downtime
- **Transparent compression**: Automatic space savings (lz4)
- **Native quotas**: Per-user storage limits (used by Anemone)
- **Self-healing**: Automatic error detection and correction

### Without ZFS

Anemone will work on a regular filesystem, but you lose:
- No redundancy (disk failure = data loss)
- No snapshots
- No compression
- No corruption protection

---

## Setup Wizard Options

When you run Anemone's setup wizard, you'll see these storage options:

| Option | Description |
|--------|-------------|
| **Default** | Uses `/srv/anemone` on existing filesystem |
| **Use existing ZFS pool** | Select an existing pool, creates `anemone` dataset |
| **Create new ZFS pool** | Create a pool from available disks |
| **Custom paths** | Specify custom directories |

---

## Option 1: Create ZFS Pool via Setup Wizard

The setup wizard can create a ZFS pool for you.

### Prerequisites

**Debian/Ubuntu**:
```bash
sudo apt update
sudo apt install zfsutils-linux -y
```

**Fedora/RHEL**:
```bash
# ZFS requires an external repository on Fedora
# See: https://openzfs.github.io/openzfs-docs/Getting%20Started/Fedora/index.html
```

### RAID Levels

The wizard supports:

| Type | Min Disks | Fault Tolerance | Usable Capacity | Recommendation |
|------|-----------|-----------------|-----------------|----------------|
| **Single** | 1 | None | 100% | Testing only |
| **Mirror** | 2 | N-1 disks | 50% | Simple & reliable |
| **RaidZ** | 3 | 1 disk | (N-1)/N | Good balance |
| **RaidZ2** | 4 | 2 disks | (N-2)/N | Production |

---

## Option 2: Create ZFS Pool Manually (Before Installation)

If you prefer to create the pool yourself before running Anemone:

### Mirror (2 disks)
```bash
sudo zpool create -m /srv/anemone anemone-pool mirror /dev/sdb /dev/sdc
```

### RaidZ (3+ disks, tolerates 1 failure)
```bash
sudo zpool create -m /srv/anemone anemone-pool raidz /dev/sdb /dev/sdc /dev/sdd
```

### RaidZ2 (4+ disks, tolerates 2 failures)
```bash
sudo zpool create -m /srv/anemone anemone-pool raidz2 /dev/sdb /dev/sdc /dev/sdd /dev/sde
```

### Recommended Optimizations

```bash
# Enable compression (saves space)
sudo zfs set compression=lz4 anemone-pool

# Disable atime (improves performance)
sudo zfs set atime=off anemone-pool
```

Then in the setup wizard, select **"Use existing ZFS pool"** and choose your pool.

---

## Option 3: Using Cockpit (GUI)

For a graphical interface:

### Install Cockpit + ZFS Manager

**Debian/Ubuntu**:
```bash
sudo apt install cockpit -y
git clone https://github.com/45drives/cockpit-zfs-manager.git
sudo cp -r cockpit-zfs-manager/zfs /usr/share/cockpit
```

**Fedora/RHEL**:
```bash
sudo dnf install cockpit -y
git clone https://github.com/45drives/cockpit-zfs-manager.git
sudo cp -r cockpit-zfs-manager/zfs /usr/share/cockpit
```

### Create Pool

1. Open `https://your-server:9090` in your browser
2. Log in with your system credentials
3. Click **"ZFS"** in the left menu
4. Click **"Create Pool"**
5. Set mountpoint to `/srv/anemone`
6. Select your disks and RAID type
7. Click **"Create"**

---

## Useful Commands

```bash
# Check pool status
zpool status

# Check space usage
zfs list

# Create a snapshot
sudo zfs snapshot anemone-pool@$(date +%Y%m%d)

# List snapshots
zfs list -t snapshot

# Restore a snapshot
sudo zfs rollback anemone-pool@20260121

# Run integrity check (scrub)
sudo zpool scrub anemone-pool
```

---

## Disk Recommendations

| Use Case | Configuration | Notes |
|----------|---------------|-------|
| **Home/Testing** | 2 disks (Mirror) | Simple and reliable |
| **Small Business** | 4 disks (RaidZ2) | Good balance |
| **Production** | 6+ disks (RaidZ2) | Maximum safety |

### RAM Requirements

ZFS benefits from RAM for caching. Recommended: **1GB per TB** of storage.

---

## FAQ

### Can I migrate to ZFS after installation?

Yes, but it requires:
1. Stop Anemone
2. Create ZFS pool
3. Copy data to new pool
4. Update systemd service paths
5. Restart

It's easier to set up ZFS before installation.

### What if I don't have multiple disks?

You can use "Single" mode (no redundancy) or "Default" storage. Anemone will work fine, but consider regular backups since you have no disk fault tolerance.

### ZFS or Btrfs?

Anemone's setup wizard only supports ZFS pool creation. If you prefer Btrfs, create it manually and use the "Custom paths" option in the wizard.

---

## Resources

- [OpenZFS Documentation](https://openzfs.github.io/openzfs-docs/)
- [Cockpit Project](https://cockpit-project.org/)
- [Cockpit ZFS Manager](https://github.com/45drives/cockpit-zfs-manager)
