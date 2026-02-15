# Advanced Configuration

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ANEMONE_DATA_DIR` | `/srv/anemone` | Data directory |
| `ANEMONE_SHARES_DIR` | `$DATA_DIR/shares` | User shares directory (can be on separate disk, e.g., ZFS pool) |
| `ANEMONE_INCOMING_DIR` | `$DATA_DIR/backups/incoming` | Incoming backups directory |
| `PORT` | `8080` | HTTP port |
| `HTTPS_PORT` | `8443` | HTTPS port |
| `ENABLE_HTTP` | `false` | Enable HTTP server |
| `ENABLE_HTTPS` | `true` | Enable HTTPS server |
| `LANGUAGE` | `fr` | Default language (fr/en) |
| `TLS_CERT_PATH` | auto-generated | Custom TLS certificate |
| `TLS_KEY_PATH` | auto-generated | Custom TLS key |
| `ANEMONE_LOG_LEVEL` | `warn` | Log level: debug, info, warn, error |
| `ANEMONE_LOG_DIR` | `$DATA_DIR/logs` | Log files directory |
| `ANEMONE_OO_ENABLED` | `false` | Enable OnlyOffice document editing (requires Docker) |
| `ANEMONE_OO_URL` | `http://localhost:9980` | Internal URL of OnlyOffice Document Server |
| `ANEMONE_OO_SECRET` | auto-generated | JWT secret for OnlyOffice communication |

### Setting Variables

**Systemd service:**

```bash
sudo systemctl edit anemone
```

Add:
```ini
[Service]
Environment="ANEMONE_DATA_DIR=/data/anemone"
Environment="HTTPS_PORT=443"
```

Then:
```bash
sudo systemctl daemon-reload
sudo systemctl restart anemone
```

**Environment file:**

Variables are stored in `/etc/anemone/anemone.env`:

```bash
ANEMONE_DATA_DIR=/srv/anemone
ANEMONE_INCOMING_DIR=/mnt/backups/incoming
```

## External Drive

### Why Use External Storage

- Larger capacity
- Portability
- Hardware isolation
- Easy expansion

### Setup

```bash
# Identify drive
lsblk

# Format (optional - ERASES DATA)
sudo mkfs.ext4 /dev/sdb1
# Or for Btrfs (recommended for quotas):
sudo mkfs.btrfs -L ANEMONE_DATA /dev/sdb1

# Mount
sudo mkdir -p /srv/anemone
sudo mount /dev/sdb1 /srv/anemone
sudo chown -R $USER:$USER /srv/anemone
```

### Persistent Mount (fstab)

```bash
# Get UUID
sudo blkid /dev/sdb1

# Edit fstab
sudo nano /etc/fstab
```

Add:
```
UUID=your-uuid-here  /srv/anemone  ext4  defaults,nofail  0  2
```

For Btrfs:
```
UUID=your-uuid-here  /srv/anemone  btrfs  defaults,nofail  0  2
```

**Important flags:**
- `nofail` - System boots even if drive is disconnected
- Use UUID, not `/dev/sdX` (device names can change)

### Systemd Mount Dependency

```bash
sudo systemctl edit anemone
```

Add:
```ini
[Unit]
RequiresMountsFor=/srv/anemone
```

## Custom TLS Certificate

### Using Let's Encrypt

```bash
# Install certbot
sudo apt install certbot

# Get certificate
sudo certbot certonly --standalone -d nas.yourdomain.com

# Configure Anemone
sudo systemctl edit anemone
```

Add:
```ini
[Service]
Environment="TLS_CERT_PATH=/etc/letsencrypt/live/nas.yourdomain.com/fullchain.pem"
Environment="TLS_KEY_PATH=/etc/letsencrypt/live/nas.yourdomain.com/privkey.pem"
```

### Using Custom Certificate

```bash
# Place certificates
sudo mkdir -p /etc/anemone/certs
sudo cp your-cert.pem /etc/anemone/certs/
sudo cp your-key.pem /etc/anemone/certs/
sudo chmod 600 /etc/anemone/certs/*

# Configure
sudo systemctl edit anemone
```

Add:
```ini
[Service]
Environment="TLS_CERT_PATH=/etc/anemone/certs/your-cert.pem"
Environment="TLS_KEY_PATH=/etc/anemone/certs/your-key.pem"
```

## Separate Incoming Directory

Store incoming backups on different disk:

```bash
# During setup wizard, choose "Separate backup storage"
# Or set environment variable:
ANEMONE_INCOMING_DIR=/mnt/backup-disk/incoming
```

## Systemd Service Customization

### Full Service File

Location: `/etc/systemd/system/anemone.service`

```ini
[Unit]
Description=Anemone NAS
After=network.target
RequiresMountsFor=/srv/anemone

[Service]
Type=simple
User=anemone
Group=anemone
EnvironmentFile=/etc/anemone/anemone.env
ExecStart=/usr/local/bin/anemone
Restart=on-failure
RestartSec=10s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

### Increase File Limits

```bash
sudo systemctl edit anemone
```

Add:
```ini
[Service]
LimitNOFILE=65535
LimitNPROC=4096
```

## Firewall Configuration

### firewalld (Fedora/RHEL)

```bash
# Open ports
sudo firewall-cmd --add-port=8443/tcp --permanent
sudo firewall-cmd --add-port=445/tcp --permanent
sudo firewall-cmd --reload
```

### ufw (Ubuntu)

```bash
sudo ufw allow 8443/tcp
sudo ufw allow 445/tcp
sudo ufw reload
```

### iptables

```bash
sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 445 -j ACCEPT
sudo iptables-save | sudo tee /etc/iptables.rules
```

## Reverse Proxy

### Nginx

```nginx
server {
    listen 443 ssl;
    server_name nas.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/nas.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/nas.yourdomain.com/privkey.pem;

    location / {
        proxy_pass https://127.0.0.1:8443;
        proxy_ssl_verify off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Caddy

```
nas.yourdomain.com {
    reverse_proxy https://127.0.0.1:8443 {
        transport http {
            tls_insecure_skip_verify
        }
    }
}
```

## Backup and Restore

### Backup Data Directory

```bash
# Stop service
sudo systemctl stop anemone

# Backup
sudo tar -czvf anemone-backup-$(date +%Y%m%d).tar.gz /srv/anemone

# Start service
sudo systemctl start anemone
```

### Backup Database Only

```bash
sqlite3 /srv/anemone/db/anemone.db ".backup '/tmp/anemone-db-backup.db'"
```

## Performance Tuning

### For Large File Counts

```bash
# Increase inotify watches
echo "fs.inotify.max_user_watches=524288" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### For Slow Networks

Increase sync timeout in peer settings:
- Admin → Peers → Edit → Timeout Hours

## See Also

- [Installation](installation.md) - Basic setup
- [Storage Setup](storage-setup.md) - RAID configuration
- [Troubleshooting](troubleshooting.md) - Common issues
