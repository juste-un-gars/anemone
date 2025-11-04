#!/bin/bash
set -e

# 🪸 Anemone NAS - Automated Installer
# This script installs and configures Anemone on your system
#
# Usage:
#   sudo ./install.sh [language]
#
# Parameters:
#   language: "fr" (French) or "en" (English) - defaults to "fr" if not specified
#
# Examples:
#   sudo ./install.sh fr      # Install with French language
#   sudo ./install.sh en      # Install with English language
#   sudo ./install.sh         # Install with default French language

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DATA_DIR="/srv/anemone"
INSTALL_DIR="$(pwd)"
BINARY_NAME="anemone"
SERVICE_NAME="anemone"
CURRENT_USER="${SUDO_USER:-$USER}"
LANGUAGE="${1:-fr}"  # Parse language parameter, default to French

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

validate_language() {
    if [ "$LANGUAGE" != "fr" ] && [ "$LANGUAGE" != "en" ]; then
        log_error "Invalid language: $LANGUAGE. Use 'fr' (French) or 'en' (English)"
        exit 1
    fi
    log_info "Language set to: $LANGUAGE"
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
        log_error "Unsupported distribution"
        exit 1
    fi
    log_info "Detected distribution: $DISTRO"
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.21+ from https://go.dev/doc/install"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $GO_VERSION"

    # Check git
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed"
        exit 1
    fi

    log_info "All prerequisites met"
}

install_samba() {
    log_info "Checking Samba installation..."

    if command -v smbd &> /dev/null || command -v smb &> /dev/null; then
        log_info "Samba is already installed"
        return
    fi

    log_info "Installing Samba..."
    if [ "$PKG_MANAGER" = "dnf" ]; then
        dnf install -y samba
    elif [ "$PKG_MANAGER" = "apt" ]; then
        apt update
        apt install -y samba
    fi

    log_info "Samba installed successfully"
}

build_binary() {
    log_info "Building Anemone binary..."

    cd "$INSTALL_DIR"
    su - "$CURRENT_USER" -c "cd '$INSTALL_DIR' && CGO_ENABLED=1 go build -o $BINARY_NAME ./cmd/anemone"

    if [ ! -f "$BINARY_NAME" ]; then
        log_error "Failed to build binary"
        exit 1
    fi

    log_info "Binary built successfully"
}

create_data_directory() {
    log_info "Creating data directory..."

    if [ ! -d "$DATA_DIR" ]; then
        mkdir -p "$DATA_DIR"
        log_info "Created $DATA_DIR"
    else
        log_warn "$DATA_DIR already exists"
    fi

    # Set ownership
    chown -R "$CURRENT_USER:$CURRENT_USER" "$DATA_DIR"
    chmod 755 "$DATA_DIR"

    log_info "Data directory configured"
}

configure_sudoers() {
    log_info "Configuring sudoers for SMB management..."

    SUDOERS_FILE="/etc/sudoers.d/anemone-smb"

    cat > "$SUDOERS_FILE" <<EOF
# Anemone NAS - SMB Management Permissions
# Allow user to reload Samba service and manage SMB users
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload $SMB_SERVICE
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload $SMB_SERVICE.service
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/sbin/useradd -M -s /usr/sbin/nologin *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/sbin/userdel *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/smbpasswd
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/chown -R *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/chmod *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/cp * /etc/samba/smb.conf
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/mv *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/rm *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/rmdir *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/sbin/semanage fcontext *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/sbin/restorecon *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/sbin/setsebool *
$CURRENT_USER ALL=(ALL) NOPASSWD: /usr/bin/btrfs *
EOF

    chmod 440 "$SUDOERS_FILE"
    log_info "Sudoers configured"
}

configure_selinux() {
    if [ "$DISTRO" != "fedora" ] && [ "$DISTRO" != "rhel" ]; then
        log_info "SELinux not applicable for this distribution"
        return
    fi

    if ! command -v getenforce &> /dev/null; then
        log_info "SELinux not installed"
        return
    fi

    if [ "$(getenforce)" = "Disabled" ]; then
        log_info "SELinux is disabled"
        return
    fi

    log_info "Configuring SELinux for Samba..."

    # Set context for shares directory
    semanage fcontext -a -t samba_share_t "$DATA_DIR/shares(/.*)?" 2>/dev/null || true

    # Enable Samba export
    setsebool -P samba_export_all_rw on

    log_info "SELinux configured"
}

configure_firewall() {
    log_info "Configuring firewall..."

    if command -v firewall-cmd &> /dev/null; then
        # FirewallD (Fedora/RHEL)
        if systemctl is-active --quiet firewalld; then
            firewall-cmd --permanent --add-service=samba
            firewall-cmd --permanent --add-port=8443/tcp
            firewall-cmd --reload
            log_info "FirewallD configured"
        else
            log_warn "FirewallD is not running"
        fi
    elif command -v ufw &> /dev/null; then
        # UFW (Ubuntu/Debian)
        ufw allow Samba
        ufw allow 8443/tcp
        log_info "UFW configured"
    else
        log_warn "No firewall detected. Please configure manually:"
        log_warn "  - Allow ports: 139, 445 (SMB)"
        log_warn "  - Allow port: 8443 (HTTPS)"
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
User=$CURRENT_USER
Group=$CURRENT_USER
WorkingDirectory=$INSTALL_DIR
Environment="ANEMONE_DATA_DIR=$DATA_DIR"
Environment="ENABLE_HTTPS=true"
Environment="HTTPS_PORT=8443"
Environment="LANGUAGE=$LANGUAGE"
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_info "Systemd service created"
}

enable_samba_service() {
    log_info "Enabling Samba service..."

    systemctl enable "$SMB_SERVICE"
    systemctl start "$SMB_SERVICE"

    if systemctl is-active --quiet "$SMB_SERVICE"; then
        log_info "Samba service is running"
    else
        log_warn "Samba service failed to start. Check logs: journalctl -u $SMB_SERVICE"
    fi
}

start_anemone_service() {
    log_info "Starting Anemone service..."

    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    sleep 2

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "Anemone service is running"
    else
        log_error "Anemone service failed to start. Check logs: journalctl -u $SERVICE_NAME"
        exit 1
    fi
}

show_completion_message() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  🪸 Anemone Installation Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "1. Access the web interface:"
    echo -e "   ${YELLOW}https://$(hostname -I | awk '{print $1}'):8443${NC}"
    echo ""
    echo "2. Complete initial setup:"
    echo "   - Choose language (FR/EN)"
    echo "   - Set NAS name and timezone"
    echo "   - Create admin user"
    echo ""
    echo "3. Useful commands:"
    echo "   - Check status: systemctl status anemone"
    echo "   - View logs:    journalctl -u anemone -f"
    echo "   - Restart:      systemctl restart anemone"
    echo ""
    echo "4. SMB shares will be created automatically when users are activated"
    echo ""
    echo "📚 Documentation: $INSTALL_DIR/README.md"
    echo ""
}

# Main installation flow
main() {
    echo -e "${GREEN}🪸 Anemone NAS - Automated Installer${NC}"
    echo ""

    check_root
    validate_language
    detect_distro
    check_prerequisites
    install_samba
    build_binary
    create_data_directory
    configure_sudoers
    configure_selinux
    configure_firewall
    create_systemd_service
    enable_samba_service
    start_anemone_service
    show_completion_message
}

# Run installation
main
