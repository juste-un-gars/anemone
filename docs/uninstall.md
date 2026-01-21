# Uninstall

Complete removal of Anemone from your system.

## Quick Uninstall

**WARNING: This deletes ALL data permanently!**

```bash
sudo systemctl stop anemone
sudo systemctl disable anemone
sudo rm -rf /srv/anemone
sudo rm -f /etc/systemd/system/anemone.service
sudo rm -f /etc/sudoers.d/anemone-smb
sudo rm -rf /etc/anemone
sudo systemctl daemon-reload
```

## Step-by-Step Uninstall

### 1. Stop the Service

```bash
# If running as systemd service
sudo systemctl stop anemone
sudo systemctl disable anemone

# If running manually
pkill -f anemone
```

### 2. Remove Data

**WARNING: Deletes ALL user data, shares, and configuration!**

```bash
sudo rm -rf /srv/anemone
```

This removes:
- `/srv/anemone/db/` - SQLite database
- `/srv/anemone/shares/` - All user files
- `/srv/anemone/incoming/` - Backups from peers
- `/srv/anemone/certs/` - TLS certificates
- `/srv/anemone/smb/` - Samba configuration

### 3. Remove System Users (Optional)

Anemone creates system users for each activated user.

```bash
# List users (UID > 1000 are typically regular users)
awk -F: '$3 >= 1000 {print $1}' /etc/passwd

# Remove specific user
sudo userdel username
sudo rm -rf /home/username
```

### 4. Remove SMB Users

```bash
# List SMB users
sudo pdbedit -L

# Remove specific user
sudo smbpasswd -x username

# Remove all Anemone SMB users
for user in $(sudo pdbedit -L | cut -d: -f1); do
    if [ "$user" != "root" ] && [ "$user" != "nobody" ]; then
        echo "Removing: $user"
        sudo smbpasswd -x "$user"
    fi
done
```

### 5. Clean Samba Configuration

```bash
# Remove Anemone SMB config
sudo rm -f /etc/samba/smb.conf.anemone

# Restore original if backed up
sudo cp /etc/samba/smb.conf.orig /etc/samba/smb.conf 2>/dev/null

# Reload Samba
sudo systemctl reload smb     # Fedora
sudo systemctl reload smbd    # Debian/Ubuntu
```

### 6. Remove Sudoers Rules

```bash
sudo rm -f /etc/sudoers.d/anemone-smb
```

### 7. Remove Systemd Service

```bash
sudo rm -f /etc/systemd/system/anemone.service
sudo rm -rf /etc/systemd/system/anemone.service.d
sudo systemctl daemon-reload
```

### 8. Remove Configuration Directory

```bash
sudo rm -rf /etc/anemone
```

### 9. Remove Binary

```bash
sudo rm -f /usr/local/bin/anemone
sudo rm -f /usr/local/bin/anemone-dfree
sudo rm -f /usr/local/bin/anemone-dfree-wrapper.sh
```

### 10. Remove Source Code (Optional)

```bash
rm -rf ~/anemone
# Or wherever you cloned the repo
```

## One-Liner (Dangerous!)

**USE WITH EXTREME CAUTION**

```bash
sudo systemctl stop anemone 2>/dev/null; \
sudo systemctl disable anemone 2>/dev/null; \
sudo rm -rf /srv/anemone; \
sudo rm -f /etc/sudoers.d/anemone-smb; \
sudo rm -f /etc/systemd/system/anemone.service; \
sudo rm -rf /etc/anemone; \
sudo rm -f /usr/local/bin/anemone*; \
sudo systemctl daemon-reload; \
echo "Anemone removed (system/SMB users NOT removed)"
```

## What's NOT Removed

The quick uninstall does **not** remove:
- System users created for each Anemone user
- SMB users in Samba database
- Go installation
- Samba package
- Firewall rules

Remove these manually if needed.

## Verify Removal

```bash
# Check service
systemctl status anemone

# Check data directory
ls -la /srv/anemone

# Check binary
which anemone

# Check SMB users
sudo pdbedit -L
```

## Reinstalling

After uninstall, you can reinstall fresh:

```bash
git clone https://github.com/juste-un-gars/anemone.git
cd anemone
sudo ./install.sh
```

## See Also

- [Installation](installation.md) - Fresh install
- [Troubleshooting](troubleshooting.md) - Issues during uninstall
