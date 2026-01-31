# Rclone Cloud Backup

Anemone can backup user data to remote SFTP servers using rclone. This guide explains how to configure both the Anemone server (source) and the remote server (destination).

## Overview

- **What is backed up**: All users' `backup/` directories
- **Protocol**: SFTP (SSH File Transfer Protocol)
- **Authentication**: SSH key (recommended) or password
- **Sync type**: Incremental (only modified files are transferred)

## Prerequisites

### On Anemone Server (Source)

1. **Install rclone**:
   ```bash
   curl https://rclone.org/install.sh | sudo bash
   ```

2. **Verify installation**:
   ```bash
   rclone version
   ```

### On Remote Server (Destination)

1. **SSH server must be running**:
   ```bash
   # Debian/Ubuntu
   sudo apt install openssh-server
   sudo systemctl enable --now ssh

   # Fedora/RHEL
   sudo dnf install openssh-server
   sudo systemctl enable --now sshd
   ```

## Configuration Steps

### Step 1: Generate SSH Key (Anemone)

1. Go to **Admin Dashboard** > **Cloud Backup (rclone)**
2. In the **Anemone SSH Key** section, click **Generate SSH Key**
3. Copy the displayed public key

The key is stored at `{DATA_DIR}/certs/rclone_key` and uses the Ed25519 algorithm.

### Step 2: Configure Remote Server

Create a dedicated user for backups on the remote server:

```bash
# Create user
sudo useradd -m -s /bin/bash anemone-backup

# Create backup directory
sudo mkdir -p /srv/anemone-backups
sudo chown anemone-backup:anemone-backup /srv/anemone-backups

# Set up SSH key authentication
sudo mkdir -p /home/anemone-backup/.ssh
sudo nano /home/anemone-backup/.ssh/authorized_keys
# Paste the public key from Anemone here

# Set proper permissions
sudo chmod 700 /home/anemone-backup/.ssh
sudo chmod 600 /home/anemone-backup/.ssh/authorized_keys
sudo chown -R anemone-backup:anemone-backup /home/anemone-backup/.ssh
```

### Step 3: Configure Firewall (if applicable)

```bash
# Debian/Ubuntu
sudo ufw allow ssh

# Fedora/RHEL
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --reload
```

### Step 4: Add Destination in Anemone

1. Go to **Admin Dashboard** > **Cloud Backup (rclone)**
2. Fill in the form:
   - **Name**: A descriptive name (e.g., "Backup Server FR2")
   - **SFTP Server**: IP address or hostname of remote server
   - **Port**: 22 (default SSH port)
   - **Username**: `anemone-backup` (the user created above)
   - **SSH Key Path**: `certs/rclone_key` (pre-filled if key exists)
   - **Remote Path**: `/srv/anemone-backups`
   - **Enabled**: Check to enable
3. Click **Add Destination**

### Step 5: Test Connection

1. Click **Test** next to the configured destination
2. If successful, you'll see "Connection successful"
3. If it fails, check:
   - Remote server is reachable
   - SSH service is running
   - Public key is correctly added to `authorized_keys`
   - Firewall allows SSH

### Step 6: Configure Schedule (Optional)

1. Click **Edit** on the destination
2. Enable **Automatic Sync**
3. Choose frequency:
   - **Interval**: Every X minutes/hours
   - **Daily**: At a specific time
   - **Weekly**: On a specific day and time
   - **Monthly**: On a specific day of month and time
4. Save changes

## Manual Sync

Click **Sync Now** to start an immediate backup. The sync runs in the background.

## How It Works

1. Anemone scans all users' `backup/` directories
2. For each user, rclone syncs to `{remote_path}/backup/{username}/`
3. Only modified files are transferred (incremental)
4. Statistics are updated after each sync

## Directory Structure on Remote Server

```
/srv/anemone-backups/
└── backup/
    ├── alice/
    │   ├── documents/
    │   └── photos/
    ├── bob/
    │   └── projects/
    └── charlie/
        └── data/
```

## Troubleshooting

### Connection Failed

1. **Check SSH access manually**:
   ```bash
   ssh -i /srv/anemone/certs/rclone_key anemone-backup@remote-server
   ```

2. **Check rclone directly**:
   ```bash
   rclone lsd :sftp,host=remote-server,user=anemone-backup,key_file=/srv/anemone/certs/rclone_key: /srv/anemone-backups
   ```

### Permission Denied

- Verify the public key is in `authorized_keys`
- Check file permissions (700 for `.ssh`, 600 for `authorized_keys`)
- Ensure the backup directory is writable by the user

### Host Key Verification

On first connection, you may need to accept the remote server's host key:
```bash
ssh -i /srv/anemone/certs/rclone_key anemone-backup@remote-server
# Type 'yes' when prompted
```

### Sync Takes Too Long

- Initial sync may take a while for large datasets
- Subsequent syncs are incremental and faster
- Consider scheduling during off-peak hours

## Security Considerations

- **Use SSH keys** instead of passwords for automation
- **Create a dedicated user** on the remote server with limited permissions
- **Restrict the backup directory** to only what's needed
- **Use firewall rules** to limit SSH access to known IPs if possible
- **Monitor disk space** on the remote server

## Multiple Destinations

You can configure multiple SFTP destinations for redundancy:
1. Add multiple destinations in the UI
2. Each destination syncs independently
3. Consider geographic distribution for disaster recovery

## Related Documentation

- [P2P Sync](p2p-sync.md) - Peer-to-peer encrypted synchronization
- [USB Backup](usb-backup.md) - Local backup to USB drives
- [User Guide](user-guide.md) - General usage guide
