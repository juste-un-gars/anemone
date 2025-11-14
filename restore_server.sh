#!/bin/bash
# Anemone - Multi-user NAS with P2P encrypted synchronization
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0
#
# Server Configuration Restore Script
# Usage: sudo bash restore_server.sh <backup_file.enc> <passphrase>

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
   exit 1
fi

# Check arguments
if [ "$#" -ne 2 ]; then
    echo -e "${RED}Usage: sudo bash restore_server.sh <backup_file.enc> <passphrase>${NC}"
    exit 1
fi

BACKUP_FILE="$1"
PASSPHRASE="$2"

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}Error: Backup file '$BACKUP_FILE' not found${NC}"
    exit 1
fi

echo -e "${BLUE}🪸 Anemone Server Configuration Restore${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Detect distribution
if [ -f /etc/os-release ]; then
    . /etc/os-release
    DISTRO=$ID
else
    echo -e "${RED}Error: Cannot detect Linux distribution${NC}"
    exit 1
fi

# Check and install dependencies
echo -e "${BLUE}Checking system dependencies...${NC}"
MISSING_DEPS=()

for cmd in jq sqlite3 openssl go smbpasswd useradd; do
    if ! command -v $cmd &> /dev/null; then
        case $cmd in
            go) MISSING_DEPS+=("golang-go") ;;
            smbpasswd) MISSING_DEPS+=("samba") ;;
            *) MISSING_DEPS+=("$cmd") ;;
        esac
    fi
done

if [ ${#MISSING_DEPS[@]} -ne 0 ]; then
    echo -e "${YELLOW}Missing dependencies: ${MISSING_DEPS[*]}${NC}"
    echo -e "${BLUE}Installing dependencies...${NC}"

    case $DISTRO in
        ubuntu|debian|linuxmint)
            apt-get update -qq
            apt-get install -y -qq jq sqlite3 openssl golang-go samba passwd
            ;;
        fedora|rhel|centos)
            dnf install -y -q jq sqlite openssl golang samba passwd
            ;;
        *)
            echo -e "${RED}Error: Unsupported distribution: $DISTRO${NC}"
            echo -e "${YELLOW}Please install manually: jq sqlite3 openssl golang samba${NC}"
            exit 1
            ;;
    esac

    echo -e "${GREEN}✓ Dependencies installed${NC}"
else
    echo -e "${GREEN}✓ All dependencies already installed${NC}"
fi

# Confirmation
echo ""
echo -e "${YELLOW}⚠️  WARNING: This will restore the server configuration.${NC}"
echo -e "${YELLOW}    Existing configuration will be backed up to /srv/anemone.backup.$(date +%s)${NC}"
echo ""
read -p "Are you sure you want to continue? (yes/no): " -r
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Restore cancelled."
    exit 0
fi

# Set data directory
ANEMONE_DATA_DIR="${ANEMONE_DATA_DIR:-/srv/anemone}"
BACKUP_DIR="${ANEMONE_DATA_DIR}.backup.$(date +%s)"

echo ""
echo -e "${BLUE}[1/10] Compiling decrypt tool...${NC}"
cd "$(dirname "$0")"
go build -o /tmp/anemone-restore-decrypt ./cmd/anemone-restore-decrypt
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to compile decrypt tool${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Decrypt tool compiled${NC}"

echo ""
echo -e "${BLUE}[2/10] Decrypting backup file...${NC}"
DECRYPTED_JSON=$(/tmp/anemone-restore-decrypt "$BACKUP_FILE" "$PASSPHRASE" 2>&1)
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to decrypt backup file${NC}"
    echo -e "${RED}$DECRYPTED_JSON${NC}"
    echo -e "${YELLOW}Check that your passphrase is correct${NC}"
    rm -f /tmp/anemone-restore-decrypt
    exit 1
fi
echo -e "${GREEN}✓ Backup decrypted successfully${NC}"

# Save decrypted JSON to temp file
TEMP_JSON="/tmp/anemone_restore_$(date +%s).json"
echo "$DECRYPTED_JSON" > "$TEMP_JSON"

# Extract server name and export date from JSON
SERVER_NAME=$(echo "$DECRYPTED_JSON" | jq -r '.server_name // "Unknown"')
EXPORT_DATE=$(echo "$DECRYPTED_JSON" | jq -r '.exported_at // "Unknown"')

echo -e "${GREEN}  Server: $SERVER_NAME${NC}"
echo -e "${GREEN}  Exported: $EXPORT_DATE${NC}"

echo ""
echo -e "${BLUE}[3/10] Stopping Anemone service...${NC}"
systemctl stop anemone 2>/dev/null || true
echo -e "${GREEN}✓ Service stopped${NC}"

echo ""
echo -e "${BLUE}[4/10] Backing up existing data...${NC}"
if [ -d "$ANEMONE_DATA_DIR" ]; then
    mv "$ANEMONE_DATA_DIR" "$BACKUP_DIR"
    echo -e "${GREEN}✓ Existing data backed up to: $BACKUP_DIR${NC}"
else
    echo -e "${YELLOW}  No existing data directory found${NC}"
fi

echo ""
echo -e "${BLUE}[5/10] Creating data directories...${NC}"
mkdir -p "$ANEMONE_DATA_DIR"/{db,certs,shares,smb,backups/incoming}
chmod 700 "$ANEMONE_DATA_DIR"
echo -e "${GREEN}✓ Directories created${NC}"

echo ""
echo -e "${BLUE}[6/10] Restoring database...${NC}"
# Create database and restore data
DB_FILE="$ANEMONE_DATA_DIR/db/anemone.db"

# Create tables
sqlite3 "$DB_FILE" <<'EOF'
CREATE TABLE IF NOT EXISTS system_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    email TEXT,
    encryption_key_hash TEXT NOT NULL,
    encryption_key_encrypted BLOB NOT NULL,
    is_admin BOOLEAN DEFAULT 0,
    quota_total_gb INTEGER DEFAULT 100,
    quota_backup_gb INTEGER DEFAULT 50,
    language VARCHAR(2) DEFAULT 'fr',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    activated_at DATETIME,
    last_login DATETIME,
    restore_acknowledged BOOLEAN DEFAULT 0,
    restore_completed BOOLEAN DEFAULT 0
);

CREATE TABLE IF NOT EXISTS shares (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    protocol TEXT DEFAULT 'smb',
    sync_enabled BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

CREATE TABLE IF NOT EXISTS peers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    address TEXT NOT NULL,
    port INTEGER DEFAULT 8443,
    public_key TEXT,
    password TEXT,
    enabled BOOLEAN DEFAULT 1,
    status TEXT DEFAULT 'unknown',
    sync_enabled BOOLEAN DEFAULT 1,
    sync_frequency TEXT DEFAULT 'daily',
    sync_time TEXT DEFAULT '23:00',
    sync_day_of_week INTEGER,
    sync_day_of_month INTEGER,
    sync_interval_minutes INTEGER DEFAULT 60,
    last_seen DATETIME,
    last_sync DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled BOOLEAN DEFAULT 0,
    interval TEXT DEFAULT '1h',
    fixed_hour INTEGER DEFAULT 23,
    last_sync DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
EOF

# Insert system_config
echo "$DECRYPTED_JSON" | jq -r '.system_config[] | @json' | while read -r item; do
    KEY=$(echo "$item" | jq -r '.key')
    VALUE=$(echo "$item" | jq -r '.value')
    UPDATED_AT=$(echo "$item" | jq -r '.updated_at')
    sqlite3 "$DB_FILE" "INSERT INTO system_config (key, value, updated_at) VALUES ('$KEY', '$VALUE', '$UPDATED_AT');"
done

# Insert users
echo "$DECRYPTED_JSON" | jq -r '.users[] | @json' | while read -r user; do
    ID=$(echo "$user" | jq -r '.id')
    USERNAME=$(echo "$user" | jq -r '.username')
    PASSWORD_HASH=$(echo "$user" | jq -r '.password_hash')
    EMAIL=$(echo "$user" | jq -r '.email // ""')
    ENCRYPTION_KEY_HASH=$(echo "$user" | jq -r '.encryption_key_hash')
    ENCRYPTION_KEY_ENCRYPTED=$(echo "$user" | jq -r '.encryption_key_encrypted | @base64')
    IS_ADMIN=$(echo "$user" | jq -r '.is_admin')
    QUOTA_TOTAL=$(echo "$user" | jq -r '.quota_total_gb')
    QUOTA_BACKUP=$(echo "$user" | jq -r '.quota_backup_gb')
    LANGUAGE=$(echo "$user" | jq -r '.language // "fr"')
    CREATED_AT=$(echo "$user" | jq -r '.created_at')
    ACTIVATED_AT=$(echo "$user" | jq -r '.activated_at // "NULL"')

    # Decode base64 encryption key
    ENC_KEY_HEX=$(echo "$ENCRYPTION_KEY_ENCRYPTED" | base64 -d | xxd -p | tr -d '\n')

    sqlite3 "$DB_FILE" "INSERT INTO users (id, username, password_hash, email, encryption_key_hash, encryption_key_encrypted, is_admin, quota_total_gb, quota_backup_gb, language, created_at, activated_at) VALUES ($ID, '$USERNAME', '$PASSWORD_HASH', '$EMAIL', '$ENCRYPTION_KEY_HASH', X'$ENC_KEY_HEX', $IS_ADMIN, $QUOTA_TOTAL, $QUOTA_BACKUP, '$LANGUAGE', '$CREATED_AT', $(if [ "$ACTIVATED_AT" = "NULL" ]; then echo "NULL"; else echo "'$ACTIVATED_AT'"; fi));"
done

# Insert shares
echo "$DECRYPTED_JSON" | jq -r '.shares[] | @json' | while read -r share; do
    ID=$(echo "$share" | jq -r '.id')
    USER_ID=$(echo "$share" | jq -r '.user_id')
    NAME=$(echo "$share" | jq -r '.name')
    SHARE_PATH=$(echo "$share" | jq -r '.path')
    PROTOCOL=$(echo "$share" | jq -r '.protocol')
    SYNC_ENABLED=$(echo "$share" | jq -r '.sync_enabled')
    CREATED_AT=$(echo "$share" | jq -r '.created_at')

    sqlite3 "$DB_FILE" "INSERT INTO shares (id, user_id, name, path, protocol, sync_enabled, created_at) VALUES ($ID, $USER_ID, '$NAME', '$SHARE_PATH', '$PROTOCOL', $SYNC_ENABLED, '$CREATED_AT');"
done

# Insert peers
echo "$DECRYPTED_JSON" | jq -r '.peers[] | @json' | while read -r peer; do
    ID=$(echo "$peer" | jq -r '.id')
    NAME=$(echo "$peer" | jq -r '.name')
    ADDRESS=$(echo "$peer" | jq -r '.address')
    PORT=$(echo "$peer" | jq -r '.port')
    PUBLIC_KEY=$(echo "$peer" | jq -r '.public_key // ""')
    PASSWORD=$(echo "$peer" | jq -r '.password // ""')
    ENABLED=$(echo "$peer" | jq -r '.enabled')
    STATUS=$(echo "$peer" | jq -r '.status')
    SYNC_ENABLED=$(echo "$peer" | jq -r '.sync_enabled')
    SYNC_FREQUENCY=$(echo "$peer" | jq -r '.sync_frequency')
    SYNC_TIME=$(echo "$peer" | jq -r '.sync_time')
    SYNC_DAY_OF_WEEK=$(echo "$peer" | jq -r '.sync_day_of_week // "NULL"')
    SYNC_DAY_OF_MONTH=$(echo "$peer" | jq -r '.sync_day_of_month // "NULL"')
    SYNC_INTERVAL_MINUTES=$(echo "$peer" | jq -r '.sync_interval_minutes')
    CREATED_AT=$(echo "$peer" | jq -r '.created_at')

    sqlite3 "$DB_FILE" "INSERT INTO peers (id, name, address, port, public_key, password, enabled, status, sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month, sync_interval_minutes, created_at) VALUES ($ID, '$NAME', '$ADDRESS', $PORT, $(if [ -z "$PUBLIC_KEY" ]; then echo "NULL"; else echo "'$PUBLIC_KEY'"; fi), $(if [ -z "$PASSWORD" ]; then echo "NULL"; else echo "'$PASSWORD'"; fi), $ENABLED, '$STATUS', $SYNC_ENABLED, '$SYNC_FREQUENCY', '$SYNC_TIME', $(if [ "$SYNC_DAY_OF_WEEK" = "NULL" ]; then echo "NULL"; else echo "$SYNC_DAY_OF_WEEK"; fi), $(if [ "$SYNC_DAY_OF_MONTH" = "NULL" ]; then echo "NULL"; else echo "$SYNC_DAY_OF_MONTH"; fi), $SYNC_INTERVAL_MINUTES, '$CREATED_AT');"
done

# Insert sync_config (if it exists in backup)
if echo "$DECRYPTED_JSON" | jq -e '.sync_config' > /dev/null 2>&1; then
    SYNC_ENABLED=$(echo "$DECRYPTED_JSON" | jq -r '.sync_config.enabled // 0')
    SYNC_INTERVAL=$(echo "$DECRYPTED_JSON" | jq -r '.sync_config.interval // "1h"')
    FIXED_HOUR=$(echo "$DECRYPTED_JSON" | jq -r '.sync_config.fixed_hour // 23')
    sqlite3 "$DB_FILE" "INSERT INTO sync_config (id, enabled, interval, fixed_hour) VALUES (1, $SYNC_ENABLED, '$SYNC_INTERVAL', $FIXED_HOUR);"
fi

# Add server restoration flags
RESTORE_TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
sqlite3 "$DB_FILE" "INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES ('server_restored', '1', '$RESTORE_TIMESTAMP');"
sqlite3 "$DB_FILE" "INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES ('server_restored_at', '$RESTORE_TIMESTAMP', '$RESTORE_TIMESTAMP');"

# Set all users' restore_acknowledged to 0 (they need to acknowledge the restore)
sqlite3 "$DB_FILE" "UPDATE users SET restore_acknowledged = 0, restore_completed = 0;"

echo -e "${GREEN}✓ Database restored${NC}"

echo ""
echo -e "${BLUE}[7/10] Creating system users and directories...${NC}"
# Recreate users from database
USER_COUNT=$(echo "$DECRYPTED_JSON" | jq '.users | length')
echo "$DECRYPTED_JSON" | jq -r '.users[] | @json' | while read -r user; do
    USERNAME=$(echo "$user" | jq -r '.username')
    USER_ID=$(echo "$user" | jq -r '.id')

    # Create system user if it doesn't exist
    if ! id "$USERNAME" &>/dev/null; then
        useradd -M -s /bin/bash "$USERNAME"
        echo -e "  ${GREEN}✓${NC} Created system user: $USERNAME"
    else
        echo -e "  ${YELLOW}○${NC} System user already exists: $USERNAME"
    fi

    # Create user directories
    mkdir -p "$ANEMONE_DATA_DIR/shares/$USERNAME"/{backup,data}
    chown -R "$USERNAME:$USERNAME" "$ANEMONE_DATA_DIR/shares/$USERNAME"
    chmod 700 "$ANEMONE_DATA_DIR/shares/$USERNAME"/{backup,data}
done
echo -e "${GREEN}✓ System users created ($USER_COUNT users)${NC}"

echo ""
echo -e "${BLUE}[8/10] Creating Samba users...${NC}"
echo "$DECRYPTED_JSON" | jq -r '.users[] | @json' | while read -r user; do
    USERNAME=$(echo "$user" | jq -r '.username')

    # Generate random password for SMB
    SMB_PASSWORD=$(openssl rand -base64 16)

    # Create SMB user
    (echo "$SMB_PASSWORD"; echo "$SMB_PASSWORD") | smbpasswd -a "$USERNAME" -s 2>/dev/null || true
    smbpasswd -e "$USERNAME" 2>/dev/null || true
    echo -e "  ${GREEN}✓${NC} Created SMB user: $USERNAME"
done
echo -e "${GREEN}✓ Samba users created${NC}"

echo ""
echo -e "${BLUE}[9/10] Generating TLS certificate...${NC}"
# Generate self-signed certificate
openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 \
    -subj "/C=FR/ST=State/L=City/O=Anemone/CN=localhost" \
    -keyout "$ANEMONE_DATA_DIR/certs/server.key" \
    -out "$ANEMONE_DATA_DIR/certs/server.crt" 2>/dev/null
chmod 600 "$ANEMONE_DATA_DIR/certs/server.key"
chmod 644 "$ANEMONE_DATA_DIR/certs/server.crt"
echo -e "${GREEN}✓ TLS certificate generated${NC}"

echo ""
echo -e "${BLUE}[10/10] Generating Samba configuration...${NC}"

# Compile anemone-smbgen if not already available
if ! command -v anemone-smbgen &> /dev/null; then
    echo -e "${YELLOW}  Compiling anemone-smbgen...${NC}"
    cd "$(dirname "$0")"
    go build -o /usr/local/bin/anemone-smbgen ./cmd/anemone-smbgen
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error: Failed to compile anemone-smbgen${NC}"
        exit 1
    fi
    echo -e "${GREEN}  ✓ anemone-smbgen compiled${NC}"
fi

# Generate Samba config
anemone-smbgen "$ANEMONE_DATA_DIR/db/anemone.db" > "$ANEMONE_DATA_DIR/smb/smb.conf.anemone"
echo -e "${GREEN}✓ Samba configuration generated${NC}"

# Cleanup
rm -f "$TEMP_JSON" /tmp/anemone-restore-decrypt

echo ""
echo -e "${BLUE}[11/12] Compiling Anemone binary...${NC}"
cd "$(dirname "$0")"
go build -o /usr/local/bin/anemone ./cmd/anemone
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to compile Anemone binary${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Anemone binary compiled${NC}"

echo ""
echo -e "${BLUE}[12/12] Setting up Anemone service...${NC}"

# Check if systemd service already exists
if [ -f /etc/systemd/system/anemone.service ]; then
    echo -e "${GREEN}✓ Systemd service already exists${NC}"
else
    echo -e "${YELLOW}  Creating systemd service...${NC}"
    cat > /etc/systemd/system/anemone.service <<EOF
[Unit]
Description=Anemone NAS Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/srv/anemone
Environment="ANEMONE_DATA_DIR=/srv/anemone"
ExecStart=/usr/local/bin/anemone
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable anemone
    echo -e "${GREEN}  ✓ Systemd service created and enabled${NC}"
fi

# Start Anemone service
systemctl restart anemone
sleep 2

if systemctl is-active --quiet anemone; then
    echo -e "${GREEN}✓ Anemone service started successfully${NC}"
else
    echo -e "${RED}⚠ Failed to start Anemone service${NC}"
    echo -e "${YELLOW}  Check logs: journalctl -u anemone -n 50${NC}"
fi

# Reload Samba
systemctl reload smbd 2>/dev/null || systemctl restart smbd 2>/dev/null || true

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ Server restoration complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${GREEN}Server status:${NC}"
systemctl status anemone --no-pager -l | head -10
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Access web interface: ${BLUE}https://$(hostname -I | awk '{print $1}'):8443${NC}"
echo -e "  2. Login with restored credentials"
echo -e "  3. Check logs: ${BLUE}sudo journalctl -u anemone -f${NC}"
echo ""
if [ -d "$BACKUP_DIR" ]; then
    echo -e "${YELLOW}Backup of previous configuration saved to:${NC}"
    echo -e "  $BACKUP_DIR"
    echo ""
fi
