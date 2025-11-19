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
echo -e "${BLUE}[1/11] Downloading Go dependencies...${NC}"
cd "$(dirname "$0")"
# Download all dependencies first to avoid timeouts during compilation
go mod download 2>/dev/null || echo -e "${YELLOW}  Some dependencies may need to be downloaded during compilation${NC}"
echo -e "${GREEN}✓ Dependencies ready${NC}"

echo ""
echo -e "${BLUE}[2/11] Compiling decrypt tool...${NC}"
go build -o /tmp/anemone-restore-decrypt ./cmd/anemone-restore-decrypt
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to compile decrypt tool${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Decrypt tool compiled${NC}"

echo ""
echo -e "${BLUE}[3/11] Decrypting backup file...${NC}"
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
echo -e "${BLUE}[4/11] Stopping Anemone service...${NC}"
systemctl stop anemone 2>/dev/null || true
echo -e "${GREEN}✓ Service stopped${NC}"

echo ""
echo -e "${BLUE}[5/11] Backing up existing data...${NC}"
if [ -d "$ANEMONE_DATA_DIR" ]; then
    mv "$ANEMONE_DATA_DIR" "$BACKUP_DIR"
    echo -e "${GREEN}✓ Existing data backed up to: $BACKUP_DIR${NC}"
else
    echo -e "${YELLOW}  No existing data directory found${NC}"
fi

echo ""
echo -e "${BLUE}[6/11] Creating data directories...${NC}"
mkdir -p "$ANEMONE_DATA_DIR"/{db,certs,shares,smb,backups/incoming}
chmod 755 "$ANEMONE_DATA_DIR"
chmod 755 "$ANEMONE_DATA_DIR/shares"
chmod 700 "$ANEMONE_DATA_DIR"/{db,certs,smb,backups}
echo -e "${GREEN}✓ Directories created${NC}"

echo ""
echo -e "${BLUE}[7/11] Restoring database...${NC}"
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
    password_encrypted BLOB,
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

# Extract old master key from backup (before inserting system_config)
OLD_MASTER_KEY=$(echo "$DECRYPTED_JSON" | jq -r '.system_config[] | select(.key == "master_key") | .value')

if [ -z "$OLD_MASTER_KEY" ]; then
    echo -e "${RED}Error: Master key not found in backup${NC}"
    exit 1
fi

# Generate NEW master key for this server
NEW_MASTER_KEY=$(openssl rand -base64 32 | tr -d '\n')
echo -e "${GREEN}  ✓ Generated new master key for this server${NC}"

# Insert system_config (excluding master_key, we'll insert it later with new value)
echo "$DECRYPTED_JSON" | jq -r '.system_config[] | @json' | while read -r item; do
    KEY=$(echo "$item" | jq -r '.key')
    VALUE=$(echo "$item" | jq -r '.value')
    UPDATED_AT=$(echo "$item" | jq -r '.updated_at')

    # Skip master_key, we'll insert the new one later
    if [ "$KEY" != "master_key" ]; then
        sqlite3 "$DB_FILE" "INSERT INTO system_config (key, value, updated_at) VALUES ('$KEY', '$VALUE', '$UPDATED_AT');"
    fi
done

# Compile encryption key re-encryption tool
echo -e "${YELLOW}  Compiling encryption key re-encryption tool...${NC}"
cd "$(dirname "$0")"
go build -o /tmp/anemone-reencrypt-key ./cmd/anemone-reencrypt-key </dev/null 2>&1
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to compile re-encryption tool${NC}"
    exit 1
fi
echo -e "${GREEN}  ✓ Re-encryption tool compiled${NC}"

# Insert users (with re-encrypted encryption keys and passwords)
echo "$DECRYPTED_JSON" | jq -r '.users[] | @json' | while read -r user; do
    ID=$(echo "$user" | jq -r '.id')
    USERNAME=$(echo "$user" | jq -r '.username')
    PASSWORD_HASH=$(echo "$user" | jq -r '.password_hash')
    PASSWORD_ENCRYPTED=$(echo "$user" | jq -r '.password_encrypted // ""')
    EMAIL=$(echo "$user" | jq -r '.email // ""')
    ENCRYPTION_KEY_HASH=$(echo "$user" | jq -r '.encryption_key_hash')
    ENCRYPTION_KEY_ENCRYPTED=$(echo "$user" | jq -r '.encryption_key_encrypted')
    IS_ADMIN=$(echo "$user" | jq -r '.is_admin')
    QUOTA_TOTAL=$(echo "$user" | jq -r '.quota_total_gb')
    QUOTA_BACKUP=$(echo "$user" | jq -r '.quota_backup_gb')
    LANGUAGE=$(echo "$user" | jq -r '.language // "fr"')
    CREATED_AT=$(echo "$user" | jq -r '.created_at')
    ACTIVATED_AT=$(echo "$user" | jq -r '.activated_at // "NULL"')

    # Re-encrypt encryption key with new master key
    NEW_ENCRYPTION_KEY_ENCRYPTED=$(/tmp/anemone-reencrypt-key "$ENCRYPTION_KEY_ENCRYPTED" "$OLD_MASTER_KEY" "$NEW_MASTER_KEY" 2>&1)
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error: Failed to re-encrypt key for user $USERNAME${NC}"
        echo -e "${RED}$NEW_ENCRYPTION_KEY_ENCRYPTED${NC}"
        exit 1
    fi

    # Re-encrypt password with new master key (if exists)
    if [ -n "$PASSWORD_ENCRYPTED" ] && [ "$PASSWORD_ENCRYPTED" != "null" ]; then
        NEW_PASSWORD_ENCRYPTED=$(/tmp/anemone-reencrypt-key "$PASSWORD_ENCRYPTED" "$OLD_MASTER_KEY" "$NEW_MASTER_KEY" 2>&1)
        if [ $? -ne 0 ]; then
            echo -e "${RED}Error: Failed to re-encrypt password for user $USERNAME${NC}"
            echo -e "${RED}$NEW_PASSWORD_ENCRYPTED${NC}"
            exit 1
        fi
        # Decode base64 and insert as BLOB
        PASS_ENC_HEX=$(echo "$NEW_PASSWORD_ENCRYPTED" | base64 -d | xxd -p | tr -d '\n')
        PASS_ENC_SQL="X'$PASS_ENC_HEX'"
    else
        PASS_ENC_SQL="NULL"
    fi

    sqlite3 "$DB_FILE" "INSERT INTO users (id, username, password_hash, password_encrypted, email, encryption_key_hash, encryption_key_encrypted, is_admin, quota_total_gb, quota_backup_gb, language, created_at, activated_at) VALUES ($ID, '$USERNAME', '$PASSWORD_HASH', $PASS_ENC_SQL, '$EMAIL', '$ENCRYPTION_KEY_HASH', '$NEW_ENCRYPTION_KEY_ENCRYPTED', $IS_ADMIN, $QUOTA_TOTAL, $QUOTA_BACKUP, '$LANGUAGE', '$CREATED_AT', $(if [ "$ACTIVATED_AT" = "NULL" ]; then echo "NULL"; else echo "'$ACTIVATED_AT'"; fi));"
done

# Count users to display success message
USER_COUNT=$(echo "$DECRYPTED_JSON" | jq '.users | length')
echo -e "${GREEN}  ✓ Re-encrypted encryption keys and passwords for $USER_COUNT users${NC}"

# Insert NEW master key into system_config
CURRENT_TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
sqlite3 "$DB_FILE" "INSERT INTO system_config (key, value, updated_at) VALUES ('master_key', '$NEW_MASTER_KEY', '$CURRENT_TIMESTAMP');"
echo -e "${GREEN}  ✓ Inserted new master key into system_config${NC}"

# Cleanup re-encryption tool
rm -f /tmp/anemone-reencrypt-key 2>/dev/null

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

# Insert peers (if any exist)
if echo "$DECRYPTED_JSON" | jq -e '.peers' > /dev/null 2>&1 && [ "$(echo "$DECRYPTED_JSON" | jq '.peers')" != "null" ]; then
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
fi

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

# Disable all peers to prevent automatic sync from deleting backup files
# Admin must manually re-enable peers after restoring user files
sqlite3 "$DB_FILE" "UPDATE peers SET sync_enabled = 0;"
echo -e "${YELLOW}⚠️  All peers have been disabled to prevent data loss${NC}"
echo -e "${YELLOW}   Re-enable peers after restoring user files from admin interface${NC}"

echo -e "${GREEN}✓ Database restored${NC}"

echo ""
echo -e "${BLUE}[8/11] Creating system users and directories...${NC}"
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
echo -e "${BLUE}[9/11] Creating Samba users...${NC}"

# Compile password decryption tool
echo -e "${YELLOW}  Compiling password decryption tool...${NC}"
cd "$(dirname "$0")"
go build -o /tmp/anemone-decrypt-password ./cmd/anemone-decrypt-password </dev/null 2>&1
if [ $? -ne 0 ]; then
    echo -e "${RED}  Failed to compile password decryption tool${NC}"
    echo -e "${YELLOW}  Using temporary password instead${NC}"
    DECRYPT_TOOL_AVAILABLE=false
else
    echo -e "${GREEN}  ✓ Password decryption tool compiled${NC}"
    DECRYPT_TOOL_AVAILABLE=true
fi

# Use NEW master key to decrypt passwords (since they were re-encrypted with it)
# Note: We re-encrypted passwords with NEW_MASTER_KEY in the user insertion loop above
MASTER_KEY="$NEW_MASTER_KEY"

if [ -z "$MASTER_KEY" ]; then
    echo -e "${YELLOW}  ⚠️  Master key not found${NC}"
    echo -e "${YELLOW}  Using temporary password instead${NC}"
    DECRYPT_TOOL_AVAILABLE=false
fi

# Temporary SMB password (fallback)
TEMP_SMB_PASSWORD="anemone123"

# Read users from database (password_encrypted is now re-encrypted with new master key)
sqlite3 "$DB_FILE" "SELECT username, hex(password_encrypted) FROM users WHERE password_encrypted IS NOT NULL;" | while IFS='|' read -r USERNAME PASSWORD_ENCRYPTED_HEX; do
    # Convert hex to base64 for decryption tool
    PASSWORD_ENCRYPTED=$(echo "$PASSWORD_ENCRYPTED_HEX" | xxd -r -p | base64)

    # Try to decrypt password if tool is available
    if [ "$DECRYPT_TOOL_AVAILABLE" = true ] && [ -n "$PASSWORD_ENCRYPTED" ]; then
        # Decrypt password using NEW master key
        DECRYPTED_PASSWORD=$(/tmp/anemone-decrypt-password "$PASSWORD_ENCRYPTED" "$MASTER_KEY" 2>/dev/null)

        if [ $? -eq 0 ] && [ -n "$DECRYPTED_PASSWORD" ]; then
            # Use decrypted password
            (echo "$DECRYPTED_PASSWORD"; echo "$DECRYPTED_PASSWORD") | smbpasswd -a "$USERNAME" -s 2>/dev/null || true
            smbpasswd -e "$USERNAME" 2>/dev/null || true
            echo -e "  ${GREEN}✓${NC} Created SMB user: $USERNAME (password restored from backup)"
        else
            # Fallback to temporary password
            (echo "$TEMP_SMB_PASSWORD"; echo "$TEMP_SMB_PASSWORD") | smbpasswd -a "$USERNAME" -s 2>/dev/null || true
            smbpasswd -e "$USERNAME" 2>/dev/null || true
            echo -e "  ${YELLOW}○${NC} Created SMB user: $USERNAME (using temporary password: $TEMP_SMB_PASSWORD)"
        fi
    else
        # Use temporary password
        (echo "$TEMP_SMB_PASSWORD"; echo "$TEMP_SMB_PASSWORD") | smbpasswd -a "$USERNAME" -s 2>/dev/null || true
        smbpasswd -e "$USERNAME" 2>/dev/null || true
        echo -e "  ${YELLOW}○${NC} Created SMB user: $USERNAME (using temporary password: $TEMP_SMB_PASSWORD)"
    fi
done

# Also handle users without password_encrypted (just create with temp password)
sqlite3 "$DB_FILE" "SELECT username FROM users WHERE password_encrypted IS NULL;" | while read -r USERNAME; do
    (echo "$TEMP_SMB_PASSWORD"; echo "$TEMP_SMB_PASSWORD") | smbpasswd -a "$USERNAME" -s 2>/dev/null || true
    smbpasswd -e "$USERNAME" 2>/dev/null || true
    echo -e "  ${YELLOW}○${NC} Created SMB user: $USERNAME (using temporary password: $TEMP_SMB_PASSWORD)"
done

if [ "$DECRYPT_TOOL_AVAILABLE" = true ]; then
    echo -e "${GREEN}✓ Samba users created with restored passwords${NC}"
else
    echo -e "${YELLOW}✓ Samba users created with temporary passwords${NC}"
    echo -e "${YELLOW}  ⚠️  Admin should reset SMB passwords after restoration!${NC}"
fi

# Cleanup
rm -f /tmp/anemone-decrypt-password 2>/dev/null

echo ""
echo -e "${BLUE}[10/11] Generating TLS certificate...${NC}"
# Generate self-signed certificate
openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 \
    -subj "/C=FR/ST=State/L=City/O=Anemone/CN=localhost" \
    -keyout "$ANEMONE_DATA_DIR/certs/server.key" \
    -out "$ANEMONE_DATA_DIR/certs/server.crt" 2>/dev/null
chmod 600 "$ANEMONE_DATA_DIR/certs/server.key"
chmod 644 "$ANEMONE_DATA_DIR/certs/server.crt"
echo -e "${GREEN}✓ TLS certificate generated${NC}"

echo ""
echo -e "${BLUE}[11/11] Compiling Anemone tools and starting services...${NC}"

# Compile anemone-smbgen if not already available
if ! command -v anemone-smbgen &> /dev/null; then
    echo -e "${YELLOW}  Compiling anemone-smbgen...${NC}"
    cd "$(dirname "$0")"

    # Try to compile with retries (in case of network issues)
    RETRY_COUNT=0
    MAX_RETRIES=3
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        go build -o /usr/local/bin/anemone-smbgen ./cmd/anemone-smbgen 2>&1
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}  ✓ anemone-smbgen compiled${NC}"
            break
        fi
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
            echo -e "${YELLOW}  Retry $RETRY_COUNT/$MAX_RETRIES...${NC}"
            sleep 2
        else
            echo -e "${RED}  Failed to compile anemone-smbgen after $MAX_RETRIES attempts${NC}"
            echo -e "${YELLOW}  Continuing without Samba config (can be generated later)${NC}"
        fi
    done
fi

# Generate Samba config if tool is available
if command -v anemone-smbgen &> /dev/null; then
    anemone-smbgen "$ANEMONE_DATA_DIR/db/anemone.db" > "$ANEMONE_DATA_DIR/smb/smb.conf.anemone"
    echo -e "${GREEN}✓ Samba configuration generated${NC}"
else
    echo -e "${YELLOW}⚠ Samba configuration will need to be generated manually${NC}"
fi

# Copy web assets to data directory
echo ""
echo -e "${YELLOW}  Copying web assets...${NC}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cp -r "$SCRIPT_DIR/web" "$ANEMONE_DATA_DIR/"
echo -e "${GREEN}  ✓ Web assets copied${NC}"

# Compile Anemone binary
echo ""
echo -e "${YELLOW}  Compiling Anemone server...${NC}"
cd "$SCRIPT_DIR"

# Try to compile with retries (in case of network issues)
RETRY_COUNT=0
MAX_RETRIES=3
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    go build -o /usr/local/bin/anemone ./cmd/anemone 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓ Anemone server compiled${NC}"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        echo -e "${YELLOW}  Retry $RETRY_COUNT/$MAX_RETRIES...${NC}"
        sleep 2
    else
        echo -e "${RED}Error: Failed to compile Anemone server after $MAX_RETRIES attempts${NC}"
        echo -e "${YELLOW}Please check your internet connection and try again${NC}"
        exit 1
    fi
done

# Setup systemd service
echo ""
echo -e "${YELLOW}  Setting up systemd service...${NC}"

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
if [ "$DECRYPT_TOOL_AVAILABLE" = true ]; then
    echo -e "${GREEN}✓ SMB passwords restored from backup${NC}"
    echo -e "  Users can access SMB shares with their original passwords"
else
    echo -e "${YELLOW}⚠️  IMPORTANT - SMB Password Reset Required:${NC}"
    echo -e "  SMB passwords could not be restored from backup"
    echo -e "  All SMB users have been created with temporary password: ${RED}anemone123${NC}"
    echo -e "  ${RED}Administrators must reset SMB passwords via web interface${NC}"
fi
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Access web interface: ${BLUE}https://$(hostname -I | awk '{print $1}'):8443${NC}"
echo -e "  2. Login with restored admin credentials"
if [ "$DECRYPT_TOOL_AVAILABLE" != true ]; then
    echo -e "  3. ${RED}Reset SMB passwords for all users${NC}"
    echo -e "  4. Test SMB access with new passwords"
    echo -e "  5. Check logs: ${BLUE}sudo journalctl -u anemone -f${NC}"
else
    echo -e "  3. Test SMB access with restored passwords"
    echo -e "  4. Check logs: ${BLUE}sudo journalctl -u anemone -f${NC}"
fi
echo ""
if [ -d "$BACKUP_DIR" ]; then
    echo -e "${YELLOW}Backup of previous configuration saved to:${NC}"
    echo -e "  $BACKUP_DIR"
    echo ""
fi
