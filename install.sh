#!/bin/bash
set -e

# Anemone NAS - Automated Installer
# This script installs Anemone and its dependencies.
# Configuration is done via the web-based setup wizard.
#
# Usage:
#   sudo ./install.sh [options]
#
# Options:
#   --data-dir=PATH    Set data directory (default: /srv/anemone)
#   --user=USERNAME    Set service user (default: current user)
#   --help             Show this help message
#
# Examples:
#   sudo ./install.sh                           # Install with defaults
#   sudo ./install.sh --data-dir=/data/anemone  # Custom data directory
#   sudo ./install.sh --user=anemone            # Run as specific user

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
DATA_DIR="/srv/anemone"
INCOMING_DIR=""  # Empty = same as DATA_DIR
INSTALL_DIR="$(pwd)"
BINARY_NAME="anemone"
SERVICE_NAME="anemone"
SERVICE_USER="${SUDO_USER:-$USER}"
INSTALL_MODE="new"  # "new" or "repair"

# Parse command line arguments
parse_args() {
    for arg in "$@"; do
        case $arg in
            --data-dir=*)
                DATA_DIR="${arg#*=}"
                ;;
            --user=*)
                SERVICE_USER="${arg#*=}"
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $arg"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    echo "Anemone NAS - Automated Installer"
    echo ""
    echo "Usage: sudo ./install.sh [options]"
    echo ""
    echo "Options:"
    echo "  --data-dir=PATH    Set data directory (default: /srv/anemone)"
    echo "  --user=USERNAME    Set service user (default: current user)"
    echo "  --help             Show this help message"
    echo ""
    echo "After installation, complete setup via the web wizard at:"
    echo "  https://<server-ip>:8443"
}

ask_install_mode() {
    echo ""
    echo -e "${BLUE}=== Installation Mode ===${NC}"
    echo ""
    echo "  1) New installation (default)"
    echo "     Fresh install with setup wizard"
    echo ""
    echo "  2) Repair / Reinstall"
    echo "     Restore from existing Anemone data (after OS reinstall)"
    echo ""
    read -p "Choose [1-2] (default: 1): " choice

    case "$choice" in
        2)
            INSTALL_MODE="repair"
            ask_repair_paths
            ;;
        *)
            INSTALL_MODE="new"
            ;;
    esac
}

ask_repair_paths() {
    echo ""
    echo -e "${YELLOW}=== Repair Mode ===${NC}"
    echo ""

    # Ask for data directory
    read -p "Anemone data directory (e.g., /mnt/zfs or /srv/anemone): " input_data_dir
    if [ -z "$input_data_dir" ]; then
        log_error "Data directory is required"
        exit 1
    fi
    DATA_DIR="$input_data_dir"

    # Check if database exists
    if [ ! -f "$DATA_DIR/db/anemone.db" ]; then
        log_error "Database not found: $DATA_DIR/db/anemone.db"
        log_error "This directory does not contain a valid Anemone installation"
        exit 1
    fi

    log_info "Database found: $DATA_DIR/db/anemone.db"

    # Ask for incoming directory
    echo ""
    read -p "Incoming backups directory (press Enter for same as data): " input_incoming_dir
    if [ -n "$input_incoming_dir" ]; then
        INCOMING_DIR="$input_incoming_dir"
        if [ ! -d "$INCOMING_DIR" ]; then
            log_error "Incoming directory not found: $INCOMING_DIR"
            exit 1
        fi
    fi

    echo ""
    log_info "Data directory: $DATA_DIR"
    if [ -n "$INCOMING_DIR" ]; then
        log_info "Incoming directory: $INCOMING_DIR"
    else
        log_info "Incoming directory: $DATA_DIR/incoming (same as data)"
    fi
    echo ""
    read -p "Continue with repair? [Y/n] " confirm
    if [[ "$confirm" =~ ^[Nn] ]]; then
        echo "Aborted."
        exit 0
    fi
}

repair_installation() {
    log_step "Repairing existing installation..."

    # Install sqlite3 if not present
    if ! command -v sqlite3 &> /dev/null; then
        log_info "Installing sqlite3..."
        if [ "$PKG_MANAGER" = "dnf" ]; then
            dnf install -y sqlite
        elif [ "$PKG_MANAGER" = "apt" ]; then
            apt install -y sqlite3
        fi
    fi

    DB_PATH="$DATA_DIR/db/anemone.db"

    # Get shares directory from system_config
    SHARES_DIR=$(sqlite3 "$DB_PATH" "SELECT value FROM system_config WHERE key='shares_dir';" 2>/dev/null)
    if [ -z "$SHARES_DIR" ]; then
        SHARES_DIR="$DATA_DIR/shares"
    fi
    log_info "Shares directory: $SHARES_DIR"

    # Get incoming directory from system_config (or use provided/default)
    if [ -z "$INCOMING_DIR" ]; then
        INCOMING_DIR=$(sqlite3 "$DB_PATH" "SELECT value FROM system_config WHERE key='incoming_dir';" 2>/dev/null)
        if [ -z "$INCOMING_DIR" ]; then
            INCOMING_DIR="$DATA_DIR/incoming"
        fi
    fi
    log_info "Incoming directory: $INCOMING_DIR"

    # Get list of users from database
    log_info "Reading users from database..."
    USERS=$(sqlite3 "$DB_PATH" "SELECT username FROM users WHERE activated_at IS NOT NULL;" 2>/dev/null)

    if [ -z "$USERS" ]; then
        log_warn "No activated users found in database"
    else
        log_info "Found users: $(echo $USERS | tr '\n' ' ')"
    fi

    # Create system users and set SMB passwords
    for username in $USERS; do
        log_info "Creating system user: $username"

        # Create system user (no login shell, no home)
        if ! id "$username" &>/dev/null; then
            useradd -M -s /usr/sbin/nologin "$username" 2>/dev/null || \
            useradd -M -s /sbin/nologin "$username" 2>/dev/null || \
            log_warn "Failed to create system user: $username"
        else
            log_info "System user already exists: $username"
        fi

        # Get encrypted password from database
        PASSWORD_ENC=$(sqlite3 "$DB_PATH" "SELECT hex(password_encrypted) FROM users WHERE username='$username';" 2>/dev/null)
        MASTER_KEY=$(sqlite3 "$DB_PATH" "SELECT value FROM system_config WHERE key='master_key';" 2>/dev/null)

        if [ -n "$PASSWORD_ENC" ] && [ -n "$MASTER_KEY" ]; then
            # We'll set SMB password later via the running service
            # For now, just note that user needs password reset or service will handle it
            log_info "User $username will need SMB password configured"
        fi
    done

    # Fix ownership on share directories
    log_info "Fixing permissions on shares..."
    for username in $USERS; do
        USER_SHARE_DIR="$SHARES_DIR/$username"
        if [ -d "$USER_SHARE_DIR" ]; then
            chown -R "$username:$username" "$USER_SHARE_DIR" 2>/dev/null || \
            log_warn "Failed to set ownership on $USER_SHARE_DIR"
            chmod 700 "$USER_SHARE_DIR" 2>/dev/null
            log_info "Fixed permissions: $USER_SHARE_DIR"
        fi
    done

    # Fix ownership on incoming directories
    log_info "Fixing permissions on incoming..."
    if [ -d "$INCOMING_DIR" ]; then
        chown -R "$SERVICE_USER:$SERVICE_USER" "$INCOMING_DIR" 2>/dev/null
        chmod 755 "$INCOMING_DIR" 2>/dev/null
    fi

    # Remove any .needs-setup marker file
    rm -f "$DATA_DIR/.needs-setup" 2>/dev/null
    rm -f "$DATA_DIR/.setup-state.json" 2>/dev/null

    log_info "Repair preparation complete"
}

regenerate_samba_config() {
    log_info "Regenerating Samba configuration..."

    DB_PATH="$DATA_DIR/db/anemone.db"
    SMB_DIR="$DATA_DIR/smb"
    SMB_CONF="$SMB_DIR/smb.conf"

    # Create smb directory
    mkdir -p "$SMB_DIR"

    # Get server name
    SERVER_NAME=$(sqlite3 "$DB_PATH" "SELECT value FROM system_config WHERE key='server_name';" 2>/dev/null)
    if [ -z "$SERVER_NAME" ]; then
        SERVER_NAME="Anemone"
    fi

    # Get shares directory
    SHARES_DIR=$(sqlite3 "$DB_PATH" "SELECT value FROM system_config WHERE key='shares_dir';" 2>/dev/null)
    if [ -z "$SHARES_DIR" ]; then
        SHARES_DIR="$DATA_DIR/shares"
    fi

    # Generate smb.conf header
    cat > "$SMB_CONF" <<EOF
# Anemone NAS - Samba Configuration
# Auto-generated by install.sh repair mode
# Do not edit manually - changes will be overwritten

[global]
workgroup = WORKGROUP
server string = $SERVER_NAME
netbios name = $SERVER_NAME
security = user
map to guest = never
passdb backend = tdbsam
unix password sync = no
guest ok = no
guest account = nobody
log file = /var/log/samba/log.%m
max log size = 1000
logging = file
panic action = /usr/share/samba/panic-action %d
server role = standalone server
obey pam restrictions = yes
unix extensions = yes
wide links = no
create mask = 0600
directory mask = 0700
force create mode = 0600
force directory mode = 0700
hide dot files = no
veto files = /._*/.DS_Store/.Thumbs.db/.desktop.ini/
delete veto files = yes
inherit permissions = no
nt acl support = no
acl allow execute always = no
vfs objects = recycle
recycle:repository = .trash/%U
recycle:keeptree = yes
recycle:touch = yes
recycle:versions = yes
recycle:maxsize = 0
dfree command = /usr/local/bin/anemone-dfree

EOF

    # Get all shares from database and add them
    sqlite3 "$DB_PATH" "SELECT s.name, s.path, u.username FROM shares s JOIN users u ON s.user_id = u.id WHERE u.activated_at IS NOT NULL;" 2>/dev/null | while IFS='|' read -r name path username; do
        # Use the path directly from the database
        cat >> "$SMB_CONF" <<EOF
[$name]
path = $path
valid users = $username
read only = no
browseable = yes
create mask = 0600
directory mask = 0700
force user = $username
force group = $username

EOF
        log_info "Added share: $name -> $path"
    done

    # Copy to /etc/samba/smb.conf
    cp "$SMB_CONF" /etc/samba/smb.conf 2>/dev/null || log_warn "Could not copy smb.conf to /etc/samba/"

    log_info "Samba configuration generated: $SMB_CONF"
}

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

detect_distro() {
    if [ -f /etc/fedora-release ]; then
        DISTRO="fedora"
        SMB_SERVICE="smb"
        PKG_MANAGER="dnf"
    elif [ -f /etc/redhat-release ]; then
        DISTRO="rhel"
        SMB_SERVICE="smb"
        PKG_MANAGER="dnf"
    elif [ -f /etc/debian_version ]; then
        DISTRO="debian"
        SMB_SERVICE="smbd"
        PKG_MANAGER="apt"
    else
        log_error "Unsupported distribution. Supported: Fedora, RHEL, Debian, Ubuntu"
        exit 1
    fi
    log_info "Detected distribution: $DISTRO"
}

install_gcc() {
    if command -v gcc &> /dev/null; then
        log_info "GCC already installed"
        return
    fi

    log_info "Installing GCC (required for CGO)..."
    if [ "$PKG_MANAGER" = "dnf" ]; then
        dnf install -y gcc
    elif [ "$PKG_MANAGER" = "apt" ]; then
        apt update
        apt install -y gcc build-essential
    fi

    if ! command -v gcc &> /dev/null; then
        log_error "Failed to install GCC"
        exit 1
    fi
    log_info "GCC installed"
}

install_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go already installed: $GO_VERSION"
        return
    fi

    log_info "Installing Go..."

    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)      GO_ARCH="amd64" ;;
        aarch64|arm64) GO_ARCH="arm64" ;;
        armv6l)      GO_ARCH="armv6l" ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    # Get latest Go version
    LATEST_GO=$(curl -sL https://go.dev/VERSION?m=text | head -n1)
    if [ -z "$LATEST_GO" ]; then
        log_error "Failed to fetch latest Go version"
        exit 1
    fi

    log_info "Downloading Go $LATEST_GO..."
    GO_TARBALL="$LATEST_GO.linux-$GO_ARCH.tar.gz"
    GO_URL="https://go.dev/dl/$GO_TARBALL"

    cd /tmp
    curl -LO "$GO_URL"

    if [ ! -f "$GO_TARBALL" ]; then
        log_error "Failed to download Go"
        exit 1
    fi

    # Remove old installation if exists
    [ -d /usr/local/go ] && rm -rf /usr/local/go

    # Extract and configure
    tar -C /usr/local -xzf "$GO_TARBALL"
    rm "$GO_TARBALL"

    # Configure PATH
    cat > /etc/profile.d/go.sh <<'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
EOF
    chmod +x /etc/profile.d/go.sh

    # Apply to current session
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin

    if ! command -v go &> /dev/null; then
        log_error "Failed to install Go"
        exit 1
    fi
    log_info "Go installed"
    cd "$INSTALL_DIR"
}

install_git() {
    if command -v git &> /dev/null; then
        log_info "Git already installed"
        return
    fi

    log_info "Installing Git..."
    if [ "$PKG_MANAGER" = "dnf" ]; then
        dnf install -y git
    elif [ "$PKG_MANAGER" = "apt" ]; then
        apt update
        apt install -y git
    fi

    if ! command -v git &> /dev/null; then
        log_error "Failed to install Git"
        exit 1
    fi
    log_info "Git installed"
}

install_samba() {
    if command -v smbd &> /dev/null; then
        log_info "Samba already installed"
    else
        log_info "Installing Samba..."
        if [ "$PKG_MANAGER" = "dnf" ]; then
            dnf install -y samba
        elif [ "$PKG_MANAGER" = "apt" ]; then
            apt update
            apt install -y samba
        fi
        log_info "Samba installed"
    fi

    # Enable and start Samba service
    log_info "Enabling and starting Samba service..."
    systemctl enable --now "$SMB_SERVICE"
    log_info "Samba service started"
}

install_storage_tools() {
    log_info "Installing storage tools..."

    # smartmontools for disk health
    if ! command -v smartctl &> /dev/null; then
        if [ "$PKG_MANAGER" = "dnf" ]; then
            dnf install -y smartmontools
        elif [ "$PKG_MANAGER" = "apt" ]; then
            apt install -y smartmontools
        fi
        log_info "smartmontools installed"
    fi

    # ZFS utilities (optional, non-fatal if unavailable)
    if ! command -v zpool &> /dev/null; then
        log_info "Checking ZFS availability..."
        if [ "$PKG_MANAGER" = "dnf" ]; then
            if dnf list available zfs 2>/dev/null | grep -q zfs; then
                dnf install -y zfs && log_info "ZFS installed" || log_warn "ZFS installation failed (optional)"
            else
                log_warn "ZFS not available in repositories (optional)"
            fi
        elif [ "$PKG_MANAGER" = "apt" ]; then
            if apt-cache show zfsutils-linux &>/dev/null; then
                apt install -y zfsutils-linux && log_info "ZFS installed" || log_warn "ZFS installation failed (optional)"
            else
                log_warn "ZFS not available (optional)"
            fi
        fi
    else
        log_info "ZFS already installed"
    fi

    # util-linux for lsblk
    if ! command -v lsblk &> /dev/null; then
        if [ "$PKG_MANAGER" = "dnf" ]; then
            dnf install -y util-linux
        elif [ "$PKG_MANAGER" = "apt" ]; then
            apt install -y util-linux
        fi
    fi

    # Filesystem tools for USB formatting (FAT32, exFAT)
    if ! command -v mkfs.vfat &> /dev/null; then
        log_info "Installing FAT32 tools (dosfstools)..."
        if [ "$PKG_MANAGER" = "dnf" ]; then
            dnf install -y dosfstools
        elif [ "$PKG_MANAGER" = "apt" ]; then
            apt install -y dosfstools
        fi
    fi

    if ! command -v mkfs.exfat &> /dev/null; then
        log_info "Installing exFAT tools (exfatprogs)..."
        if [ "$PKG_MANAGER" = "dnf" ]; then
            dnf install -y exfatprogs
        elif [ "$PKG_MANAGER" = "apt" ]; then
            apt install -y exfatprogs
        fi
    fi

    log_info "Storage tools ready"
}

build_binary() {
    log_info "Building Anemone..."

    cd "$INSTALL_DIR"

    # Build as non-root user if possible
    if [ -n "$SERVICE_USER" ] && [ "$SERVICE_USER" != "root" ]; then
        su - "$SERVICE_USER" -c "cd '$INSTALL_DIR' && CGO_ENABLED=1 go build -o $BINARY_NAME ./cmd/anemone"
    else
        CGO_ENABLED=1 go build -o $BINARY_NAME ./cmd/anemone
    fi

    if [ ! -f "$BINARY_NAME" ]; then
        log_error "Failed to build binary"
        exit 1
    fi

    # Build dfree helper for Samba quota display
    if [ -d "cmd/anemone-dfree" ]; then
        log_info "Building dfree helper..."
        if [ -n "$SERVICE_USER" ] && [ "$SERVICE_USER" != "root" ]; then
            su - "$SERVICE_USER" -c "cd '$INSTALL_DIR' && CGO_ENABLED=1 go build -o anemone-dfree ./cmd/anemone-dfree" 2>/dev/null || true
        else
            CGO_ENABLED=1 go build -o anemone-dfree ./cmd/anemone-dfree 2>/dev/null || true
        fi
    fi

    log_info "Build complete"
}

create_data_directory() {
    log_info "Creating data directory: $DATA_DIR"

    mkdir -p "$DATA_DIR"
    chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
    chmod 755 "$DATA_DIR"

    # Create /etc/anemone for config file (owned by service user for setup wizard)
    mkdir -p /etc/anemone
    chown "$SERVICE_USER:$SERVICE_USER" /etc/anemone
    chmod 755 /etc/anemone

    log_info "Data directory ready"
}

configure_sudoers() {
    log_info "Configuring sudoers..."

    SUDOERS_FILE="/etc/sudoers.d/anemone"

    cat > "$SUDOERS_FILE" <<EOF
# Anemone NAS - Sudo Permissions
# Generated by install.sh
# Data directory: $DATA_DIR

# Service management
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart anemone
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/systemctl restart anemone
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload $SMB_SERVICE
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload $SMB_SERVICE.service

# User management for Samba
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/useradd -M -s /usr/sbin/nologin *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/userdel *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/smbpasswd

# File operations - restricted to data directory
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/chown * $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/chown -R * $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/chmod * $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/chmod -R * $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/rm -rf $DATA_DIR/shares/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/rm -rf $DATA_DIR/backups/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/rmdir $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mv $DATA_DIR/* $DATA_DIR/*

# SMB configuration
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/cp $DATA_DIR/smb/smb.conf /etc/samba/smb.conf

# SELinux (RHEL/Fedora)
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/semanage fcontext -a -t samba_share_t $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/restorecon -Rv $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/setsebool -P samba_enable_home_dirs on
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/setsebool -P samba_export_all_rw on

# Btrfs quota management
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/btrfs subvolume create $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/btrfs subvolume delete $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/btrfs subvolume show $DATA_DIR/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/btrfs qgroup *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/btrfs quota enable *

# Storage management (SMART, ZFS)
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/smartctl *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/zpool *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/zpool *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/zfs *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/zfs *

# Disk formatting (for setup wizard and storage management)
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/mkfs.ext4 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/mkfs.ext4 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/mkfs.xfs *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/mkfs.xfs *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/mkfs.vfat *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/mkfs.vfat *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/mkfs.exfat *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/mkfs.exfat *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mkfs.exfat *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/wipefs *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/wipefs *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/parted *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/parted *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/partprobe *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/partprobe *
$SERVICE_USER ALL=(ALL) NOPASSWD: /sbin/blockdev *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/sbin/blockdev *

# Disk mounting (restricted to /mnt/ and /media/)
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mount /dev/sd* /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mount /dev/sd* /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mount /dev/nvme* /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mount /dev/nvme* /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/mount /dev/sd* /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/mount /dev/sd* /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/mount /dev/nvme* /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/mount /dev/nvme* /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/umount /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/umount /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/umount /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/umount /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/eject /dev/sd*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/eject /dev/nvme*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/eject /dev/sd*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/eject /dev/nvme*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /media/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/mkdir -p /mnt/*
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/mkdir -p /media/*

# Disk wiping (zero only, restricted to device paths)
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/dd if=/dev/zero of=/dev/sd* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/dd if=/dev/zero of=/dev/nvme* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/dd if=/dev/zero of=/dev/vd* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/dd if=/dev/zero of=/dev/loop* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/dd if=/dev/zero of=/dev/sd* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/dd if=/dev/zero of=/dev/nvme* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/dd if=/dev/zero of=/dev/vd* bs=1M count=1 *
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/dd if=/dev/zero of=/dev/loop* bs=1M count=1 *

# Setup wizard - update sudoers when custom path is used
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/cat /etc/sudoers.d/anemone
$SERVICE_USER ALL=(ALL) NOPASSWD: /bin/cat /etc/sudoers.d/anemone
$SERVICE_USER ALL=(ALL) NOPASSWD: /usr/bin/tee /etc/sudoers.d/anemone
EOF

    chmod 440 "$SUDOERS_FILE"
    log_info "Sudoers configured"
}

configure_selinux() {
    if [ "$DISTRO" != "fedora" ] && [ "$DISTRO" != "rhel" ]; then
        return
    fi

    if ! command -v getenforce &> /dev/null; then
        return
    fi

    if [ "$(getenforce)" = "Disabled" ]; then
        log_info "SELinux is disabled"
        return
    fi

    log_info "Configuring SELinux..."
    semanage fcontext -a -t samba_share_t "$DATA_DIR/shares(/.*)?" 2>/dev/null || true
    setsebool -P samba_export_all_rw on 2>/dev/null || true
    log_info "SELinux configured"
}

configure_firewall() {
    log_info "Configuring firewall..."

    if command -v firewall-cmd &> /dev/null; then
        if systemctl is-active --quiet firewalld; then
            firewall-cmd --permanent --add-service=samba 2>/dev/null || true
            firewall-cmd --permanent --add-port=8443/tcp 2>/dev/null || true
            firewall-cmd --reload 2>/dev/null || true
            log_info "FirewallD configured"
        else
            log_warn "FirewallD is not running"
        fi
    elif command -v ufw &> /dev/null; then
        ufw allow Samba 2>/dev/null || true
        ufw allow 8443/tcp 2>/dev/null || true
        log_info "UFW configured"
    else
        log_warn "No firewall detected. Ensure ports 139, 445 (SMB) and 8443 (HTTPS) are open"
    fi
}

create_systemd_service() {
    log_info "Creating systemd service..."

    SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Anemone NAS Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=-/etc/anemone/anemone.env
Environment="ENABLE_HTTPS=true"
Environment="HTTPS_PORT=8443"
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_info "Systemd service created"
}

enable_services() {
    log_info "Enabling services..."

    # Enable and start Samba
    systemctl enable "$SMB_SERVICE"
    systemctl start "$SMB_SERVICE" || log_warn "Samba failed to start"

    # Enable and start Anemone
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    sleep 2

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "Anemone service is running"
    else
        log_error "Anemone service failed to start"
        log_error "Check logs: journalctl -u $SERVICE_NAME -n 50"
        exit 1
    fi
}

show_completion_message() {
    # Get server IP
    SERVER_IP=$(hostname -I | awk '{print $1}')

    echo ""
    echo -e "${GREEN}================================================================${NC}"
    echo -e "${GREEN}       Anemone NAS - Installation Complete                     ${NC}"
    echo -e "${GREEN}================================================================${NC}"
    echo ""
    echo -e "  ${BLUE}Next step: Complete setup via the web wizard${NC}"
    echo ""
    echo -e "  Open your browser and go to:"
    echo -e "  ${YELLOW}https://${SERVER_IP}:8443${NC}"
    echo ""
    echo -e "  The setup wizard will guide you through:"
    echo -e "    1. Storage configuration (default path, ZFS, or custom)"
    echo -e "    2. Backup storage location"
    echo -e "    3. Admin account creation"
    echo ""
    echo -e "  ${BLUE}Useful commands:${NC}"
    echo -e "    Status:   systemctl status anemone"
    echo -e "    Logs:     journalctl -u anemone -f"
    echo -e "    Restart:  sudo systemctl restart anemone"
    echo ""
    echo -e "  ${BLUE}Configuration:${NC}"
    echo -e "    Data directory: $DATA_DIR"
    echo -e "    Service user:   $SERVICE_USER"
    echo -e "    Install path:   $INSTALL_DIR"
    echo ""
}

show_repair_completion_message() {
    # Get server IP
    SERVER_IP=$(hostname -I | awk '{print $1}')

    echo ""
    echo -e "${GREEN}================================================================${NC}"
    echo -e "${GREEN}       Anemone NAS - Repair Complete                           ${NC}"
    echo -e "${GREEN}================================================================${NC}"
    echo ""
    echo -e "  ${BLUE}Your installation has been restored!${NC}"
    echo ""
    echo -e "  Open your browser and log in:"
    echo -e "  ${YELLOW}https://${SERVER_IP}:8443${NC}"
    echo ""
    echo -e "  ${YELLOW}Note about SMB passwords:${NC}"
    echo -e "  Users may need to reset their passwords to restore SMB access."
    echo -e "  They can do this from their profile page after logging in."
    echo ""
    echo -e "  ${BLUE}Useful commands:${NC}"
    echo -e "    Status:   systemctl status anemone"
    echo -e "    Logs:     journalctl -u anemone -f"
    echo -e "    Restart:  sudo systemctl restart anemone"
    echo ""
    echo -e "  ${BLUE}Configuration:${NC}"
    echo -e "    Data directory: $DATA_DIR"
    echo -e "    Service user:   $SERVICE_USER"
    echo -e "    Install path:   $INSTALL_DIR"
    echo ""
}

# Main installation flow
main() {
    echo -e "${GREEN}Anemone NAS - Automated Installer${NC}"
    echo ""

    parse_args "$@"
    check_root
    detect_distro

    # Ask for installation mode
    ask_install_mode

    if [ "$INSTALL_MODE" = "repair" ]; then
        # Repair mode - fewer steps
        log_step "1/6 Installing build tools..."
        install_gcc
        install_go
        install_git

        log_step "2/6 Installing Samba..."
        install_samba

        log_step "3/6 Building Anemone..."
        build_binary

        log_step "4/6 Repairing installation..."
        repair_installation
        regenerate_samba_config

        log_step "5/6 Configuring permissions..."
        configure_sudoers
        configure_selinux
        configure_firewall

        log_step "6/6 Starting services..."
        create_systemd_service
        enable_services

        # Reload Samba with new config
        systemctl reload "$SMB_SERVICE" 2>/dev/null || systemctl restart "$SMB_SERVICE"

        show_repair_completion_message
    else
        # New installation mode
        log_step "1/8 Installing build tools..."
        install_gcc
        install_go
        install_git

        log_step "2/8 Installing Samba..."
        install_samba

        log_step "3/8 Installing storage tools..."
        install_storage_tools

        log_step "4/8 Building Anemone..."
        build_binary

        log_step "5/8 Creating data directory..."
        create_data_directory

        log_step "6/8 Configuring permissions..."
        configure_sudoers
        configure_selinux

        log_step "7/8 Configuring firewall..."
        configure_firewall

        log_step "8/8 Starting services..."
        create_systemd_service
        enable_services

        show_completion_message
    fi
}

# Run installation
main "$@"
